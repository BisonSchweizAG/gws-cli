package tui

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
	Muted     = lipgloss.Color("240")
	Slate     = lipgloss.Color("238")
	White     = lipgloss.Color("255")
	Amber     = lipgloss.Color("214")
)

type Styles struct {
	Border     lipgloss.Style
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Badge      lipgloss.Style
	Help       lipgloss.Style
	HelpKey    lipgloss.Style
	HelpDesc   lipgloss.Style
	Err        lipgloss.Style
	ErrText    lipgloss.Style
	Info       lipgloss.Style
	Success    lipgloss.Style
	Logo       lipgloss.Style
	Card       lipgloss.Style
	URLBox     lipgloss.Style
	Label      lipgloss.Style
	Value      lipgloss.Style
	Divider    lipgloss.Style
	LogLine    lipgloss.Style
	StatusPill lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Indigo).
		Padding(1, 2)
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Indigo).
		Padding(0, 1)
	s.Subtitle = lipgloss.NewStyle().
		Foreground(LightGray).
		Italic(true)
	s.Badge = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Indigo).
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
	s.Err = lipgloss.NewStyle().
		Foreground(HotPink)
	s.ErrText = s.Err.Bold(true)
	s.Info = lipgloss.NewStyle().
		Foreground(Cyan)
	s.Success = lipgloss.NewStyle().
		Foreground(Green)
	s.Logo = lipgloss.NewStyle().
		Foreground(Indigo)
	s.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Slate).
		Padding(0, 1)
	s.URLBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Cyan).
		Foreground(White).
		Padding(0, 1)
	s.Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(LightGray)
	s.Value = lipgloss.NewStyle().
		Foreground(White)
	s.Divider = lipgloss.NewStyle().
		Foreground(Slate)
	s.LogLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))
	s.StatusPill = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Green).
		Padding(0, 1)
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
