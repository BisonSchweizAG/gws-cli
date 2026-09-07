package setup

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

func Run(cfg *types.Config, context string) error {
	cfg.CurrentContextName = context

	p := tea.NewProgram(InitialModel(cfg))
	m, err := p.Run()
	if err != nil {
		return err
	}

	if model, ok := m.(Model); ok {
		if model.Aborted {
			log.Log("Setup aborted.")
			return nil
		}
		return SaveConfig(model)
	}
	return errors.New("could not assert model type")
}
