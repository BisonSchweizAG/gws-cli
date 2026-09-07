package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/bisonschweizag/gws-cli/internal/env"
	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/passwd"
)

func NewClient(addr, user, privateKeyFile string, timeout time.Duration) (Client, error) {
	return NewClientWithPassphrase(addr, user, privateKeyFile, timeout, nil)
}

func NewClientWithPassphrase(addr, user, privateKeyFile string, timeout time.Duration, passphrase []byte) (Client, error) {
	privateKey, err := os.ReadFile(env.ExpandEnv(privateKeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse the private key
	auths, err := evaluateAuthMethodWithPassphrase(privateKey, privateKeyFile, passphrase)
	if err != nil {
		return nil, err
	}

	// Define SSH connection details
	var knownHostsEntry string
	clientConfig := &ssh.ClientConfig{
		User: user,
		Auth: auths,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// #nosec G106: Insecure, as we always get a new cert with gcloud
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
			return nil
		},
	}

	var sshClient *ssh.Client
	log.Logf("⏲  Using ssh client with timeout %s", timeout.String())
	clientConfig.Timeout = timeout
	sshClient, err = clientWithTimeout(addr, timeout, clientConfig)
	if err != nil {
		return nil, err
	}

	// For other authentication methods see ssh.ClientConfig and ssh.AuthMethod

	// Create a new SCP client
	scpClient := scp.NewClient(addr, clientConfig)

	// Connect to the remote server
	err = scpClient.Connect()
	if err != nil {
		return nil, fmt.Errorf("couldn't establish a connection to the remote server: %w", err)
	}

	return &client{
		sshClient:       sshClient,
		scpClient:       scpClient,
		knownHostsEntry: knownHostsEntry,
	}, nil
}

func formatHostKey(host string, port int, key ssh.PublicKey) string {
	addr := knownhosts.Normalize(net.JoinHostPort(host, strconv.Itoa(port)))
	return knownhosts.Line([]string{addr}, key)
}

// GetHostKey fetches the host public key without authenticating.
func GetHostKey(addr string, timeout time.Duration) ([]string, error) {
	var hostKey ssh.PublicKey

	config := &ssh.ClientConfig{
		// We provide no Auth methods, so authentication will fail,
		// but the HostKeyCallback happens before authentication.
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			hostKey = key
			return nil
		},
		Timeout: timeout,
	}

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// This performs the handshake. It will return an error because
	// no Auth methods are provided, but the HostKeyCallback will have been triggered.
	sshConn, _, _, err := ssh.NewClientConn(conn, addr, config)
	if err == nil {
		_ = sshConn.Close()
	}

	if hostKey == nil {
		return nil, errors.New("failed to extract host key")
	}
	tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("failed to extract tcp address")
	}

	var lines []string
	lines = append(lines, formatHostKey(tcpAddr.IP.String(), tcpAddr.Port, hostKey))
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	lines = append(lines, formatHostKey(host, tcpAddr.Port, hostKey))
	return lines, nil
}

func clientWithTimeout(addr string, timeout time.Duration, clientConfig *ssh.ClientConfig) (*ssh.Client, error) {
	// Use a dialer with TCP KeepAlive enabled to prevent connection drops
	dialer := net.Dialer{
		Timeout:   timeout,
		KeepAlive: timeout,
	}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	// Connect to the SSH server using the existing TCP connection
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	return sshClient, nil
}

type Client interface {
	Close()
	Execute(command string) (output string, err error)
	UploadFile(from, to, permissions string) (err error)
	DownloadFile(from, to, permissions string) (err error)
	KnownHostsEntry() string
}

type client struct {
	sshClient       *ssh.Client
	scpClient       scp.Client
	knownHostsEntry string
}

func (c *client) Close() {
	if c.sshClient != nil {
		_ = c.sshClient.Close()
	}

	c.scpClient.Close()
}

func (c *client) KnownHostsEntry() string {
	return c.knownHostsEntry
}

func (c *client) Execute(command string) (string, error) {
	log.Logf("Executing ssh %q", command)

	// Start a new SSH session
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Execute the command
	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	return string(output), nil
}

func (c *client) UploadFile(from, to, permissions string) error {
	log.Logf("Copy file from %q to %q with permissions %s", from, to, permissions)
	// Open a file
	f, err := os.Open(env.ExpandEnv(from))
	if err != nil {
		return fmt.Errorf("error while opening file: %w", err)
	}

	// Close the file after it has been copied
	defer f.Close()

	err = c.scpClient.CopyFromFile(context.Background(), *f, to, permissions)
	if err != nil {
		return fmt.Errorf("error while copying file: %w", err)
	}
	_, err = c.Execute(fmt.Sprintf("chmod %s %s", permissions, to))
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (c *client) DownloadFile(from, to, permissions string) error {
	log.Logf("Copy file from %q to %q", from, to)

	localPath := env.ExpandEnv(to)

	perm, _ := strconv.ParseUint(permissions, 8, 32)

	f, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE, os.FileMode(perm))
	if err != nil {
		return fmt.Errorf("error while opening file: %w", err)
	}

	defer f.Close()

	err = c.scpClient.CopyFromRemote(context.Background(), f, from)
	if err != nil {
		return fmt.Errorf("error while copying file: %w", err)
	}

	return nil
}

func NeedsPassphrase(privateKeyFile string) (bool, error) {
	auth, err := getSSHAgentAuthMethod()
	if err != nil {
		return false, err
	}
	if auth != nil {
		return false, nil
	}

	privateKey, err := os.ReadFile(env.ExpandEnv(privateKeyFile))
	if err != nil {
		return false, fmt.Errorf("failed to read private key: %w", err)
	}

	_, err = ssh.ParsePrivateKey(privateKey)
	if err != nil {
		if _, ok := errors.AsType[*ssh.PassphraseMissingError](err); ok {
			return true, nil
		}
	}
	return false, nil
}

func evaluateAuthMethodWithPassphrase(privateKey []byte, privateKeyFile string, passphrase []byte) ([]ssh.AuthMethod, error) {
	var auths []ssh.AuthMethod

	// Try SSH agent first if available
	if agentAuth, err := getSSHAgentAuthMethod(); err != nil {
		return nil, err
	} else if agentAuth != nil {
		auths = append(auths, agentAuth)
	}

	// Try to parse the private key without a passphrase
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err == nil {
		auths = append(auths, ssh.PublicKeys(signer))
		return auths, nil
	}
	if _, ok := errors.AsType[*ssh.PassphraseMissingError](err); !ok {
		// Parsing failed for another reason
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Key is encrypted
	if len(passphrase) > 0 {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key with passphrase: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
		return auths, nil
	}

	// Defer prompting for passphrase until after other methods (like agent) fail
	auths = append(auths, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		pass, err := passwd.Prompt(fmt.Sprintf("🔐 Please enter the passphrase for private key (%s):", privateKeyFile))
		if err != nil {
			return nil, err
		}
		passBytes := []byte(pass)
		s, err := ssh.ParsePrivateKeyWithPassphrase(privateKey, passBytes)
		for i := range passBytes {
			passBytes[i] = 0
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key with passphrase: %w", err)
		}
		return []ssh.Signer{s}, nil
	}))

	if len(auths) == 0 {
		return nil, errors.New("no authentication methods available")
	}
	return auths, nil
}

func getSSHAgentAuthMethod() (ssh.AuthMethod, error) {
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return nil, nil
	}

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", sshAuthSock)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)

	signers, err := agentClient.Signers()
	if err != nil {
		return nil, fmt.Errorf("failed to get signers from SSH agent: %w", err)
	}

	if len(signers) == 0 {
		return nil, nil
	}

	return ssh.PublicKeysCallback(agentClient.Signers), nil
}
