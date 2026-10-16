package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestTUIView(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": {
				GCloud: &types.GCloud{
					Name: "my-workstation",
				},
				Port: 2222,
			},
		},
	}
	_ = cfg.UseContext("default")

	for _, w := range []int{80, 100, 120} {
		m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
			return nil
		})
		m.Width = w
		m.Height = 30
		m.AddHeader("Local Port", "2222")
		view := m.View()

		// Verify that the view contains version.Logo elements and header
		if !strings.Contains(view.Content, "GWS Tunnel") {
			t.Errorf("View missing title at width %d", w)
		}
		if !strings.Contains(view.Content, "my-workstation") {
			t.Errorf("View missing workstation at width %d", w)
		}

		t.Logf("Width %d rendered view:\n%s\n", w, view.Content)
	}
}

func TestStylesLogo(t *testing.T) {
	styles := DefaultStyles()
	if styles.Logo.GetForeground() != Indigo {
		t.Errorf("Expected Logo style foreground to be %v, got %v", Indigo, styles.Logo.GetForeground())
	}
}

func TestTUIView_AuthURL(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Login", func(context.Context) error {
		return nil
	})
	m.Width = 100
	m.Height = 40
	m.AuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=123"
	m.ClipboardMsg = "✓ Auth URL copied to clipboard!"

	view := m.View()
	if !strings.Contains(view.Content, "AUTHENTICATION REQUIRED") {
		t.Error("View missing AUTHENTICATION REQUIRED card")
	}
	if !strings.Contains(view.Content, "https://accounts.google.com/o/oauth2/auth?client_id=123") {
		t.Error("View missing Auth URL")
	}
	if !strings.Contains(view.Content, "copied to clipboard") {
		t.Error("View missing clipboard notice")
	}
	if !strings.Contains(view.Content, "Ctrl+Y") {
		t.Error("View missing Ctrl+Y help")
	}
}

func TestTUI_ClipboardCopy(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Login", func(context.Context) error {
		return nil
	})
	m.AuthURL = "https://accounts.google.com/o/oauth2/auth?client_id=123"

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected newModel to be *Model, got %T", newModel)
	}
	if !m2.CopiedToClipboard {
		t.Error("expected CopiedToClipboard to be true")
	}
	if !strings.Contains(m2.ClipboardMsg, "copied to clipboard") {
		t.Errorf("unexpected ClipboardMsg: %s", m2.ClipboardMsg)
	}
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd for SetClipboard")
	}
}

func TestTUIView_ShutdownHook(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": {
				StopWorkstationOnExitTunnel: true,
			},
		},
	}
	_ = cfg.UseContext("default")

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.ShutdownHookActive = true

	view := m.View()
	if !strings.Contains(view.Content, "⚡ Stop WS on exit ⚡") {
		t.Error("View missing stop on exit indicator")
	}
}

func TestTUIView_NoShutdownHook(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.ShutdownHookActive = false

	view := m.View()
	if strings.Contains(view.Content, "⚡ Stop WS on exit ⚡") {
		t.Error("View should not contain stop on exit indicator when inactive")
	}
}

func TestTUIView_NewVersionBadge(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.NewVersionAvailable = "v1.2.3"

	view := m.View()
	if !strings.Contains(view.Content, "v1.2.3 available") {
		t.Error("View missing new version badge")
	}
}

func TestTUIView_Error(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.Err = errors.New("connection refused")

	view := m.View()
	if !strings.Contains(view.Content, "Error") {
		t.Error("View missing error message")
	}
	if !strings.Contains(view.Content, "connection refused") {
		t.Error("View missing error details")
	}
}

func TestTUIView_Logs(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.Logs = []string{"Starting tunnel...", "Connected successfully"}
	m.Done = true
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "Starting tunnel...") {
		t.Error("View missing first log")
	}
	if !strings.Contains(view.Content, "Connected successfully") {
		t.Error("View missing second log")
	}
	if !strings.Contains(view.Content, "Activity Logs (2)") {
		t.Error("View missing log count header")
	}
}

func TestTUIView_Quitting(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.Quitting = true

	view := m.View()
	if view.Content != "" {
		t.Error("View should be empty when quitting")
	}
}

func TestTUIView_DoneState(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.Done = true
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "Any key") {
		t.Error("View should show 'Any key' help when done")
	}
}

func TestTUIView_WaitingForLogs(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "Waiting for logs...") {
		t.Error("View should show waiting message when no logs")
	}
}

func TestTUIView_DefaultHelp(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "toggle stop on exit") {
		t.Error("View should show stop on exit toggle help by default")
	}
	if !strings.Contains(view.Content, "Ctrl+C") {
		t.Error("View should show Ctrl+C help")
	}
}

func TestUpdate_LogMsg(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	newModel, cmd := m.Update(logMsg("test log message"))
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if len(m2.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(m2.Logs))
	}
	if m2.Logs[0] != "test log message" {
		t.Errorf("expected 'test log message', got %q", m2.Logs[0])
	}
	if cmd == nil {
		t.Error("expected non-nil command after log message")
	}
}

func TestUpdate_StopSpinner(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	newModel, _ := m.Update(logMsg(StopSpinner))
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if !m2.SpinnerStopped {
		t.Error("expected SpinnerStopped to be true")
	}
}

func TestUpdate_ErrMsg(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	testErr := errors.New("something went wrong")
	newModel, cmd := m.Update(errMsg{err: testErr})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if m2.Err == nil {
		t.Error("expected error to be set")
	}
	if !m2.Done {
		t.Error("expected Done to be true after error")
	}
	if cmd != nil {
		t.Error("expected nil command after error")
	}
}

func TestUpdate_OpDoneMsg(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	newModel, cmd := m.Update(opDoneMsg{})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if !m2.Done {
		t.Error("expected Done to be true")
	}
	if cmd != nil {
		t.Error("expected nil command after op done (no auto quit)")
	}
}

func TestUpdate_OpDoneMsg_AutoQuit(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.AutoQuit = true

	_, cmd := m.Update(opDoneMsg{})
	if cmd == nil {
		t.Error("expected tea.Quit command with AutoQuit enabled")
	}
}

func TestUpdate_WindowSize(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 50})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if m2.Width != 150 {
		t.Errorf("expected width 150, got %d", m2.Width)
	}
	if m2.Height != 50 {
		t.Errorf("expected height 50, got %d", m2.Height)
	}
}

func TestUpdate_NewVersionAvailable(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	newModel, _ := m.Update(newVersionAvailableMsg{latestVersion: "v2.0.0"})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if m2.NewVersionAvailable != "v2.0.0" {
		t.Errorf("expected 'v2.0.0', got %q", m2.NewVersionAvailable)
	}
}

func TestUpdate_ShutdownToggle(t *testing.T) {
	ctx := &types.Context{
		StopWorkstationOnExitTunnel: false,
	}
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": ctx,
		},
	}
	_ = cfg.UseContext("default")

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	// Press 't' to toggle stop on exit on
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 't'})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if !m2.ShutdownHookActive {
		t.Error("expected ShutdownHookActive to be true after toggle")
	}
	if !ctx.StopWorkstationOnExitTunnel {
		t.Error("expected context StopWorkstationOnExitTunnel to be true")
	}

	// Press 't' again to toggle off
	newModel, _ = m2.Update(tea.KeyPressMsg{Code: 't'})
	m3, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if m3.ShutdownHookActive {
		t.Error("expected ShutdownHookActive to be false after second toggle")
	}
	if ctx.StopWorkstationOnExitTunnel {
		t.Error("expected context StopWorkstationOnExitTunnel to be false")
	}
}

func TestUpdate_CtrlC(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if !m2.Quitting {
		t.Error("expected Quitting to be true after Ctrl+C")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command after Ctrl+C")
	}
}

func TestUpdate_Done_KeyPress(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Done = true

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd == nil {
		t.Error("expected tea.Quit command when pressing any key in done state")
	}
}

func TestUpdate_AuthURLMsg(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Login", func(context.Context) error {
		return nil
	})

	newModel, cmds := m.Update(authURLMsg("https://accounts.google.com/auth"))
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatalf("expected *Model, got %T", newModel)
	}
	if m2.AuthURL != "https://accounts.google.com/auth" {
		t.Errorf("expected AuthURL to be set, got %q", m2.AuthURL)
	}
	if !m2.CopiedToClipboard {
		t.Error("expected CopiedToClipboard to be true")
	}
	if cmds == nil {
		t.Error("expected non-nil commands after auth URL message")
	}
}

func TestLastLog(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	if m.LastLog() != "" {
		t.Error("expected empty last log when no logs")
	}

	m.Logs = []string{"first", "second", "third"}
	if m.LastLog() != "third" {
		t.Errorf("expected 'third', got %q", m.LastLog())
	}
}

func TestAddHeader(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	m.AddHeader("Port", "2222").AddHeader("Host", "localhost")

	if len(m.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(m.Headers))
	}
	if m.Headers[0].Key != "Port" || m.Headers[0].Value != "2222" {
		t.Errorf("unexpected first header: %+v", m.Headers[0])
	}
	if m.Headers[1].Key != "Host" || m.Headers[1].Value != "localhost" {
		t.Errorf("unexpected second header: %+v", m.Headers[1])
	}
}

func TestAddHeaderInView(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.AddHeader("Local Port", "2222")
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "Local Port") {
		t.Error("View missing header key")
	}
	if !strings.Contains(view.Content, "2222") {
		t.Error("View missing header value")
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	if styles.Logo.GetForeground() != Indigo {
		t.Errorf("Logo foreground expected %v, got %v", Indigo, styles.Logo.GetForeground())
	}
	if styles.Title.GetForeground() != Indigo {
		t.Errorf("Title foreground expected %v, got %v", Indigo, styles.Title.GetForeground())
	}
	if styles.Success.GetForeground() != Green {
		t.Errorf("Success foreground expected %v, got %v", Green, styles.Success.GetForeground())
	}
	if styles.Info.GetForeground() != Cyan {
		t.Errorf("Info foreground expected %v, got %v", Cyan, styles.Info.GetForeground())
	}
}

func TestRenderHelpBar(t *testing.T) {
	styles := DefaultStyles()

	help := styles.RenderHelpBar(
		[2]string{"Ctrl+C", "quit"},
		[2]string{"t", "toggle"},
	)

	if !strings.Contains(help, "Ctrl+C") {
		t.Error("help bar missing Ctrl+C")
	}
	if !strings.Contains(help, "quit") {
		t.Error("help bar missing 'quit'")
	}
	if !strings.Contains(help, "t") {
		t.Error("help bar missing 't'")
	}
	if !strings.Contains(help, "toggle") {
		t.Error("help bar missing 'toggle'")
	}
}

func TestNewModel(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	operationCalled := false
	operation := func(_ context.Context) error {
		operationCalled = true
		return nil
	}

	m := NewModel(context.Background(), cfg, ">_ Test Title", operation)

	if m.Title != ">_ Test Title" {
		t.Errorf("expected title '>_ Test Title', got %q", m.Title)
	}
	if m.Config != cfg {
		t.Error("expected config to be set")
	}
	if m.Styles == nil {
		t.Error("expected styles to be initialized")
	}
	if m.LogChan == nil {
		t.Error("expected LogChan to be initialized")
	}
	if m.Operation == nil {
		t.Error("expected Operation to be set")
	}
	_ = operationCalled
}

func TestTUIView_ContextAndWorkstation(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "production",
		Contexts: map[string]*types.Context{
			"production": {
				GCloud: &types.GCloud{
					Name: "prod-ws",
				},
				Port: 2222,
			},
		},
	}
	_ = cfg.UseContext("production")

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "production") {
		t.Error("View missing context name")
	}
	if !strings.Contains(view.Content, "prod-ws") {
		t.Error("View missing workstation name")
	}
}

func TestTUIView_NoConfig(t *testing.T) {
	m := NewModel(context.Background(), nil, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	view := m.View()
	if !strings.Contains(view.Content, "GWS Tunnel") {
		t.Error("View should render title even without config")
	}
}

func TestTUIView_AuthURL_HelpBar(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Login", func(context.Context) error {
		return nil
	})
	m.Width = 100
	m.Height = 40
	m.AuthURL = "https://accounts.google.com/auth"

	view := m.View()
	if !strings.Contains(view.Content, "Ctrl+Y") {
		t.Error("View should show Ctrl+Y help when auth URL is present")
	}
	if !strings.Contains(view.Content, "copy URL") {
		t.Error("View should show 'copy URL' help text")
	}
}

func TestUpdate_ShutdownToggle_Logs(t *testing.T) {
	ctx := &types.Context{
		StopWorkstationOnExitTunnel: false,
	}
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": ctx,
		},
	}
	_ = cfg.UseContext("default")

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	// Toggle on
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 't'})
	m2, ok := newModel.(*Model)
	if !ok {
		t.Fatal("expected *Model after toggle on")
	}

	found := false
	for _, log := range m2.Logs {
		if strings.Contains(log, "Stop on exit enabled") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log message about stop on exit being enabled")
	}

	// Toggle off
	newModel, _ = m2.Update(tea.KeyPressMsg{Code: 't'})
	m3, ok := newModel.(*Model)
	if !ok {
		t.Fatal("expected *Model after toggle off")
	}

	found = false
	for _, log := range m3.Logs {
		if strings.Contains(log, "Stop on exit disabled") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log message about stop on exit being disabled")
	}
}

func TestTUIView_MultipleHeaders(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	for i := range 5 {
		m.AddHeader(fmt.Sprintf("Header%d", i), fmt.Sprintf("Value%d", i))
	}

	view := m.View()
	for i := range 5 {
		if !strings.Contains(view.Content, fmt.Sprintf("Header%d", i)) {
			t.Errorf("View missing header %d", i)
		}
		if !strings.Contains(view.Content, fmt.Sprintf("Value%d", i)) {
			t.Errorf("View missing value %d", i)
		}
	}
}

func TestUpdate_LogMsg_Multiple(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})

	logs := []string{"log 1", "log 2", "log 3"}
	for _, l := range logs {
		var cmd tea.Cmd
		m2, cmd := m.Update(logMsg(l))
		var ok bool
		m, ok = m2.(*Model)
		if !ok {
			t.Fatal("expected *Model after log message")
		}
		if cmd == nil {
			t.Error("expected non-nil command after log message")
		}
	}

	if len(m.Logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(m.Logs))
	}
	for i, expected := range logs {
		if m.Logs[i] != expected {
			t.Errorf("log %d: expected %q, got %q", i, expected, m.Logs[i])
		}
	}
}

func TestShutdownHookStyle(t *testing.T) {
	styles := DefaultStyles()

	rendered := styles.ShutdownHook.Render("test")
	if rendered == "" {
		t.Error("ShutdownHook style should render non-empty string")
	}
}

// TestTUIView_Visual_UpdateAndShutdownHook verifies the rendered view when both
// a new version is available and the stop on exit is active.
func TestTUIView_Visual_UpdateAndShutdownHook(t *testing.T) {
	ctx := &types.Context{
		StopWorkstationOnExitTunnel: true,
	}
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": ctx,
		},
	}
	_ = cfg.UseContext("default")

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.ShutdownHookActive = true
	m.NewVersionAvailable = "v9.9.9"
	m.SpinnerStopped = true

	// Simulate receiving the new version available message
	newModel, _ := m.Update(newVersionAvailableMsg{latestVersion: "v9.9.9"})
	m, ok := newModel.(*Model)
	if !ok {
		t.Fatal("expected *Model after newVersionAvailableMsg")
	}

	view := m.View()

	// Verify stop on exit indicator is visible in header
	if !strings.Contains(view.Content, "⚡ Stop WS on exit ⚡") {
		t.Error("View should contain '⚡ Stop WS on exit ⚡' when stop on exit is active")
	}

	// Verify new version badge is visible
	if !strings.Contains(view.Content, "v9.9.9 available") {
		t.Error("View should contain new version badge 'v9.9.9 available'")
	}

	// Verify both appear together with other header elements
	if !strings.Contains(view.Content, "Version:") {
		t.Error("View should contain Version line")
	}
	if !strings.Contains(view.Content, "Context:") {
		t.Error("View should contain Context line")
	}

	// Verify help bar shows toggle stop on exit
	if !strings.Contains(view.Content, "toggle stop on exit") {
		t.Error("View should show stop on exit toggle in help bar")
	}

	t.Logf("Combined view output:\n%s", view.Content)
}

// TestTUIView_Visual_ShutdownToggleSequence verifies the full toggle sequence:
// start inactive → toggle on → view shows indicator → toggle off → view hides indicator.
func TestTUIView_Visual_ShutdownToggleSequence(t *testing.T) {
	ctx := &types.Context{
		StopWorkstationOnExitTunnel: false,
	}
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
		Contexts: map[string]*types.Context{
			"default": ctx,
		},
	}
	_ = cfg.UseContext("default")

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	// Step 1: Initially inactive — view must NOT show stop on exit badge
	view := m.View()
	if strings.Contains(view.Content, "⚡ Stop WS on exit ⚡") {
		t.Error("Step 1: view should NOT show stop on exit badge initially")
	}

	// Step 2: Toggle on via 't' key press
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 't'})
	m, ok := newModel.(*Model)
	if !ok {
		t.Fatal("expected *Model after toggle on")
	}
	if !m.ShutdownHookActive {
		t.Fatal("Step 2: stop on exit should be active after toggle")
	}

	view = m.View()
	if !strings.Contains(view.Content, "⚡ Stop WS on exit ⚡") {
		t.Error("Step 2: view should show stop on exit badge after toggle on")
	}

	// Step 3: Toggle off via 't' key press
	newModel, _ = m.Update(tea.KeyPressMsg{Code: 't'})
	m, ok = newModel.(*Model)
	if !ok {
		t.Fatal("expected *Model after toggle off")
	}
	if m.ShutdownHookActive {
		t.Fatal("Step 3: stop on exit should be inactive after second toggle")
	}

	view = m.View()
	if strings.Contains(view.Content, "⚡ Stop WS on exit ⚡") {
		t.Error("Step 3: view should NOT show stop on exit badge after toggle off")
	}
}

// TestTUIView_Visual_NewVersionSequence verifies receiving the new version message
// and seeing the badge appear in the view.
func TestTUIView_Visual_NewVersionSequence(t *testing.T) {
	cfg := &types.Config{
		FilePath:           "/path/to/config.yaml",
		CurrentContextName: "default",
	}

	m := NewModel(context.Background(), cfg, ">_ GWS Tunnel", func(context.Context) error {
		return nil
	})
	m.Width = 120
	m.Height = 30
	m.SpinnerStopped = true

	// Initially no version badge
	view := m.View()
	if strings.Contains(view.Content, "v3.0.0 available") {
		t.Error("view should NOT show version badge initially")
	}

	// Receive new version message
	newModel, _ := m.Update(newVersionAvailableMsg{latestVersion: "v3.0.0"})
	var ok bool
	m, ok = newModel.(*Model)
	if !ok {
		t.Fatal("expected *Model after newVersionAvailableMsg")
	}

	view = m.View()
	if !strings.Contains(view.Content, "v3.0.0 available") {
		t.Error("view should show version badge after receiving newVersionAvailableMsg")
	}
}
