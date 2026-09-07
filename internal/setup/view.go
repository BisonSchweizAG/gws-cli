package setup

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (i Input) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, i.Style.Render(i.Label), i.Model.View())
}

func (m Model) View() tea.View {
	var b strings.Builder
	if m.Step == stepLogin {
		b.WriteString(m.Styles.Title.Render(">_ Google Cloud Login"))
		b.WriteString("\n\n")
		b.WriteString("Please login in your browser...")
		b.WriteString("\n\n")
		if m.StatusMessage != "" {
			b.WriteString(m.Styles.ErrText.Render(m.StatusMessage))
			b.WriteString("\n\n")
		}
		b.WriteString(m.Styles.Help.Render("esc: quit"))
		b.WriteString(m.renderLogs())
		v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
		v.AltScreen = true
		return v
	}

	if m.Step == stepProject || m.Step == stepConfig {
		title := "Select Google Project"
		if m.Step == stepConfig {
			title = "Select Workstation"
		}
		b.WriteString(m.Styles.Title.Render(">_ " + title))
		b.WriteString("\n\n")
		b.WriteString(m.FilterInput.View())
		b.WriteString("\n\n")

		if m.StatusMessage != "" {
			b.WriteString(m.Styles.ErrText.Render(m.StatusMessage))
			b.WriteString("\n\n")
		} else {
			for i, item := range m.FilteredItems {
				cursor := "  "
				if i == m.ListCursor {
					cursor = "> "
					b.WriteString(m.Styles.InputFocused.Render(cursor + item.title))
				} else {
					b.WriteString(m.Styles.InputUnfocused.Render(cursor + item.title))
				}
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(m.Styles.Help.Render("up/down: select / enter: confirm / esc: quit"))
		b.WriteString(m.renderLogs())
		v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
		v.AltScreen = true
		return v
	}

	if m.FilePickerActive {
		help := m.Styles.Help.Render("enter: select / backspace: directory up / esc: quit")
		fpView := lipgloss.JoinVertical(lipgloss.Left, m.Fp.View(), help, m.renderLogs())
		v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(fpView))
		v.AltScreen = true
		return v
	}

	b.WriteString(m.Styles.Title.Render(">_ Modify the gws context"))
	b.WriteString("\n\n")

	for i := range m.Inputs {
		b.WriteString(m.Inputs[i].View())
		b.WriteString("\n")
	}

	var button string
	if m.Focused == Submit {
		button = m.Styles.Button.Render("[ Submit ]")
	} else {
		button = fmt.Sprintf("[ %s ]", m.Styles.Blurred.Render("Submit"))
	}
	fmt.Fprintf(&b, "\n%s\n\n", button)

	if m.StatusMessage != "" {
		b.WriteString(m.Styles.ErrText.Render(m.StatusMessage))
		b.WriteString("\n\n")
	}

	help := m.Help
	if m.Focused == PrivateKeyFile || m.Focused == KnownHostsFile {
		help += " / ctrl+f: file picker"
	}
	b.WriteString(m.Styles.Help.Render(help))
	b.WriteString(m.renderLogs())

	v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
	v.AltScreen = true
	return v
}

func (m Model) renderLogs() string {
	if len(m.Logs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(m.Styles.Label.Padding(0).Render("Logs:"))
	b.WriteString("\n")

	start := 0
	if len(m.Logs) > 5 {
		start = len(m.Logs) - 5
	}
	b.WriteString(strings.Join(m.Logs[start:], "\n"))
	return b.String()
}
