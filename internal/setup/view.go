package setup

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (i Input) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, i.Style.Render(i.Label), "  "+i.Model.View())
}

func (m Model) View() tea.View {
	frameSize := m.Styles.Border.GetHorizontalFrameSize()
	innerWidth := max(m.Width-4-frameSize, 60)

	var b strings.Builder

	if m.Step == stepLogin {
		stepPill := m.Styles.StepIndicator.Render(" STEP 1/4 ")
		title := m.Styles.Title.Render("Google Cloud Authentication")
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, stepPill, "  ", title))
		b.WriteString("\n")
		b.WriteString(m.Styles.Divider.Render(strings.Repeat("─", max(innerWidth, 40))))
		b.WriteString("\n\n")

		var authCard strings.Builder
		authCard.WriteString(m.Styles.Badge.Background(Indigo).Render(" 🔐 BROWSER LOGIN "))
		authCard.WriteString("\n\n")
		authCard.WriteString(m.Styles.Info.Render("Opening Google Cloud authentication in your default browser..."))
		authCard.WriteString("\n")
		authCard.WriteString(m.Styles.Help.Render("If your browser did not open automatically, copy and open the URL below:"))
		authCard.WriteString("\n\n")

		authBoxWidth := max(innerWidth-6, 40)
		if m.AuthURL != "" {
			authCard.WriteString(m.Styles.URLBox.Width(authBoxWidth).Render(m.AuthURL))
		} else {
			authCard.WriteString(m.Styles.Blurred.Render("Generating authorization URL..."))
		}
		authCard.WriteString("\n\n")

		if m.ClipboardMsg != "" {
			authCard.WriteString(m.Styles.Success.Render(m.ClipboardMsg))
		} else {
			authCard.WriteString(m.Styles.Help.Render("Tip: Press Ctrl+Y to copy URL to clipboard"))
		}
		authCard.WriteString("\n\n")
		authCard.WriteString(m.Styles.Blurred.Render("Waiting for authentication in browser..."))
		authCard.WriteString("\n")

		cardRendered := m.Styles.Card.Width(innerWidth).Render(authCard.String())
		b.WriteString(cardRendered)
		b.WriteString("\n\n")

		if m.StatusMessage != "" {
			b.WriteString(m.Styles.ErrText.Render("🚨 " + m.StatusMessage))
			b.WriteString("\n\n")
		}

		var help string
		if m.AuthURL != "" {
			help = m.Styles.RenderHelpBar(
				[2]string{"Ctrl+Y", "copy URL"},
				[2]string{"Esc", "cancel"},
			)
		} else {
			help = m.Styles.RenderHelpBar([2]string{"Esc", "cancel"})
		}
		b.WriteString(help)
		b.WriteString(m.renderLogs())

		v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
		v.AltScreen = true
		return v
	}

	if m.Step == stepProject || m.Step == stepConfig {
		var stepLabel, titleText string
		if m.Step == stepProject {
			stepLabel = " STEP 2/4 "
			titleText = "Select Google Cloud Project"
		} else {
			stepLabel = " STEP 3/4 "
			titleText = "Select Cloud Workstation"
		}

		stepPill := m.Styles.StepIndicator.Render(stepLabel)
		title := m.Styles.Title.Render(titleText)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, stepPill, "  ", title))
		b.WriteString("\n")
		b.WriteString(m.Styles.Divider.Render(strings.Repeat("─", max(innerWidth, 40))))
		b.WriteString("\n\n")

		b.WriteString(m.Styles.Label.Render("🔍 Filter: "))
		b.WriteString(m.FilterInput.View())
		b.WriteString("\n\n")

		if m.StatusMessage != "" {
			b.WriteString(m.Styles.ErrText.Render("🚨 " + m.StatusMessage))
			b.WriteString("\n\n")
		} else {
			for i, item := range m.FilteredItems {
				if i == m.ListCursor {
					cursor := m.Styles.InputFocused.Bold(true).Render("❯ ")
					text := m.Styles.InputFocused.Bold(true).Render(item.title)
					b.WriteString(cursor + text)
				} else {
					cursor := "  "
					text := m.Styles.InputUnfocused.Render(item.title)
					b.WriteString(cursor + text)
				}
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		help := m.Styles.RenderHelpBar(
			[2]string{"↑/↓", "navigate"},
			[2]string{"Enter", "select"},
			[2]string{"Esc", "quit"},
		)
		b.WriteString(help)
		b.WriteString(m.renderLogs())

		v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
		v.AltScreen = true
		return v
	}

	if m.FilePickerActive {
		stepPill := m.Styles.StepIndicator.Render(" FILE PICKER ")
		title := m.Styles.Title.Render("Select SSH Key or Known Hosts File")
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, stepPill, "  ", title))
		b.WriteString("\n")
		b.WriteString(m.Styles.Divider.Render(strings.Repeat("─", max(innerWidth, 40))))
		b.WriteString("\n\n")

		b.WriteString(m.Fp.View())
		b.WriteString("\n\n")
		help := m.Styles.RenderHelpBar(
			[2]string{"Enter", "select"},
			[2]string{"Backspace", "directory up"},
			[2]string{"Esc", "close"},
		)
		b.WriteString(help)
		b.WriteString(m.renderLogs())

		v := tea.NewView(m.Styles.Border.Width(m.Width - 4).Render(b.String()))
		v.AltScreen = true
		return v
	}

	stepPill := m.Styles.StepIndicator.Render(" STEP 4/4 ")
	title := m.Styles.Title.Render("Configure GWS Context")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, stepPill, "  ", title))
	b.WriteString("\n")
	b.WriteString(m.Styles.Divider.Render(strings.Repeat("─", max(innerWidth, 40))))
	b.WriteString("\n\n")

	for i := range m.Inputs {
		b.WriteString(m.Inputs[i].View())
		b.WriteString("\n")
	}

	var button string
	if m.Focused == Submit {
		button = m.Styles.Button.Render(" ↵ Submit Configuration ")
	} else {
		button = m.Styles.Card.Render(" " + m.Styles.Blurred.Render("Submit Configuration") + " ")
	}
	fmt.Fprintf(&b, "\n%s\n\n", button)

	if m.StatusMessage != "" {
		b.WriteString(m.Styles.ErrText.Render("🚨 " + m.StatusMessage))
		b.WriteString("\n\n")
	}

	var helpItems [][2]string
	helpItems = append(helpItems,
		[2]string{"Tab", "next field"},
		[2]string{"Up/Down", "navigate"},
		[2]string{"Enter", "confirm"},
	)
	if m.Focused == PrivateKeyFile || m.Focused == KnownHostsFile {
		helpItems = append(helpItems, [2]string{"Ctrl+F", "file picker"})
	}
	helpItems = append(helpItems, [2]string{"Esc", "quit"})

	b.WriteString(m.Styles.RenderHelpBar(helpItems...))
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
	logsHeader := fmt.Sprintf("📋 Activity Logs (%d):", len(m.Logs))
	b.WriteString(m.Styles.Label.Padding(0).Render(logsHeader))
	b.WriteString("\n")

	start := 0
	if len(m.Logs) > 5 {
		start = len(m.Logs) - 5
	}
	b.WriteString(strings.Join(m.Logs[start:], "\n"))
	return b.String()
}
