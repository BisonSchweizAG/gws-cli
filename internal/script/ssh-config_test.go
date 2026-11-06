package script

import (
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestSSHConfig(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		context  *types.Context
		wantName string
		want     string
	}{
		{
			name: "Basic Configuration",
			context: &types.Context{
				Host: "example.com",
				Port: 2222,
				User: "user",
			},
			wantName: "gws-ssh-config-test-context",
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
			wantName: "gws-ssh-config-test-context",
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

			name, content, err := SSHConfig(cfg, tt.fileName)
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
