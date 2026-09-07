package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

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
	logMsg    string
	errMsg    struct{ err error }
	opDoneMsg struct{}
)

func (m *Model) Init() tea.Cmd {
	log.SetLogger(func(log string) {
		select {
		case m.LogChan <- log:
		default:
		}
	})

	return tea.Batch(
		func() tea.Msg {
			err := m.Operation(m.ctx)
			if err != nil {
				return errMsg{err}
			}
			return opDoneMsg{}
		},
		m.waitForLog(),
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
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
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

	var b strings.Builder

	b.WriteString(m.Styles.Title.Render(m.Title))
	b.WriteString("\n\n")

	currCtx := m.Config.CurrentContext()
	fmt.Fprintf(&b, "  Version:     %s\n", version.Version)
	fmt.Fprintf(&b, "  Config File: %s\n", m.Config.FilePath)
	fmt.Fprintf(&b, "  Context:     %s\n", m.Styles.Success.Render(m.Config.CurrentContextName))
	fmt.Fprintf(&b, "  Workstation: %s\n", m.Styles.Success.Render(currCtx.GCloud.Name))
	for _, h := range m.Headers {
		fmt.Fprintf(&b, "  %-13s%s\n", h.Key+":", h.Value)
	}
	b.WriteString("\n")

	if m.Err != nil {
		b.WriteString(m.Styles.ErrText.Render(fmt.Sprintf("Error: %v", m.Err)))
		b.WriteString("\n\n")
	}

	b.WriteString(m.Styles.Info.Render("Logs:"))
	b.WriteString("\n")

	if len(m.Logs) > 0 {
		start := 0
		reservedHeight := 14 + len(m.Headers)
		if m.Err != nil {
			reservedHeight += 2
		}
		availableHeight := max(m.Height-reservedHeight, 5)

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
		b.WriteString("Waiting for logs...")
	}
	b.WriteString("\n\n")

	var help string
	if m.Done {
		help = m.Styles.Help.Render("press any key to quit")
	} else {
		help = m.Styles.Help.Render("ctrl+c: quit")
	}
	b.WriteString(help)

	v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
	v.AltScreen = true
	return v
}
