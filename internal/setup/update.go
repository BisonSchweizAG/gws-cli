package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/log"
)

type (
	loginMsg        struct{}
	projectsMsg     []gcloud.Project
	workstationsMsg []gcloud.Workstation
	errMsg          error
)

func (m Model) Init() tea.Cmd {
	log.SetLogger(func(l string) {
		select {
		case m.LogChan <- l:
		default:
		}
	})

	if m.Config.NoBrowser {
		// If token can already be refreshed without browser interaction, try that first:
		if m.Config.Token != nil && m.Config.Token.Token.RefreshToken != "" {
			return tea.Batch(
				func() tea.Msg {
					_, err := gcloud.Login(context.Background(), m.Config)
					if err != nil {
						return nil
					}
					return loginMsg{}
				},
				m.waitForLog(),
			)
		}
		return m.waitForLog()
	}

	return tea.Batch(
		func() tea.Msg {
			_, err := gcloud.Login(context.Background(), m.Config)
			if err != nil {
				return errMsg(err)
			}
			return loginMsg{}
		},
		m.waitForLog(),
	)
}

func (m Model) waitForLog() tea.Cmd {
	return func() tea.Msg {
		l, ok := <-m.LogChan
		if !ok {
			return nil
		}
		return logMsg(l)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.FilePickerActive {
			var cmd tea.Cmd
			m.Fp, cmd = m.Fp.Update(msg)
			return m, cmd
		}
		return m, nil
	case loginMsg:
		m.Step = stepProject
		m.StatusMessage = "Fetching projects..."
		return m, func() tea.Msg {
			projects, err := gcloud.ListProjects(context.Background(), m.Config)
			if err != nil {
				return errMsg(err)
			}
			return projectsMsg(projects)
		}
	case projectsMsg:
		m.Projects = msg
		if len(msg) == 1 {
			return m.selectProject(msg[0].ID)
		}
		m.Step = stepProject
		m.StatusMessage = ""
		m.applyFilter()
		return m, nil
	case workstationsMsg:
		m.Workstations = msg
		if len(msg) == 1 {
			return m.selectWorkstation(msg[0]), nil
		}
		m.Step = stepConfig
		m.StatusMessage = ""
		m.applyFilter()
		return m, nil
	case logMsg:
		if strings.HasPrefix(string(msg), "Opening URL: ") {
			m.AuthURL = strings.TrimPrefix(string(msg), "Opening URL: ")
			m.Logs = append(m.Logs, "Opening browser...")
			return m, m.waitForLog()
		}
		m.Logs = append(m.Logs, string(msg))
		return m, m.waitForLog()
	case errMsg:
		m.StatusMessage = fmt.Sprintf("Error: %v", msg)
		return m, nil
	}

	if m.Step == stepLogin {
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "ctrl+c", "esc":
				m.Aborted = true
				return m, tea.Quit
			case "enter":
				if m.Config.NoBrowser {
					code := strings.TrimSpace(m.AuthCodeInput.Value())
					if code == "" {
						m.StatusMessage = "Please enter an authorization code"
						return m, nil
					}
					m.StatusMessage = "Exchanging authorization code..."
					return m, func() tea.Msg {
						_, err := gcloud.ExchangeAuthCode(
							context.Background(),
							m.Config,
							code,
							m.CodeVerifier,
							gcloud.OOBRedirectURI,
						)
						if err != nil {
							return errMsg(err)
						}
						return loginMsg{}
					}
				}
			}
		}
		if m.Config.NoBrowser {
			var cmd tea.Cmd
			m.AuthCodeInput, cmd = m.AuthCodeInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.Step == stepProject || m.Step == stepConfig {
		return m.updateList(msg)
	}

	if m.FilePickerActive {
		var cmd tea.Cmd
		m.Fp, cmd = m.Fp.Update(msg)

		if didSelect, path := m.Fp.DidSelectFile(msg); didSelect {
			m.Inputs[m.FilePickerField].SetValue(path)
			m.FilePickerActive = false
			return m, nil
		}
		if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
			m.FilePickerActive = false
			return m, nil
		}

		return m, cmd
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if (m.Focused == PrivateKeyFile || m.Focused == KnownHostsFile) && msg.String() == "ctrl+f" {
			m.FilePickerActive = true
			m.FilePickerField = m.Focused
			m.StatusMessage = ""
			currentFile := m.Inputs[m.Focused].Value()
			if currentFile != "" {
				if info, err := os.Stat(currentFile); err == nil && !info.IsDir() {
					m.Fp.CurrentDirectory = filepath.Dir(currentFile)
				} else if err == nil && info.IsDir() {
					m.Fp.CurrentDirectory = currentFile
				}
			}
			return m, m.Fp.Init()
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Aborted = true
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// check if the user wants to submit
			if s == "enter" && m.Focused == Submit {
				return m.validateAndSubmit()
			}

			// Clear status message on navigation
			m.StatusMessage = ""

			if s == "up" || s == "shift+tab" {
				m.Focused--
			} else {
				m.Focused++
			}

			if m.Focused > Submit {
				m.Focused = 0
			} else if m.Focused < 0 {
				m.Focused = Submit
			}

			for i := range m.Inputs {
				s := m.Inputs[i].Styles()
				if i == int(m.Focused) {
					m.Inputs[i].Focus()
					s.Focused.Prompt = m.Styles.InputFocused
					s.Focused.Text = m.Styles.InputFocused
					s.Cursor.Color = m.Styles.InputFocused.GetForeground()
				} else {
					m.Inputs[i].Blur()
					s.Blurred.Prompt = m.Styles.InputUnfocused
					s.Blurred.Text = m.Styles.InputUnfocused
					s.Cursor.Color = m.Styles.InputUnfocused.GetForeground()
				}
				m.Inputs[i].SetStyles(s)
			}

			return m, nil
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m Model) validateAndSubmit() (tea.Model, tea.Cmd) {
	ctxName := m.Inputs[ContextName].Value()
	portVal, err := strconv.Atoi(m.Inputs[Port].Value())
	if err != nil || portVal < 1000 || portVal > 65535 {
		m.StatusMessage = fmt.Sprintf(
			"Error: Port must be a number between 1001 and 65535. (Current value: %s)",
			m.Inputs[Port].Value(),
		)
		return m, nil
	}

	if m.Config.Contexts != nil {
		for name, ctx := range m.Config.Contexts {
			if ctx.Port == portVal && name != ctxName {
				m.StatusMessage = fmt.Sprintf("Error: Port %d is already used by context %q.", portVal, name)
				return m, nil
			}
		}
	}

	for i := range m.Inputs {
		if m.Inputs[i].Value() == "" && focusable(i) != KnownHostsFile && focusable(i) != GcloudAccount {
			m.StatusMessage = fmt.Sprintf("Error: %s is a required field.", m.Inputs[i].Label)
			return m, nil
		}
	}

	privateKeyFile := m.Inputs[PrivateKeyFile].Value()
	if st, err := os.Stat(privateKeyFile); os.IsNotExist(err) {
		m.StatusMessage = "Error: private key file does not exist: " + privateKeyFile
		return m, nil
	} else if st.IsDir() {
		m.StatusMessage = "Error: private key must not be directory: " + privateKeyFile
		return m, nil
	}

	knownHostsFile := m.Inputs[KnownHostsFile].Value()
	if knownHostsFile != "" {
		if st, err := os.Stat(knownHostsFile); os.IsNotExist(err) {
			m.StatusMessage = "Error: known hosts file does not exist: " + knownHostsFile
			return m, nil
		} else if st.IsDir() {
			m.StatusMessage = "Error: known hosts must not be directory: " + knownHostsFile
			return m, nil
		}
	}

	return m, tea.Quit
}

func (m Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.Inputs))
	for i := range m.Inputs {
		m.Inputs[i].Model, cmds[i] = m.Inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m *Model) applyFilter() {
	filter := strings.ToLower(m.FilterInput.Value())
	m.FilteredItems = nil
	switch m.Step {
	case stepProject:
		for _, p := range m.Projects {
			title := p.ID
			if p.Name != "" {
				title = fmt.Sprintf("%s (%s)", p.Name, p.ID)
			}
			if filter == "" || strings.Contains(strings.ToLower(p.ID), filter) ||
				strings.Contains(strings.ToLower(p.Name), filter) {
				m.FilteredItems = append(m.FilteredItems, listItem{title: title, value: p.ID})
			}
		}
	case stepConfig:
		for _, w := range m.Workstations {
			title := w.Name
			if filter == "" || strings.Contains(strings.ToLower(title), filter) {
				m.FilteredItems = append(m.FilteredItems, listItem{title: title, value: w})
			}
		}
	}

	slices.SortFunc(m.FilteredItems, func(a, b listItem) int {
		return strings.Compare(a.title, b.title)
	})

	if m.ListCursor >= len(m.FilteredItems) {
		m.ListCursor = 0
	}
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Aborted = true
			return m, tea.Quit
		case "up":
			if m.ListCursor > 0 {
				m.ListCursor--
			}
		case "down":
			if m.ListCursor < len(m.FilteredItems)-1 {
				m.ListCursor++
			}
		case "enter":
			if len(m.FilteredItems) > 0 {
				selected := m.FilteredItems[m.ListCursor]
				switch m.Step {
				case stepProject:
					if projectID, ok := selected.value.(string); ok {
						return m.selectProject(projectID)
					}
				case stepConfig:
					if ws, ok := selected.value.(gcloud.Workstation); ok {
						return m.selectWorkstation(ws), nil
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.FilterInput, cmd = m.FilterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m Model) selectWorkstation(ws gcloud.Workstation) Model {
	if m.Inputs[ContextName].Value() == "" {
		m.Inputs[ContextName].SetValue(ws.Name)
	}
	m.Inputs[GcloudProject].SetValue(ws.Project)
	m.Inputs[GcloudRegion].SetValue(ws.Region)
	m.Inputs[GcloudCluster].SetValue(ws.Cluster)
	m.Inputs[GcloudConfig].SetValue(ws.Config)
	m.Inputs[GcloudName].SetValue(ws.Name)
	m.Step = stepForm
	m.StatusMessage = ""
	return m
}

func (m Model) selectProject(projectID string) (Model, tea.Cmd) {
	m.Inputs[GcloudProject].SetValue(projectID)
	m.Step = stepConfig
	m.StatusMessage = "Fetching workstations..."
	m.FilterInput.SetValue("")
	m.ListCursor = 0
	return m, func() tea.Msg {
		workstations, err := gcloud.ListWorkstations(context.Background(), m.Config, projectID)
		if err != nil {
			return errMsg(err)
		}
		return workstationsMsg(workstations)
	}
}
