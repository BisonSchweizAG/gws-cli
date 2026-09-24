package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bisonschweizag/gws-cli/version"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		cur, cand string
		want      bool
		wantErr   error
	}{
		{"v0.0.0", "v1.2.4", true, nil},
		{"v1.2.3", "v1.2.4", true, nil},
		{"v1.2.3", "v1.10.0", true, nil}, // numeric, not lexical
		{"v1.2.3", "v1.2.3", false, nil},
		{"v2.0.0", "v1.9.9", false, nil},
		{"1.2.3", "v1.2.4", true, nil},
		{"dev", "v1.0.0", false, ErrUnknownVersion},
		{"v1.2.3-next", "v1.2.4", false, ErrUnknownVersion},
		{"v1.2.3-rc.1", "v1.2.3", false, ErrUnknownVersion},
		{"", "v1.0.0", false, ErrUnknownVersion},
		{version.Version, "v1.2.4", true, nil},
	}
	for _, tt := range tests {
		got, err := IsNewer(tt.cur, tt.cand)
		if !errors.Is(err, tt.wantErr) || got != tt.want {
			t.Errorf("IsNewer(%q,%q) = %v, %v; want %v, %v", tt.cur, tt.cand, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestFindChecksum(t *testing.T) {
	data := []byte("AAA1  gws_1.0.0_linux_amd64.tar.gz\nbbb2  gws_1.0.0_windows_amd64.zip\n")
	got, err := findChecksum(data, "gws_1.0.0_linux_amd64.tar.gz")
	if err != nil || got != "aaa1" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := findChecksum(data, "missing"); err == nil {
		t.Fatal("expected error for missing entry")
	}
}

//nolint:unparam
func makeTarGz(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// Extra files like the real archives (LICENSE, README.md).
	for _, f := range []struct{ n, c string }{{"LICENSE", "lic"}, {name, content}} {
		hdr := &tar.Header{Name: f.n, Mode: 0o755, Size: int64(len(f.c)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.c)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func makeZip(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range []struct{ n, c string }{{"README.md", "readme"}, {name, content}} {
		w, err := zw.Create(f.n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(f.c)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// newServer mimics the GitHub API plus asset downloads for one platform.
func newServer(t *testing.T, archiveName string, archive []byte, checksum string) *Updater {
	t.Helper()
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/o/r/releases", func(w http.ResponseWriter, _ *http.Request) {
		asset := func(n string) string {
			return fmt.Sprintf(`{"name":%q,"browser_download_url":"%s/dl/%s"}`, n, srv.URL, n)
		}
		_, _ = fmt.Fprintf(w, `[
		 {"tag_name":"v9.0.0","prerelease":true,"assets":[]},
		 {"tag_name":"v8.0.0","draft":true,"assets":[]},
		 {"tag_name":"nightly","assets":[]},
		 {"tag_name":"v99","assets":[]},
		 {"tag_name":"v50.0.0-rc.1","assets":[]},
		 {"tag_name":"v60.0.0+build","assets":[]},
		 {"tag_name":"v1.2.0","assets":[]},
		 {"tag_name":"v1.10.0","assets":[%s,%s]}
		]`, asset(archiveName), asset("checksums.txt"))
	})
	mux.HandleFunc("/dl/"+archiveName, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/dl/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  %s\n", checksum, archiveName)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &Updater{Owner: "o", Repo: "r", APIBase: srv.URL, Client: srv.Client()}
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

//nolint:unparam
func writeExe(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUpdateLinuxTarGz(t *testing.T) {
	archive := makeTarGz(t, "gws", "NEW")
	u := newServer(t, "gws_1.10.0_linux_amd64.tar.gz", archive, sum(archive))
	u.GOOS, u.GOARCH = "linux", "amd64"
	u.ExePath = writeExe(t, "gws", "OLD")

	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "v1.10.0" {
		t.Fatalf("picked %s, want v1.10.0", rel.Tag)
	}
	if err := u.Apply(context.Background(), rel); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(u.ExePath)
	if string(got) != "NEW" {
		t.Fatalf("exe content = %q", got)
	}
	if fi, _ := os.Stat(u.ExePath); fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(u.ExePath))
	if len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}

func TestUpdateWindowsZip(t *testing.T) {
	archive := makeZip(t, "gws.exe", "NEW")
	u := newServer(t, "gws_1.10.0_windows_amd64.zip", archive, sum(archive))
	u.GOOS, u.GOARCH = "windows", "amd64"
	u.ExePath = writeExe(t, "gws.exe", "OLD")

	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(u.ExePath); string(got) != "NEW" {
		t.Fatalf("exe content = %q", got)
	}
}

func TestChecksumMismatchKeepsOldBinary(t *testing.T) {
	archive := makeTarGz(t, "gws", "NEW")
	u := newServer(t, "gws_1.10.0_linux_amd64.tar.gz", archive, sum([]byte("something else")))
	u.GOOS, u.GOARCH = "linux", "amd64"
	u.ExePath = writeExe(t, "gws", "OLD")

	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err == nil {
		t.Fatal("expected checksum error")
	}
	if got, _ := os.ReadFile(u.ExePath); string(got) != "OLD" {
		t.Fatalf("exe was modified: %q", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(u.ExePath))
	if len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}

func TestUnsupportedPlatform(t *testing.T) {
	archive := makeTarGz(t, "gws", "NEW")
	u := newServer(t, "gws_1.10.0_linux_amd64.tar.gz", archive, sum(archive))
	u.GOOS, u.GOARCH = "linux", "arm64" // not built by .goreleaser.yml
	if _, err := u.Latest(context.Background()); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("err = %v, want ErrNoAsset", err)
	}
}

func TestMissingBinaryInArchiveLeavesNoFiles(t *testing.T) {
	archive := makeTarGz(t, "not-gws", "NEW")
	u := newServer(t, "gws_1.10.0_linux_amd64.tar.gz", archive, sum(archive))
	u.GOOS, u.GOARCH = "linux", "amd64"
	u.ExePath = writeExe(t, "gws", "OLD")

	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err == nil {
		t.Fatal("expected error for archive without gws binary")
	}
	if got, _ := os.ReadFile(u.ExePath); string(got) != "OLD" {
		t.Fatalf("exe was modified: %q", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(u.ExePath))
	if len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}
