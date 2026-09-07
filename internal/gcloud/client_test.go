package gcloud

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	workstations "cloud.google.com/go/workstations/apiv1"
	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/bisonschweizag/gws-cli/internal/log"
)

type mockWorkstationsServer struct {
	workstationspb.UnimplementedWorkstationsServer
	getWorkstationFunc func(ctx context.Context, req *workstationspb.GetWorkstationRequest) (*workstationspb.Workstation, error)
}

func (m *mockWorkstationsServer) GetWorkstation(
	ctx context.Context,
	req *workstationspb.GetWorkstationRequest,
) (*workstationspb.Workstation, error) {
	if m.getWorkstationFunc != nil {
		return m.getWorkstationFunc(ctx, req)
	}
	return nil, errors.New("unimplemented")
}

func setupTestClient(t *testing.T, srv workstationspb.WorkstationsServer) (*workstations.Client, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	workstationspb.RegisterWorkstationsServer(s, srv)

	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}

	client, err := workstations.NewClient(context.Background(), option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	cleanup := func() {
		_ = client.Close()
		_ = conn.Close()
		s.Stop()
		_ = lis.Close()
	}

	return client, cleanup
}

func Test_waitForWorkstationRunning_TimeoutDoesNotCancelGoogleCall(t *testing.T) {
	prevInterval := pollInterval
	pollInterval = 10 * time.Millisecond
	defer func() { pollInterval = prevInterval }()

	var callCount atomic.Int32
	var ctxCancelledOnCall atomic.Bool

	mockSrv := &mockWorkstationsServer{
		getWorkstationFunc: func(ctx context.Context, req *workstationspb.GetWorkstationRequest) (*workstationspb.Workstation, error) {
			callCount.Add(1)
			if ctx.Err() != nil {
				ctxCancelledOnCall.Store(true)
			}
			return &workstationspb.Workstation{
				Name:  req.GetName(),
				State: workstationspb.Workstation_STATE_STARTING,
			}, nil
		},
	}

	client, cleanup := setupTestClient(t, mockSrv)
	defer cleanup()

	ctx := context.Background()
	ws := &workstationspb.Workstation{
		Name:  "test-workstation",
		State: workstationspb.Workstation_STATE_STARTING,
	}

	timeout := 50 * time.Millisecond
	err := waitForWorkstationRunning(ctx, client, ws, timeout)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	expectedErr := "timeout waiting for workstation test-workstation to start"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	// Verify the parent context is not canceled
	if ctx.Err() != nil {
		t.Fatalf("parent context was canceled: %v", ctx.Err())
	}

	// Verify Google calls did not receive a canceled context
	if ctxCancelledOnCall.Load() {
		t.Fatal("Google API call received a canceled context during/after timeout")
	}

	// Verify that polling actually occurred
	if callCount.Load() == 0 {
		t.Fatal("expected at least one GetWorkstation call")
	}
}

func Test_waitForWorkstationRunning_SuccessWhenRunning(t *testing.T) {
	prevInterval := pollInterval
	pollInterval = 10 * time.Millisecond
	defer func() { pollInterval = prevInterval }()

	var callCount atomic.Int32

	mockSrv := &mockWorkstationsServer{
		getWorkstationFunc: func(_ context.Context, req *workstationspb.GetWorkstationRequest) (*workstationspb.Workstation, error) {
			count := callCount.Add(1)
			state := workstationspb.Workstation_STATE_STARTING
			if count >= 3 {
				state = workstationspb.Workstation_STATE_RUNNING
			}
			return &workstationspb.Workstation{
				Name:  req.GetName(),
				State: state,
			}, nil
		},
	}

	client, cleanup := setupTestClient(t, mockSrv)
	defer cleanup()

	ctx := context.Background()
	ws := &workstationspb.Workstation{
		Name:  "test-workstation",
		State: workstationspb.Workstation_STATE_STARTING,
	}

	timeout := 500 * time.Millisecond
	err := waitForWorkstationRunning(ctx, client, ws, timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.GetState() != workstationspb.Workstation_STATE_RUNNING {
		t.Fatalf("expected workstation state RUNNING, got %v", ws.GetState())
	}
}

func Test_waitForWorkstationRunning_ContextCancellation(t *testing.T) {
	prevInterval := pollInterval
	pollInterval = 50 * time.Millisecond
	defer func() { pollInterval = prevInterval }()

	mockSrv := &mockWorkstationsServer{
		getWorkstationFunc: func(_ context.Context, req *workstationspb.GetWorkstationRequest) (*workstationspb.Workstation, error) {
			return &workstationspb.Workstation{
				Name:  req.GetName(),
				State: workstationspb.Workstation_STATE_STARTING,
			}, nil
		},
	}

	client, cleanup := setupTestClient(t, mockSrv)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	ws := &workstationspb.Workstation{
		Name:  "test-workstation",
		State: workstationspb.Workstation_STATE_STARTING,
	}

	// Cancel context quickly
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := waitForWorkstationRunning(ctx, client, ws, 5*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func Test_waitForWorkstationRunning_TimeoutErrorLogging(t *testing.T) {
	prevInterval := pollInterval
	pollInterval = 10 * time.Millisecond
	defer func() { pollInterval = prevInterval }()

	mockSrv := &mockWorkstationsServer{
		getWorkstationFunc: func(_ context.Context, req *workstationspb.GetWorkstationRequest) (*workstationspb.Workstation, error) {
			return &workstationspb.Workstation{
				Name:  req.GetName(),
				State: workstationspb.Workstation_STATE_STARTING,
			}, nil
		},
	}

	client, cleanup := setupTestClient(t, mockSrv)
	defer cleanup()

	ctx := context.Background()
	ws := &workstationspb.Workstation{
		Name:  "my-ws",
		State: workstationspb.Workstation_STATE_STARTING,
	}

	timeout := 30 * time.Millisecond
	err := waitForWorkstationRunning(ctx, client, ws, timeout)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Capture log when error is logged
	var loggedMessages []string
	log.SetLogger(func(msg string) {
		loggedMessages = append(loggedMessages, msg)
	})

	log.Logf("Error waiting for workstation to start: %v", err)

	expectedMsg := "Error waiting for workstation to start: timeout waiting for workstation my-ws to start"
	if len(loggedMessages) != 1 || loggedMessages[0] != expectedMsg {
		t.Fatalf("expected log message %q, got %v", expectedMsg, loggedMessages)
	}
}
