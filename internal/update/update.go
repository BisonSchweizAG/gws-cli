// Package update implements self-update for gws from GitHub releases.
//
// It relies only on the standard library and on the release layout produced by
// .goreleaser.yml:
//
//	gws_<version>_<os>_<arch>.tar.gz   (.zip on Windows), containing gws / gws.exe
//	checksums.txt                       ("<sha256>  <archive name>" per line)
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	binaryName    = "gws"
	checksumsName = "checksums.txt"
	maxBinarySize = 512 << 20 // sanity limit for the extracted binary
	maxTextSize   = 1 << 20   // sanity limit for API / checksum responses
)

var (
	// ErrUnknownVersion is returned when the running binary has no release version
	// (e.g. a `go build` or snapshot build).
	ErrUnknownVersion = errors.New("current version is not a release version")
	// ErrNoRelease is returned when no published, stable release exists.
	ErrNoRelease = errors.New("no published release found")
	// ErrNoAsset is returned when the release has no archive for this OS/arch.
	ErrNoAsset = errors.New("no release asset for this platform")
)

// Updater checks GitHub for new releases and replaces the running executable.
// Use New for production defaults; the fields are exported to make tests easy.
type Updater struct {
	Owner   string
	Repo    string
	APIBase string
	Client  *http.Client
	GOOS    string
	GOARCH  string
	// ExePath overrides the executable to replace (defaults to os.Executable()).
	ExePath string
}

// New returns an Updater for BisonSchweizAG/gws-cli on github.com.
func New() *Updater {
	return &Updater{
		Owner:   "BisonSchweizAG",
		Repo:    "gws-cli",
		APIBase: "https://api.github.com",
		Client:  &http.Client{Timeout: 5 * time.Minute},
		GOOS:    runtime.GOOS,
		GOARCH:  runtime.GOARCH,
	}
}

// Release describes the newest stable release usable on this platform.
type Release struct {
	Tag          string // e.g. "v1.2.3"
	Archive      string // e.g. "gws_1.2.3_linux_amd64.tar.gz"
	archiveURL   string
	checksumsURL string
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

// Latest returns the highest stable (non-draft, non-prerelease) release that
// has an archive for the configured OS/arch and a checksums.txt.
func (u *Updater) Latest(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", u.APIBase, u.Owner, u.Repo)

	var releases []ghRelease
	if err := u.getJSON(ctx, url, &releases); err != nil {
		return nil, err
	}

	var (
		best    *ghRelease
		bestVer [3]int
	)
	for i := range releases {
		r := &releases[i]
		if r.Draft || r.Prerelease {
			continue
		}
		v, ok := parseSemver(r.TagName)
		if !ok {
			continue
		}
		if best == nil || compare(v, bestVer) > 0 {
			best, bestVer = r, v
		}
	}
	if best == nil {
		return nil, ErrNoRelease
	}

	ext := "tar.gz"
	if u.GOOS == "windows" {
		ext = "zip"
	}
	// goreleaser's {{ .Version }} is the tag without the leading "v".
	archive := fmt.Sprintf("%s_%d.%d.%d_%s_%s.%s",
		binaryName, bestVer[0], bestVer[1], bestVer[2], u.GOOS, u.GOARCH, ext)

	rel := &Release{Tag: best.TagName, Archive: archive}
	for _, a := range best.Assets {
		switch a.Name {
		case archive:
			rel.archiveURL = a.URL
		case checksumsName:
			rel.checksumsURL = a.URL
		}
	}
	if rel.archiveURL == "" {
		return nil, fmt.Errorf("%w: %s/%s (%s)", ErrNoAsset, u.GOOS, u.GOARCH, best.TagName)
	}
	if rel.checksumsURL == "" {
		return nil, fmt.Errorf("release %s has no %s", best.TagName, checksumsName)
	}
	return rel, nil
}

// IsNewer reports whether candidate is a higher version than current.
// It returns ErrUnknownVersion if current is not a plain vMAJOR.MINOR.PATCH.
func IsNewer(current, candidate string) (bool, error) {
	c, ok := parseSemver(current)
	if !ok {
		return false, fmt.Errorf("%w: %q", ErrUnknownVersion, current)
	}
	n, ok := parseSemver(candidate)
	if !ok {
		return false, fmt.Errorf("invalid version %q", candidate)
	}
	return compare(n, c) > 0, nil
}

// Apply downloads the release archive, verifies its SHA-256 against
// checksums.txt, extracts the binary and atomically swaps it in for the
// running executable. On failure the previous binary is restored.
func (u *Updater) Apply(ctx context.Context, rel *Release) error {
	exe, err := u.executable()
	if err != nil {
		return err
	}
	info, err := os.Stat(exe)
	if err != nil {
		return err
	}

	want, err := u.expectedChecksum(ctx, rel)
	if err != nil {
		return err
	}

	archivePath, got, err := u.downloadToTemp(ctx, rel.archiveURL)
	if err != nil {
		return err
	}
	defer removeQuietly(archivePath)

	if got != want {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", rel.Archive, want, got)
	}

	binName := binaryName
	if u.GOOS == "windows" {
		binName += ".exe"
	}
	dir, base := filepath.Dir(exe), filepath.Base(exe)
	newPath := filepath.Join(dir, "."+base+".new")
	oldPath := filepath.Join(dir, "."+base+".old")

	defer removeQuietly(newPath) // no-op after a successful swap
	if err := extractBinary(archivePath, rel.Archive, binName, newPath, info.Mode().Perm()); err != nil {
		return fmt.Errorf("extract %s: %w", rel.Archive, err)
	}
	return replaceExecutable(exe, newPath, oldPath)
}

func (u *Updater) executable() (string, error) {
	p := u.ExePath
	if p == "" {
		var err error
		if p, err = os.Executable(); err != nil {
			return "", fmt.Errorf("locate executable: %w", err)
		}
	}
	return filepath.EvalSymlinks(p)
}

func (u *Updater) expectedChecksum(ctx context.Context, rel *Release) (string, error) {
	resp, err := u.get(ctx, rel.checksumsURL, "")
	if err != nil {
		return "", err
	}
	defer closeQuietly(resp.Body)
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxTextSize))
	if err != nil {
		return "", err
	}
	return findChecksum(data, rel.Archive)
}

// findChecksum parses sha256sum-style output: "<hex>  <filename>" per line.
func findChecksum(data []byte, name string) (string, error) {
	for line := range strings.SplitSeq(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("no checksum for %s in %s", name, checksumsName)
}

func (u *Updater) downloadToTemp(ctx context.Context, url string) (tmpPath, checksum string, err error) {
	resp, err := u.get(ctx, url, "application/octet-stream")
	if err != nil {
		return "", "", err
	}
	defer closeQuietly(resp.Body)

	f, err := os.CreateTemp("", "gws-update-*")
	if err != nil {
		return "", "", err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", "", fmt.Errorf("download: %w", err)
	}
	return f.Name(), hex.EncodeToString(h.Sum(nil)), nil
}

func (u *Updater) getJSON(ctx context.Context, url string, v any) error {
	resp, err := u.get(ctx, url, "application/vnd.github+json")
	if err != nil {
		return err
	}
	defer closeQuietly(resp.Body)
	return json.NewDecoder(io.LimitReader(resp.Body, maxTextSize)).Decode(v)
}

func (u *Updater) get(ctx context.Context, url, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gws-cli-updater")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if strings.HasPrefix(url, u.APIBase) {
		req.Header.Set("X-Github-Api-Version", "2022-11-28")
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("GET %s: HTTP %d (GitHub rate limit? try again later)", url, resp.StatusCode)
		}
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return resp, nil
}

// extractBinary writes the file named binName from the archive to dst.
// Only the base name of archive entries is compared and the entry name is
// never used as a path, so a malicious archive cannot write elsewhere.
func extractBinary(archivePath, archiveName, binName, dst string, mode os.FileMode) error {
	if strings.HasSuffix(archiveName, ".zip") {
		return extractZip(archivePath, binName, dst, mode)
	}
	return extractTarGz(archivePath, binName, dst, mode)
}

func extractZip(archivePath, binName, dst string, mode os.FileMode) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer closeQuietly(zr)

	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() || path.Base(zf.Name) != binName {
			continue
		}
		return writeZipEntry(zf, dst, mode)
	}
	return fmt.Errorf("%s not found in archive", binName)
}

func writeZipEntry(zf *zip.File, dst string, mode os.FileMode) error {
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer closeQuietly(rc)
	return writeFile(dst, rc, mode)
}

func extractTarGz(archivePath, binName, dst string, mode os.FileMode) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer closeQuietly(f)

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer closeQuietly(gz)

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s not found in archive", binName)
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeReg && path.Base(hdr.Name) == binName {
			return writeFile(dst, tr, mode)
		}
	}
}

func writeFile(dst string, src io.Reader, mode os.FileMode) error {
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(src, maxBinarySize+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxBinarySize {
		err = errors.New("binary in archive exceeds size limit")
	}
	if err == nil {
		err = os.Chmod(dst, mode) // OpenFile's mode is subject to umask
	}
	if err != nil {
		_ = os.Remove(dst)
	}
	return err
}

// replaceExecutable swaps newPath in for exe. Renaming a running executable is
// allowed on Linux/macOS and on Windows (deleting it is not), so the old binary
// is moved aside first and removed afterwards; on Windows that removal fails
// while we are still running, and the leftover is deleted by the next update.
func replaceExecutable(exe, newPath, oldPath string) error {
	_ = os.Remove(oldPath) // leftover from a previous update

	if err := os.Rename(exe, oldPath); err != nil {
		return fmt.Errorf("move current binary aside (need write access to %s): %w", filepath.Dir(exe), err)
	}
	if err := os.Rename(newPath, exe); err != nil {
		if rerr := os.Rename(oldPath, exe); rerr != nil {
			return fmt.Errorf("install new binary: %w; rollback failed, previous binary is at %s: %w", err, oldPath, rerr)
		}
		return fmt.Errorf("install new binary (rolled back): %w", err)
	}
	_ = os.Remove(oldPath)
	return nil
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

// parseSemver accepts plain vMAJOR.MINOR.PATCH only; suffixes such as
// "-next" (goreleaser snapshots) or "-rc1" are rejected on purpose.
func parseSemver(s string) ([3]int, bool) {
	m := semverRe.FindStringSubmatch(s)
	if m == nil {
		return [3]int{}, false
	}
	var v [3]int
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return [3]int{}, false
		}
		v[i] = n
	}
	return v, true
}

func compare(a, b [3]int) int {
	for i := range a {
		switch {
		case a[i] > b[i]:
			return 1
		case a[i] < b[i]:
			return -1
		}
	}
	return 0
}

func closeQuietly(w io.Closer) {
	if err := w.Close(); err != nil {
		log.Printf("failed to close writer: %v", err)
	}
}

func removeQuietly(filePath string) {
	if err := os.Remove(filePath); err != nil {
		log.Printf("failed to remove %s: %v", filePath, err)
	}
}
