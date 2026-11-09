package shutdownlog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

const retentionPeriod = 7 * 24 * time.Hour

// Dir returns the configuration directory for cfg, falling back to the default
// configuration directory when cfg has no file path.
func Dir(cfg *types.Config) string {
	if cfg != nil && cfg.FilePath != "" {
		return filepath.Dir(cfg.FilePath)
	}
	_, configDir, _ := types.DefaultConfigPaths()
	return configDir
}

// Append adds msg to shutdown.log and removes timestamped entries older than
// seven days. Lines without a valid leading RFC3339 timestamp are retained.
func Append(dir, msg string, now time.Time) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	path := filepath.Join(dir, "shutdown.log")
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	lines := existingLines(data)
	cutoff := now.Add(-retentionPeriod)
	keptLines := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		timestampText, _, found := strings.Cut(line, " ")
		timestamp, parseErr := time.Parse(time.RFC3339, timestampText)
		if found && parseErr == nil && timestamp.Before(cutoff) {
			continue
		}
		keptLines = append(keptLines, line)
	}
	keptLines = append(keptLines, now.Format(time.RFC3339)+" "+msg)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		return errors.Join(err, file.Close())
	}
	if _, err := file.WriteString(strings.Join(keptLines, "\n") + "\n"); err != nil {
		return errors.Join(err, file.Close())
	}
	return file.Close()
}

func existingLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
