package setup

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	Indigo    = lipgloss.Color("63")
	HotPink   = lipgloss.Color("205")
	LightGray = lipgloss.Color("244")
	Green     = lipgloss.Color("42")
	Cyan      = lipgloss.Color("51")
	Slate     = lipgloss.Color("238")
	White     = lipgloss.Color("255")
	Amber     = lipgloss.Color("214")
)

type Styles struct {
	Border         lipgloss.Style
	Label          lipgloss.Style
	Title          lipgloss.Style
	Badge          lipgloss.Style
	StepIndicator  lipgloss.Style
	Card           lipgloss.Style
	URLBox         lipgloss.Style
	Help           lipgloss.Style
	HelpKey        lipgloss.Style
	HelpDesc       lipgloss.Style
	Divider        lipgloss.Style
	Info           lipgloss.Style
	Success        lipgloss.Style
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
		Padding(0, 1)
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Indigo).
		Padding(0, 1)
	s.Badge = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Indigo).
		Padding(0, 1)
	s.StepIndicator = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Cyan).
		Padding(0, 1)
	s.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Slate).
		Padding(0, 1)
	s.URLBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Cyan).
		Foreground(White).
		Padding(0, 1)
	s.Help = lipgloss.NewStyle().
		Foreground(LightGray)
	s.HelpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Slate).
		Padding(0, 1)
	s.HelpDesc = lipgloss.NewStyle().
		Foreground(LightGray)
	s.Divider = lipgloss.NewStyle().
		Foreground(Slate)
	s.Info = lipgloss.NewStyle().
		Foreground(Cyan)
	s.Success = lipgloss.NewStyle().
		Foreground(Green)
	s.Err = lipgloss.NewStyle().
		Foreground(HotPink)
	s.ErrText = s.Err.Bold(true)
	s.Focused = lipgloss.NewStyle().
		Foreground(HotPink)
	s.Blurred = lipgloss.NewStyle().
		Foreground(LightGray)
	s.NoStyle = lipgloss.NewStyle()
	s.Button = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(HotPink).
		Padding(0, 3)
	s.InputFocused = lipgloss.NewStyle().
		Foreground(HotPink)
	s.InputUnfocused = lipgloss.NewStyle().
		Foreground(LightGray)
	return s
}

// RenderHelpBar renders keyboard shortcuts with modern pill badges.
func (s *Styles) RenderHelpBar(items ...[2]string) string {
	var parts []string
	for _, it := range items {
		key := s.HelpKey.Render(it[0])
		desc := s.HelpDesc.Render(it[1])
		parts = append(parts, key+" "+desc)
	}
	return strings.Join(parts, "  ")
}
