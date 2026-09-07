package ssh

import (
	"encoding/base64"
	"net"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type mockPublicKey struct {
	ssh.PublicKey
	t string
	m []byte
}

func (m *mockPublicKey) Type() string    { return m.t }
func (m *mockPublicKey) Marshal() []byte { return m.m }

func Test_formatHostKey(t *testing.T) {
	key := &mockPublicKey{t: "ssh-rsa", m: []byte("keydata")}
	got := formatHostKey("host", 22, key)
	wantPrefix := knownhosts.Normalize(net.JoinHostPort("host", "22"))
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("formatHostKey() should start with %s, but got %v", wantPrefix, got)
	}
	wantSuffix := " ssh-rsa " + base64.StdEncoding.EncodeToString([]byte("keydata"))
	if !strings.HasSuffix(got, wantSuffix) {
		t.Errorf("formatHostKey() suffix = %v, want suffix %v", got, wantSuffix)
	}
}

func verifyPlainLine(t *testing.T, line, wantPrefix, wantSuffix string) {
	t.Helper()
	if !strings.HasPrefix(line, wantPrefix) {
		t.Errorf("line should start with %s, but got %v", wantPrefix, line)
	}
	if !strings.HasSuffix(line, wantSuffix) {
		t.Errorf("line suffix = %v, want suffix %v", line, wantSuffix)
	}
}

func Test_HostKeyCallback_Localhost(t *testing.T) {
	key := &mockPublicKey{t: "ssh-rsa", m: []byte("keydata")}
	keyStr := base64.StdEncoding.EncodeToString([]byte("keydata"))
	wantSuffix := " ssh-rsa " + keyStr

	tests := []struct {
		name      string
		hostname  string
		remote    net.Addr
		lineCount int
	}{
		{
			name:      "Not localhost",
			hostname:  "example.com:22",
			remote:    &net.TCPAddr{IP: net.ParseIP("1.2.3.4"), Port: 22},
			lineCount: 1,
		},
		{
			name:      "Localhost",
			hostname:  "localhost:22",
			remote:    &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22},
			lineCount: 2,
		},
		{
			name:      "Localhost already IP",
			hostname:  "127.0.0.1:22",
			remote:    &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22},
			lineCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var knownHostsEntry string
			callback := func(hostname string, remote net.Addr, key ssh.PublicKey) {
				if tcpAddr, ok := remote.(*net.TCPAddr); ok {
					knownHostsEntry = formatHostKey(tcpAddr.IP.String(), tcpAddr.Port, key)
					host, _, err := net.SplitHostPort(hostname)
					if err != nil {
						host = hostname
					}
					if host == "localhost" {
						knownHostsEntry += "\n" + formatHostKey(host, tcpAddr.Port, key)
					}
				}
			}

			callback(tt.hostname, tt.remote, key)
			lines := strings.Split(knownHostsEntry, "\n")
			if len(lines) != tt.lineCount {
				t.Errorf("Expected %d lines, got %d: %q", tt.lineCount, len(lines), knownHostsEntry)
			}

			wantPrefixes := []string{}
			if tcpAddr, ok := tt.remote.(*net.TCPAddr); ok {
				wantPrefixes = append(
					wantPrefixes,
					knownhosts.Normalize(net.JoinHostPort(tcpAddr.IP.String(), strconv.Itoa(tcpAddr.Port))),
				)
				host, _, err := net.SplitHostPort(tt.hostname)
				if err != nil {
					host = tt.hostname
				}
				if host == "localhost" {
					wantPrefixes = append(
						wantPrefixes,
						knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(tcpAddr.Port))),
					)
				}
			}

			for i, line := range lines {
				verifyPlainLine(t, line, wantPrefixes[i], wantSuffix)
			}
		})
	}
}
