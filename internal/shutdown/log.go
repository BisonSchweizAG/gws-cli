package shutdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

var (
	logFileOnce sync.Once
	logFile     *os.File
	logMu       sync.Mutex
)

// logFile returns the shared shutdown log file, creating it on first use.
func getLogFile() (*os.File, error) {
	var err error
	logFileOnce.Do(func() {
		_, configDir, _ := types.DefaultConfigPaths()
		if configDir == "" {
			logFile = nil
			err = fmt.Errorf("config dir not available")
			return
		}
		if err = os.MkdirAll(configDir, 0o755); err != nil {
			logFile = nil
			return
		}
		path := filepath.Join(configDir, "shutdown.log")
		logFile, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	})
	return logFile, err
}

// log writes a timestamped line to the shared shutdown log file.
// It is safe to call from any goroutine.
func log(format string, args ...interface{}) {
	f, err := getLogFile()
	if err != nil || f == nil {
		return
	}
	msg := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
	logMu.Lock()
	defer logMu.Unlock()
	_, _ = f.WriteString(msg)
}
