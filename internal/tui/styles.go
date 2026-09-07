package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	Indigo    = lipgloss.Color("63")
	HotPink   = lipgloss.Color("205")
	LightGray = lipgloss.Color("244")
	Green     = lipgloss.Color("42")
)

type Styles struct {
	Border  lipgloss.Style
	Title   lipgloss.Style
	Help    lipgloss.Style
	Err     lipgloss.Style
	ErrText lipgloss.Style
	Info    lipgloss.Style
	Success lipgloss.Style
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
		Padding(0, 2)
	s.Help = lipgloss.NewStyle().
		Foreground(LightGray)
	s.Err = lipgloss.NewStyle().
		Foreground(HotPink)
	s.ErrText = s.Err.Bold(true)
	s.Info = lipgloss.NewStyle().
		Foreground(LightGray)
	s.Success = lipgloss.NewStyle().
		Foreground(Green)
	return s
}
