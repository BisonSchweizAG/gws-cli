package types

import (
	"os"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

func TestConfig_Validate(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with multiple contexts",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "localhost", Port: 8080, Files: []File{
						{SourcePath: tmpFile.Name(), Path: "/path/to/dest"},
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config with missing contexts",
			config: &Config{
				Contexts: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid context fields",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "", Port: 0, Files: nil},
				},
			},
			wantErr: true,
		},
		{
			name: "missing required file fields",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "localhost", Port: 8080, Files: []File{
						{SourcePath: "", Path: "/path/to/dest"},
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "valid config with one context",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "example.com", Port: 22, User: "user", Files: []File{
						{SourcePath: tmpFile.Name(), Path: "/valid/dest"},
					}},
				},
				CurrentContextName: "context1",
			},
			wantErr: false,
		},
		{
			name: "chrome executable missing",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "localhost", Port: 8080, Files: []File{
						{SourcePath: tmpFile.Name(), Path: "/path/to/dest"},
					}},
				},
				ChromeBrowser: &ChromeBrowserConfig{ProfileDirectory: "foo"},
			},
			wantErr: true,
		},
		{
			name: "chrome profile missing",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "localhost", Port: 8080, Files: []File{
						{SourcePath: tmpFile.Name(), Path: "/path/to/dest"},
					}},
				},
				ChromeBrowser: &ChromeBrowserConfig{ExecutablePath: tmpFile.Name()},
			},
			wantErr: true,
		},
		{
			name: "valid chrome config",
			config: &Config{
				Contexts: map[string]*Context{
					"context1": {Host: "localhost", Port: 8080, Files: []File{
						{SourcePath: tmpFile.Name(), Path: "/path/to/dest"},
					}},
				},
				ChromeBrowser: &ChromeBrowserConfig{
					ExecutablePath:   tmpFile.Name(),
					ProfileDirectory: "foo",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validate := validator.New()
			err := validate.Struct(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_StartTimeout(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected time.Duration
	}{
		{
			name:     "nil config returns default 100s",
			config:   nil,
			expected: defaultStartTimeoutSeconds * time.Second,
		},
		{
			name:     "empty config returns default 100s",
			config:   &Config{},
			expected: defaultStartTimeoutSeconds * time.Second,
		},
		{
			name: "StartTimeoutSeconds on Config",
			config: &Config{
				StartTimeoutSeconds: 45,
			},
			expected: 45 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.StartTimeout()
			if got != tt.expected {
				t.Errorf("StartTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_SSHTimeout(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected time.Duration
	}{
		{
			name:     "default ssh timeout",
			config:   &Config{},
			expected: 30 * time.Second,
		},
		{
			name: "custom ssh timeout",
			config: &Config{
				SSHTimeoutSeconds: 15,
			},
			expected: 15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.SSHTimeout()
			if got != tt.expected {
				t.Errorf("SSHTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}
