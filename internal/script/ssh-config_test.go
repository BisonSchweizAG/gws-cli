package script

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestSSHConfig(t *testing.T) {
	tests := []struct {
		name         string
		fileName     string
		context      *types.Context
		forwardAgent bool
		hostName     string
		wantName     string
		want         string
	}{
		{
			name: "Basic Configuration",
			context: &types.Context{
				Host: "example.com",
				Port: 2222,
				User: "user",
			},
			wantName: "gws-test-context",
			want: `Host test-context
  HostName example.com
  Port 2222
  User user

  Compression yes
  TCPKeepAlive yes
  ServerAliveInterval 60
  ServerAliveCountMax 10`,
		},
		{
			name: "With private key and known hosts",
			context: &types.Context{
				Host:           "example.com",
				Port:           2222,
				User:           "user",
				PrivateKeyFile: "/home/user/.ssh/id_ed25519",
				KnownHostsFile: "/home/user/.ssh/known_hosts",
			},
			wantName: "gws-test-context",
			want: `Host test-context
  HostName example.com
  Port 2222
  User user
  IdentityFile /home/user/.ssh/id_ed25519
  UserKnownHostsFile /home/user/.ssh/known_hosts
  Compression yes
  TCPKeepAlive yes
  ServerAliveInterval 60
  ServerAliveCountMax 10`,
		},
		{
			name: "With forward agent",
			context: &types.Context{
				Host: "example.com",
				Port: 2222,
				User: "user",
			},
			forwardAgent: true,
			wantName:     "gws-test-context",
			want: `Host test-context
  HostName example.com
  Port 2222
  User user

  ForwardAgent yes
  Compression yes
  TCPKeepAlive yes
  ServerAliveInterval 60
  ServerAliveCountMax 10`,
		},
		{
			name: "With name override",
			context: &types.Context{
				Host: "example.com",
				Port: 2222,
				User: "user",
			},
			hostName: "my-host",
			wantName: "gws-my-host",
			want: `Host my-host
  HostName example.com
  Port 2222
  User user

  Compression yes
  TCPKeepAlive yes
  ServerAliveInterval 60
  ServerAliveCountMax 10`,
		},
		{
			name:     "Custom file name",
			fileName: "/tmp/custom.conf",
			context: &types.Context{
				Host: "example.com",
				Port: 2222,
			},
			wantName: "custom.conf",
			want: `Host test-context
  HostName example.com
  Port 2222

  Compression yes
  TCPKeepAlive yes
  ServerAliveInterval 60
  ServerAliveCountMax 10`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &types.Config{
				CurrentContextName: "test-context",
				Contexts: map[string]*types.Context{
					"test-context": tt.context,
				},
			}
			_ = cfg.SwitchContext(cfg.CurrentContextName, false)

			name, content, err := SSHConfigWithOptions(tt.forwardAgent, tt.hostName)(cfg, tt.fileName)
			if err != nil {
				t.Fatalf("SSHConfig() error = %v", err)
			}
			if name != tt.wantName {
				t.Errorf("SSHConfig() name = %q, want %q", name, tt.wantName)
			}
			if string(content) != tt.want {
				t.Errorf("SSHConfig() content = %q, want %q", content, tt.want)
			}
		})
	}
}

func TestHasSSHConfigInclude(t *testing.T) {
	tests := []struct {
		name        string
		existing    *string
		wantPresent bool
	}{
		{
			name: "No ssh config",
		},
		{
			name:     "Existing config without include",
			existing: new("Host example\n  HostName example.com\n"),
		},
		{
			name:        "Existing include",
			existing:    new("include ~/.ssh/config.d/*\n\nHost example\n"),
			wantPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			if tt.existing != nil {
				if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
					t.Fatalf("failed to create ssh dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte(*tt.existing), 0o600); err != nil {
					t.Fatalf("failed to write ssh config: %v", err)
				}
			}

			configPath, present, err := HasSSHConfigInclude()
			if err != nil {
				t.Fatalf("HasSSHConfigInclude() error = %v", err)
			}
			if present != tt.wantPresent {
				t.Errorf("HasSSHConfigInclude() present = %v, want %v", present, tt.wantPresent)
			}
			if configPath != filepath.Join(home, ".ssh", "config") {
				t.Errorf("HasSSHConfigInclude() path = %q", configPath)
			}

			got, err := os.ReadFile(configPath)
			if tt.existing == nil {
				if !os.IsNotExist(err) {
					t.Errorf("ssh config must not be created, got %q (err = %v)", got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to read ssh config: %v", err)
			}
			if string(got) != *tt.existing {
				t.Errorf("ssh config = %q, want %q (must stay untouched)", got, *tt.existing)
			}
		})
	}
}

func TestAddSSHConfigInclude(t *testing.T) {
	tests := []struct {
		name      string
		existing  *string
		wantAdded bool
		want      string
	}{
		{
			name:      "No ssh config",
			wantAdded: true,
			want:      "Include config.d/*\n",
		},
		{
			name:      "Existing config without include",
			existing:  new("Host example\n  HostName example.com\n"),
			wantAdded: true,
			want:      "Include config.d/*\n\nHost example\n  HostName example.com\n",
		},
		{
			name:      "Existing include",
			existing:  new("include ~/.ssh/config.d/*\n\nHost example\n"),
			wantAdded: false,
			want:      "include ~/.ssh/config.d/*\n\nHost example\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			if tt.existing != nil {
				if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
					t.Fatalf("failed to create ssh dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte(*tt.existing), 0o600); err != nil {
					t.Fatalf("failed to write ssh config: %v", err)
				}
			}

			configPath, added, err := AddSSHConfigInclude()
			if err != nil {
				t.Fatalf("AddSSHConfigInclude() error = %v", err)
			}
			if added != tt.wantAdded {
				t.Errorf("AddSSHConfigInclude() added = %v, want %v", added, tt.wantAdded)
			}
			if configPath != filepath.Join(home, ".ssh", "config") {
				t.Errorf("AddSSHConfigInclude() path = %q", configPath)
			}

			got, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read ssh config: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("ssh config = %q, want %q", got, tt.want)
			}
		})
	}
}
