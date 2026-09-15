package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
	"github.com/bisonschweizag/gws-cli/version"
)

const (
	// StopSpinner is a special log message that tells the TUI to stop the spinner.
	StopSpinner = "\x00stop-spinner"
)

type Model struct {
	Title    string
	Config   *types.Config
	Headers  []HeaderField
	Styles   *Styles
	Width    int
	Height   int
	Logs     []string
	Err      error
	Quitting bool
	Done     bool
	AutoQuit bool
	Spinner  spinner.Model

	SpinnerStopped bool

	// Auth URL and Clipboard support
	AuthURL           string
	CopiedToClipboard bool
	ClipboardMsg      string

	ctx    context.Context //nolint:containedctx
	cancel context.CancelFunc

	LogChan chan string

	Operation func(context.Context) error
}

type HeaderField struct {
	Key   string
	Value string
}

func NewModel(ctx context.Context, cfg *types.Config, title string, operation func(context.Context) error) *Model {
	c, cancel := context.WithCancel(ctx)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(HotPink)

	return &Model{
		Config:    cfg,
		Title:     title,
		Styles:    DefaultStyles(),
		ctx:       c,
		cancel:    cancel,
		LogChan:   make(chan string, 100),
		Operation: operation,
		Spinner:   s,
	}
}

func (m *Model) AddHeader(key, value string) *Model {
	m.Headers = append(m.Headers, HeaderField{Key: key, Value: value})
	return m
}

// LastLog returns the last log message.
func (m *Model) LastLog() string {
	if len(m.Logs) > 0 {
		return m.Logs[len(m.Logs)-1]
	}
	return ""
}

type (
	logMsg     string
	authURLMsg string
	errMsg     struct{ err error }
	opDoneMsg  struct{}
)

func (m *Model) Init() tea.Cmd {
	log.SetLogger(func(l string) {
		select {
		case m.LogChan <- l:
		default:
		}
	})

	if m.Config != nil {
		m.Config.InitAuthChannels()
	}

	return tea.Batch(
		func() tea.Msg {
			err := m.Operation(m.ctx)
			if err != nil {
				return errMsg{err}
			}
			return opDoneMsg{}
		},
		m.waitForLog(),
		m.waitForAuthURL(),
		m.Spinner.Tick,
	)
}

func (m *Model) waitForLog() tea.Cmd {
	return func() tea.Msg {
		l, ok := <-m.LogChan
		if !ok {
			return nil
		}
		return logMsg(l)
	}
}

func (m *Model) waitForAuthURL() tea.Cmd {
	return func() tea.Msg {
		if m.Config == nil || m.Config.AuthURLChan == nil {
			return nil
		}
		url, ok := <-m.Config.AuthURLChan
		if !ok {
			return nil
		}
		return authURLMsg(url)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.Quitting = true
			m.cancel()
			return m, tea.Quit
		}
		if m.Done {
			return m, tea.Quit
		}
		if m.AuthURL != "" && msg.String() == "ctrl+y" {
			_ = clipboard.WriteAll(m.AuthURL)
			m.CopiedToClipboard = true
			m.ClipboardMsg = "✓ Auth URL copied to clipboard!"
			return m, tea.SetClipboard(m.AuthURL)
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case authURLMsg:
		m.AuthURL = string(msg)
		var cmds []tea.Cmd
		cmds = append(cmds, m.waitForAuthURL())
		_ = clipboard.WriteAll(m.AuthURL)
		m.CopiedToClipboard = true
		m.ClipboardMsg = "✓ Auth URL automatically copied to clipboard!"
		cmds = append(cmds, tea.SetClipboard(m.AuthURL))
		return m, tea.Batch(cmds...)
	case logMsg:
		if string(msg) == StopSpinner {
			m.SpinnerStopped = true
			cmd := m.waitForLog()
			return m, cmd
		}
		m.Logs = append(m.Logs, string(msg))
		cmd := m.waitForLog()
		return m, cmd
	case errMsg:
		m.Err = msg.err
		m.Done = true
		return m, nil
	case opDoneMsg:
		m.Done = true
		if m.AutoQuit {
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() tea.View {
	if m.Quitting {
		return tea.NewView("")
	}

	frameSize := m.Styles.Border.GetHorizontalFrameSize()
	innerWidth := max(m.Width-4-frameSize, 60)

	var left strings.Builder
	badge := m.Styles.Badge.Render(" GWS ")
	title := m.Styles.Title.Render(m.Title)
	left.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, badge, " ", title))
	left.WriteString("\n\n")

	var currCtx *types.Context
	if m.Config != nil {
		currCtx = m.Config.CurrentContext()
	}
	fmt.Fprintf(&left, "  Version:     %s\n", m.Styles.Info.Render(version.Version))
	if m.Config != nil && m.Config.FilePath != "" {
		fmt.Fprintf(&left, "  Config File: %s\n", m.Config.FilePath)
	}
	if m.Config != nil && m.Config.CurrentContextName != "" {
		fmt.Fprintf(&left, "  Context:     %s\n", m.Styles.Success.Render(m.Config.CurrentContextName))
	}
	if currCtx != nil && currCtx.GCloud != nil {
		fmt.Fprintf(&left, "  Workstation: %s\n", m.Styles.Success.Render(currCtx.GCloud.Name))
	}
	for _, h := range m.Headers {
		fmt.Fprintf(&left, "  %-13s%s\n", h.Key+":", h.Value)
	}

	logo := m.Styles.Logo.Render(strings.TrimRight(version.Logo, "\n"))
	logoWidth := lipgloss.Width(logo)

	leftContentWidth := lipgloss.Width(left.String())

	var header string
	if innerWidth > leftContentWidth+logoWidth {
		leftWidth := innerWidth - logoWidth
		leftStyled := lipgloss.NewStyle().Width(leftWidth).Render(left.String())
		header = lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, logo)
	} else {
		header = lipgloss.JoinHorizontal(lipgloss.Top, left.String(), "  ", logo)
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")

	divider := m.Styles.Divider.Render(strings.Repeat("─", max(innerWidth, 40)))
	b.WriteString(divider)
	b.WriteString("\n\n")

	if m.AuthURL != "" {
		var authCard strings.Builder
		authCard.WriteString(m.Styles.Badge.Background(Indigo).Render(" 🔐 AUTHENTICATION REQUIRED "))
		authCard.WriteString("\n\n")
		authCard.WriteString(m.Styles.Info.Render("Opening Google Cloud authentication in your default browser..."))
		authCard.WriteString("\n")
		authCard.WriteString(m.Styles.Help.Render("If your browser did not open automatically, copy and open the link below:"))
		authCard.WriteString("\n\n")

		authBoxWidth := max(innerWidth-6, 40)
		urlContent := m.Styles.URLBox.Width(authBoxWidth).Render(m.AuthURL)
		authCard.WriteString(urlContent)
		authCard.WriteString("\n\n")

		if m.ClipboardMsg != "" {
			authCard.WriteString(m.Styles.Success.Render(m.ClipboardMsg))
		} else {
			authCard.WriteString(m.Styles.Help.Render("Tip: Press Ctrl+Y to copy URL to clipboard"))
		}
		authCard.WriteString("\n\n")

		authCard.WriteString(m.Styles.Help.Render("Waiting for authentication in browser..."))
		authCard.WriteString("\n")

		cardRendered := m.Styles.Card.Width(innerWidth).Render(authCard.String())
		b.WriteString(cardRendered)
		b.WriteString("\n\n")
	}

	if m.Err != nil {
		b.WriteString(m.Styles.ErrText.Render(fmt.Sprintf("🚨 Error: %v", m.Err)))
		b.WriteString("\n\n")
	}

	logsHeader := fmt.Sprintf("📋 Activity Logs (%d)", len(m.Logs))
	b.WriteString(m.Styles.Info.Bold(true).Render(logsHeader))
	b.WriteString("\n")

	if len(m.Logs) > 0 {
		start := 0
		reservedHeight := max(10, 7+len(m.Headers)) + 8
		if m.AuthURL != "" {
			reservedHeight += 11
		}
		if m.Err != nil {
			reservedHeight += 2
		}
		availableHeight := max(m.Height-reservedHeight, 4)

		if len(m.Logs) > availableHeight {
			start = len(m.Logs) - availableHeight
		}

		lines := m.Logs[start:]
		lastIndex := len(lines) - 1
		for i, line := range lines {
			if i == lastIndex && !m.Done && !m.SpinnerStopped {
				b.WriteString(m.Spinner.View())
				b.WriteString(" ")
			}
			b.WriteString(line)
			if i < lastIndex {
				b.WriteString("\n")
			}
		}
	} else {
		if !m.Done && !m.SpinnerStopped {
			b.WriteString(m.Spinner.View())
			b.WriteString(" ")
		}
		b.WriteString(m.Styles.Help.Render("Waiting for logs..."))
	}
	b.WriteString("\n\n")

	var help string
	switch {
	case m.AuthURL != "":
		help = m.Styles.RenderHelpBar(
			[2]string{"Ctrl+Y", "copy URL"},
			[2]string{"Ctrl+C", "quit"},
		)
	case m.Done:
		help = m.Styles.RenderHelpBar([2]string{"Any key", "quit"})
	default:
		help = m.Styles.RenderHelpBar([2]string{"Ctrl+C", "quit"})
	}
	b.WriteString(help)

	v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
	v.AltScreen = true
	return v
}
