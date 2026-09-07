package types

import (
	"os"
	"testing"
)

func TestFile_Validate(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	tests := []struct {
		name       string
		file       File
		privateKey string
		knownHosts string
		wantErr    bool
	}{
		{
			name: "valid",
			file: File{
				SourcePath:  tmpFile.Name(),
				Path:        "/dest/path",
				Permissions: "0644",
				Direction:   DirectionUp,
			},
			wantErr: false,
		},
		{
			name: "valid without permissions direction is up",
			file: File{
				SourcePath: tmpFile.Name(),
				Path:       "/dest/path",
				Direction:  DirectionUp,
			},
			wantErr: false,
		},
		{
			name: "valid without permissions direction is down",
			file: File{
				SourcePath: tmpFile.Name(),
				Path:       "/dest/path",
				Direction:  DirectionDown,
			},
			wantErr: false,
		},
		{
			name: "sourcePath is directorydirection is down",
			file: File{
				SourcePath: t.TempDir(),
				Path:       "/dest/path",
				Direction:  DirectionDown,
			},
			wantErr: true,
		},
		{
			name: "valid without Direction",
			file: File{
				SourcePath:  tmpFile.Name(),
				Path:        "/dest/path",
				Permissions: "0644",
			},
			wantErr: false,
		},
		{
			name: "missing sourcePath",
			file: File{
				Path:        "/dest/path",
				Permissions: "0644",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			file: File{
				SourcePath:  tmpFile.Name(),
				Permissions: "0644",
			},
			wantErr: true,
		},
		{
			name: "missing both required fields",
			file: File{
				Permissions: "0644",
			},
			wantErr: true,
		},
		{
			name: "sourcePath not a file and direction is up",
			file: File{
				SourcePath: "/nonexistent/file",
				Path:       "/dest/path",
				Direction:  "up",
			},
			wantErr: true,
		},
		{
			name: "sourcePath not a file and direction is down",
			file: File{
				SourcePath: "/nonexistent/file",
				Path:       "/dest/path",
				Direction:  "down",
			},
			wantErr: false,
		},
		{
			name: "sourcePath is directory",
			file: File{
				SourcePath: t.TempDir(),
				Path:       "/dest/path",
			},
			wantErr: true,
		},
		{
			name:    "empty struct",
			file:    File{},
			wantErr: true,
		},
		{
			name: "direction not up or down",
			file: File{
				SourcePath:  tmpFile.Name(),
				Path:        "/dest/path",
				Permissions: "0644",
				Direction:   "none",
			},
			wantErr: true,
		},
		{
			name:       "key is not file",
			privateKey: "/nonexistent/file",
			wantErr:    true,
		},
		{
			name:       "known host is not file",
			knownHosts: "/nonexistent/file",
			wantErr:    true,
		},
		{
			name:       "key and known host are files",
			privateKey: tmpFile.Name(),
			knownHosts: tmpFile.Name(),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Contexts: map[string]*Context{
					"test": {
						Host:           "localhost",
						Port:           22,
						Files:          []File{tt.file},
						PrivateKeyFile: tt.privateKey,
						KnownHostsFile: tt.knownHosts,
					},
				},
			}
			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
