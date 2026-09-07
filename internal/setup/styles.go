package setup

import "charm.land/lipgloss/v2"

var (
	Indigo    = lipgloss.Color("63")
	HotPink   = lipgloss.Color("205")
	LightGray = lipgloss.Color("244")
	Green     = lipgloss.Color("42")
)

type Styles struct {
	Border         lipgloss.Style
	Label          lipgloss.Style
	Title          lipgloss.Style
	Help           lipgloss.Style
	Err            lipgloss.Style
	ErrText        lipgloss.Style
	Focused        lipgloss.Style
	Blurred        lipgloss.Style
	NoStyle        lipgloss.Style
	Button         lipgloss.Style
	InputFocused   lipgloss.Style
	InputUnfocused lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Indigo).
		Padding(1, 2)
	s.Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(Green).
		Padding(0, 2)
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Indigo).
		Padding(0, 2)
	s.Help = lipgloss.NewStyle().
		Foreground(LightGray)
	s.Err = lipgloss.NewStyle().
		Foreground(HotPink)
	s.ErrText = s.Err.Bold(true)
	s.Focused = lipgloss.NewStyle().
		Foreground(HotPink)
	s.Blurred = lipgloss.NewStyle().
		Foreground(LightGray)
	s.NoStyle = lipgloss.NewStyle()
	s.Button = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(HotPink).
		Padding(0, 3)
	s.InputFocused = lipgloss.NewStyle().
		Foreground(HotPink)
	s.InputUnfocused = lipgloss.NewStyle()
	return s
}
