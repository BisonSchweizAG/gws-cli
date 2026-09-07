package gcloud

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	workstations "cloud.google.com/go/workstations/apiv1"
	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/ssh"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

type tunnel struct {
	headers http.Header
	wsName  string
	wsHost  string
	client  *workstations.Client
}

func TCPTunnelWithPassphrase(ctx context.Context, cfg *types.Config, port int) error {
	sshContext, c, ws, err := setup(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeIt(c)

	t := &tunnel{
		headers: http.Header{},
		wsHost:  ws.GetHost(),
		wsName:  ws.GetName(),
		client:  c,
	}
	go t.refreshAuthToken(ctx)
	t.setAuthToken(ctx)

	p := sshContext.Port
	if port != 0 {
		p = port
	}

	lc := net.ListenConfig{}
	sshAddress := net.JoinHostPort(sshContext.Host, strconv.Itoa(p))
	listener, err := lc.Listen(ctx, "tcp", sshAddress)
	if err != nil {
		log.Logf("🚨 Failed to start TCP listener: %v", err)
		return err
	}
	defer closeIt(listener)

	log.Logf("🕳️ Opening tunnel to %s and listening on local ssh port %d ...", sshContext.GCloud.Name, p)

	var postConnectOnce sync.Once
	postConnectOnce.Do(func() {
		postConnectCommand(ctx, sshContext)
	})

	// Create an error channel to handle errors from goroutines
	errChan := make(chan error, 1)

	// Start accepting connections in a separate goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				clientConn, err := listener.Accept()
				if err != nil {
					if !errors.Is(err, net.ErrClosed) {
						log.Logf("🚨 Failed to accept connection: %v", err)
					}
					continue
				}
				log.Log("🤝 Accepted TCP connection")
				go t.handleConnection(clientConn)
			}
		}
	}()

	if sshContext.KnownHostsFile != "" {
		go func() {
			// Get host key by connecting to the address
			knownHostLines, err := ssh.GetHostKey(sshAddress, cfg.SSHTimeout())
			if err != nil {
				log.Logf("🚨 Error getting host key: %v", err)
				return
			}

			updateKnownHosts(sshContext, knownHostLines)
		}()
	}

	// Wait for either context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
}

func postConnectCommand(ctx context.Context, sshContext *types.Context) {
	if len(sshContext.PostConnectCommand) > 0 {
		go func() {
			log.Logf(">_ Starting post-connect command: %s", strings.Join(sshContext.PostConnectCommand, " "))
			if err := exec.CommandContext(ctx, sshContext.PostConnectCommand[0], sshContext.PostConnectCommand[1:]...).
				Start(); err != nil {
				log.Logf("🚨 Failed to start post-connect command: %v", err)
			}
		}()
	}
}

func updateKnownHosts(sshContext *types.Context, knownHostLines []string) {
	if sshContext.KnownHostsFile == "" {
		return
	}

	f, err := os.ReadFile(sshContext.KnownHostsFile)
	if err != nil {
		log.Logf("🚨 Error reading known_hosts %s file: %v", sshContext.KnownHostsFile, err)
		return
	}

	var newLines []string
	var newHostAddresses []string
	for _, line := range knownHostLines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			newHostAddresses = append(newHostAddresses, parts[0])
		}
	}

	satisfied := make([]bool, len(knownHostLines))
	currentLines := strings.Split(string(f), "\n")
	for _, line := range currentLines {
		line = strings.TrimSpace(line)
		if line == "" {
			// we want to remove empty lines from the file
			continue
		}

		matched := false
		keep := false
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			hostPart := parts[0]
			keyPart := parts[1]
			hosts := strings.SplitSeq(hostPart, ",")

			for h := range hosts {
				for i, addr := range newHostAddresses {
					hostMatch := false
					if h == addr {
						hostMatch = true
					} else if strings.HasPrefix(h, "|1|") {
						subParts := strings.Split(h, "|")
						if len(subParts) == 4 {
							salt, _ := base64.StdEncoding.DecodeString(subParts[2])
							hash, _ := base64.StdEncoding.DecodeString(subParts[3])
							mac := hmac.New(sha1.New, salt)
							_, _ = mac.Write([]byte(addr))
							if hmac.Equal(mac.Sum(nil), hash) {
								hostMatch = true
							}
						}
					}

					if hostMatch {
						matched = true
						newParts := strings.SplitN(knownHostLines[i], " ", 2)
						if len(newParts) == 2 && newParts[1] == keyPart {
							satisfied[i] = true
							keep = true
						}
					}
				}
			}
		}

		if !matched || keep {
			newLines = append(newLines, line)
		}
	}

	for i, line := range knownHostLines {
		if !satisfied[i] {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				newLines = append(newLines, knownhosts.HashHostname(parts[0])+" "+parts[1])
			}
		}
	}

	slices.Sort(newLines)
	newLines = slices.Compact(newLines)

	var filteredCurrentLines []string
	for _, line := range currentLines {
		line = strings.TrimSpace(line)
		if line != "" {
			filteredCurrentLines = append(filteredCurrentLines, line)
		}
	}

	if !slices.Equal(filteredCurrentLines, newLines) {
		err = os.WriteFile(sshContext.KnownHostsFile, []byte(strings.Join(newLines, "\n")), 0o644)
		if err != nil {
			log.Logf("🚨 Error writing known_hosts file: %v", err)
			return
		}
		log.Logf("📝 KnownHosts file %s updated", sshContext.KnownHostsFile)
	}
}

func (t *tunnel) connectWebsocket() (*websocket.Conn, error) {
	wsURL := fmt.Sprintf("wss://%s/_workstation/tcp/%d", t.wsHost, 22)
	// Establish a persistent WebSocket connection
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, t.headers)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			defer closeIt(resp.Body)
			log.Log(string(body))
		}
		log.Logf("🚨 Failed to connect to WebSocket %q: %v", wsURL, err)
		return nil, err
	}
	return conn, nil
}

// handleConnection forwards data between the TCP client and the WebSocket connection.
func (t *tunnel) handleConnection(clientConn net.Conn) {
	wsConn, err := t.connectWebsocket()
	if err != nil {
		return
	}

	// Create a local context to coordinate the shutdown of both goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer closeIt(clientConn)
	defer closeIt(wsConn)

	// Create a goroutine to send data from TCP client to WebSocket
	go func() {
		defer cancel() // Trigger cancel if the TCP client disconnects
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := clientConn.Read(buf)
				if err != nil {
					return // EOF or closed connection is handled by the defer cancel()
				}

				// Send TCP data over WebSocket
				if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// Read data from WebSocket and send to the TCP client
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := wsConn.ReadMessage()
			if err != nil {
				if _, ok := errors.AsType[*websocket.CloseError](err); !ok && !errors.Is(err, net.ErrClosed) {
					log.Logf("🚨 Error reading from WebSocket: %v", err)
				}
				return
			}

			// Send WebSocket data to the TCP client
			_, err = clientConn.Write(msg)
			if err != nil {
				// Prevent logging expected errors when the connection is closed or aborted by the host
				if !errors.Is(err, net.ErrClosed) && !strings.Contains(err.Error(), "wsasend") {
					log.Logf("🚨 Error writing to TCP connection: %v", err)
				}
				return
			}
		}
	}
}

func (t *tunnel) refreshAuthToken(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.setAuthToken(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (t *tunnel) setAuthToken(ctx context.Context) {
	tr, err := t.client.GenerateAccessToken(ctx, &workstationspb.GenerateAccessTokenRequest{Workstation: t.wsName})
	if err != nil {
		log.Logf("🚨 Error generating token: %v", err)
		return
	}
	t.headers["Authorization"] = []string{"Bearer " + tr.GetAccessToken()}
	log.Log("🎫 Got new Tunnel Auth Token")
}

func closeIt(cl io.Closer) {
	_ = cl.Close()
}
