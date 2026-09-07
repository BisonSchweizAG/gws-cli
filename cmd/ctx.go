package cmd

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

var flagCurrent bool

// ctxCmd represents the ctx command.
var ctxCmd = &cobra.Command{
	Use:   "ctx",
	Short: "Set the active context",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig()
		if err != nil {
			return err
		}

		if flagCurrent {
			cmd.Printf("Current selected context: %q\n", cfg.CurrentContextName)
			return nil
		}

		if len(args) == 1 {
			return cfg.SwitchContext(args[0], false)
		}
		selected, err := selectContext(cfg)
		if err != nil {
			return err
		}

		if selected != "" {
			cmd.Printf("Switching to context %q\n", selected)
			return cfg.SwitchContext(selected, false)
		}

		return nil
	},
}

func init() {
	ctxCmd.PersistentFlags().BoolVar(&flagCurrent, "current", false, "Print the current active context")
}

func selectContext(cfg *types.Config) (string, error) {
	m := &ctxModel{choices: slices.Sorted(maps.Keys(cfg.Contexts))}
	m.cursor = slices.Index(m.choices, cfg.CurrentContextName)

	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		return "", err
	}
	return m.selected, nil
}

func init() {
	rootCmd.AddCommand(ctxCmd)
}

type ctxModel struct {
	cursor   int
	choices  []string
	selected string
}

func (*ctxModel) Init() tea.Cmd {
	return nil
}

func (m *ctxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msgType, ok := msg.(tea.KeyPressMsg); ok {
		switch msgType.String() {
		case "q":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		default:
		}
	}
	return m, nil
}

func (m *ctxModel) View() tea.View {
	s := "Choose an option:\n\n"
	var sb strings.Builder
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		fmt.Fprintf(&sb, "%s %s\n", cursor, choice)
	}
	s += sb.String()
	s += "\nPress ↑/↓ to move, Enter to select, Q to quit.\n"
	return tea.NewView(s)
}
