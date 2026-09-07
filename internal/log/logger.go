package log

import (
	"fmt"
	"os"
	"sync"
)

type Logger func(string)

var (
	logger   Logger = Stdout
	isCustom bool
	buffer   []string
	mu       sync.Mutex
)

func SetLogger(l Logger) {
	mu.Lock()
	defer mu.Unlock()
	if l != nil {
		logger = l
		if !isCustom {
			for _, s := range buffer {
				logger(s)
			}
			buffer = nil
			isCustom = true
		}
	}
}

func GetLogger() Logger {
	mu.Lock()
	defer mu.Unlock()
	return logger
}

func Log(s string) {
	mu.Lock()
	l := logger
	if !isCustom {
		buffer = append(buffer, s)
		if len(buffer) > 100 {
			buffer = buffer[1:]
		}
	}
	mu.Unlock()
	l(s)
}

func Logf(s string, args ...any) {
	Log(fmt.Sprintf(s, args...))
}

var Stdout = func(s string) {
	fmt.Fprintln(os.Stdout, s)
}

var Stderr = func(s string) {
	fmt.Fprintln(os.Stderr, s)
}

var Null = func(string) {
}
