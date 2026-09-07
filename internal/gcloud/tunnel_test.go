package gcloud

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func Test_updateKnownHostsWithGetHostKey(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsFile := filepath.Join(tempDir, "known_hosts")

	type args struct {
		sshContext *types.Context
		port       int
		timeout    time.Duration
		hostKey    []string
	}
	tests := []struct {
		name           string
		args           args
		initialContent string
		wantContent    string
	}{
		{
			name: "New host added to empty file",
			args: args{
				sshContext: &types.Context{Host: "host1", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
				},
			},
			initialContent: "",
			wantContent:    "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
		},
		{
			name: "empty line removed",
			args: args{
				sshContext: &types.Context{Host: "host1", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
				},
			},
			initialContent: "  \n",
			wantContent:    "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
		},
		{
			name: "New host added to existing file",
			args: args{
				sshContext: &types.Context{Host: "host2", KnownHostsFile: knownHostsFile},
				port:       2223,
				timeout:    time.Second,
				hostKey: []string{
					"[host2]:2223 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIaO5q9p5W/0dBPRimDW/KX0Qp7MDDBFmFW5j04o4yJP",
				},
			},
			initialContent: "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
			wantContent:    "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs\n[host2]:2223 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIaO5q9p5W/0dBPRimDW/KX0Qp7MDDBFmFW5j04o4yJP",
		},
		{
			name: "Existing host key updated",
			args: args{
				sshContext: &types.Context{Host: "host1", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAXCseVAx4bM5AGl1Zzym5ju/i3FZbtevUXJeerqS1dQ",
				},
			},
			initialContent: "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs\n[host2]:2223 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIaO5q9p5W/0dBPRimDW/KX0Qp7MDDBFmFW5j04o4yJP",
			wantContent:    "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAXCseVAx4bM5AGl1Zzym5ju/i3FZbtevUXJeerqS1dQ\n[host2]:2223 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIaO5q9p5W/0dBPRimDW/KX0Qp7MDDBFmFW5j04o4yJP",
		},
		{
			name: "Existing host key same, no change",
			args: args{
				sshContext: &types.Context{Host: "host1", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
				},
			},
			initialContent: "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs\n[host2]:2223 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIaO5q9p5W/0dBPRimDW/KX0Qp7MDDBFmFW5j04o4yJP",
			wantContent:    "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs\n[host2]:2223 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIaO5q9p5W/0dBPRimDW/KX0Qp7MDDBFmFW5j04o4yJP",
		},
		{
			name: "Multiple host keys (localhost case)",
			args: args{
				sshContext: &types.Context{Host: "localhost", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[127.0.0.1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
					"[localhost]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
				},
			},
			initialContent: "",
			wantContent:    "[127.0.0.1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs\n[localhost]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
		},
		{
			name: "Update one of multiple host keys",
			args: args{
				sshContext: &types.Context{Host: "localhost", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[127.0.0.1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrsBAflylxBfWGcJxqlYDkI5mv5R8UrKx2FNjnJtncq",
					"[localhost]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrsBAflylxBfWGcJxqlYDkI5mv5R8UrKx2FNjnJtncq",
				},
			},
			initialContent: "[127.0.0.1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs\n[localhost]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
			wantContent:    "[127.0.0.1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrsBAflylxBfWGcJxqlYDkI5mv5R8UrKx2FNjnJtncq\n[localhost]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrsBAflylxBfWGcJxqlYDkI5mv5R8UrKx2FNjnJtncq",
		},
		{
			name: "Existing hashed host key updated",
			args: args{
				sshContext: &types.Context{Host: "host1", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGfIKnaLsbnn6B3ty0PaOYwhGs+WHmL9R6KRBge3ktB7",
				},
			},
			initialContent: knownhosts.HashHostname(
				"[host1]:2222",
			) + " ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
			wantContent: "[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGfIKnaLsbnn6B3ty0PaOYwhGs+WHmL9R6KRBge3ktB7",
		},
		{
			name: "Existing hashed host key same, salt preserved",
			args: args{
				sshContext: &types.Context{Host: "host1", KnownHostsFile: knownHostsFile},
				port:       2222,
				timeout:    time.Second,
				hostKey: []string{
					"[host1]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
				},
			},
			initialContent: "|1|4xwzUkKFmnx9DMq/llELNhEsIkM=|sXfOGajfTzuClVT3s4ftdt2hkv8= ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
			wantContent:    "|1|4xwzUkKFmnx9DMq/llELNhEsIkM=|sXfOGajfTzuClVT3s4ftdt2hkv8= ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEDommg1NSIETRXFu6W0WongUTpVIBX2EbfifqcD7rvs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.WriteFile(knownHostsFile, []byte(tt.initialContent), 0o644)
			updateKnownHosts(
				tt.args.sshContext,
				tt.args.hostKey,
			)

			content, _ := os.ReadFile(knownHostsFile)
			gotLines := strings.Split(strings.TrimSpace(string(content)), "\n")
			wantLines := strings.Split(strings.TrimSpace(tt.wantContent), "\n")

			if tt.name == "Existing hashed host key same, salt preserved" {
				if strings.TrimSpace(string(content)) != strings.TrimSpace(tt.wantContent) {
					t.Errorf("salt not preserved: got %q, want %q", string(content), tt.wantContent)
				}
				return
			}

			if len(gotLines) != len(wantLines) {
				if tt.wantContent == "" && len(gotLines) == 1 && gotLines[0] == "" {
					return
				}
				t.Errorf(
					"updateKnownHostsWithGetHostKey() got %d lines, want %d\ngot:\n%v\n\nwant:\n%v",
					len(gotLines),
					len(wantLines),
					strings.Join(gotLines, "\n"),
					tt.wantContent,
				)
				return
			}

			cb, err := knownhosts.New(knownHostsFile)
			if err != nil {
				t.Fatal(err)
			}

			for _, wantLine := range wantLines {
				wantParts := strings.SplitN(wantLine, " ", 2)
				addr := wantParts[0]

				pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(wantLine))
				if err != nil {
					t.Fatal(err)
				}

				// Check if the address exists in the file with the correct key
				err = cb(addr, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pubKey)
				if err != nil {
					t.Errorf("address %s with correct key not found in known_hosts: %v", addr, err)
				}
			}
		})
	}
}
