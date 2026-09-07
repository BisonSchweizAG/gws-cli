package setup

import (
	"os"
	"path/filepath"
	"strconv"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textinput"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func InitialModel(cfg *types.Config) Model {
	fi := textinput.New()
	fi.Placeholder = "Filter..."
	fi.SetWidth(100)
	fi.Focus()

	m := Model{
		Inputs:      make([]Input, maxFocusable-1),
		Config:      cfg,
		Styles:      DefaultStyles(),
		Help:        "tab: next field / up: prev field / esc: quit / enter: confirm",
		Step:        stepLogin,
		FilterInput: fi,
		LogChan:     make(chan string, 10),
	}
	fis := m.FilterInput.Styles()
	fis.Cursor.Color = m.Styles.InputFocused.GetForeground()
	m.FilterInput.SetStyles(fis)

	configDir, _, userHomeDir := types.DefaultConfigPaths()

	if m.Config.FilePath == "" {
		m.Config.FilePath = configDir
	}

	context := &types.Context{GCloud: &types.GCloud{}}
	if m.Config.CurrentContextName != "" {
		if m.Config.Contexts != nil {
			c := m.Config.Contexts[m.Config.CurrentContextName]
			if c != nil {
				context = c
			}
		}
	}

	fp := filepicker.New()
	fp.SetHeight(24)
	fp.ShowHidden = true                // Show hidden files (e.g., .ssh)
	fp.KeyMap.Back.SetKeys("backspace") // Explicitly set key for going up a directory
	fp.KeyMap.Open.SetKeys("enter")     // Explicitly set key for going down into a directory
	startDir := ""
	if context.PrivateKeyFile != "" {
		startDir = filepath.Dir(context.PrivateKeyFile)
	} else if userHomeDir != "" {
		startDir = filepath.Join(userHomeDir, ".ssh")
	}

	if startDir != "" {
		if _, err := os.Stat(startDir); !os.IsNotExist(err) {
			fp.CurrentDirectory = startDir
		}
	}
	m.Fp = fp

	for i := range m.Inputs {
		t := textinput.New()
		s := t.Styles()
		s.Cursor.Color = m.Styles.InputFocused.GetForeground()
		t.CharLimit = 100
		t.SetWidth(120)

		switch focusable(i) {
		case ContextName:
			m.Inputs[i].Label = "Context Name"
			t.Focus()
			s.Focused.Prompt = m.Styles.InputFocused
			s.Focused.Text = m.Styles.InputFocused
			t.SetValue(m.Config.CurrentContextName)
		case Port:
			m.Inputs[i].Label = "Local SSH Port"
			t.CharLimit = 5
			if context.Port > 0 {
				t.SetValue(strconv.Itoa(context.Port))
			} else {
				t.SetValue("22022")
			}
		case User:
			m.Inputs[i].Label = "Cloud Workstation User"
			if context.User != "" {
				t.SetValue(context.User)
			} else {
				t.SetValue("user")
			}
		case PrivateKeyFile:
			m.Inputs[i].Label = "Private Key File"
			t.CharLimit = 128
			if context.PrivateKeyFile != "" {
				t.SetValue(context.PrivateKeyFile)
			} else if userHomeDir != "" {
				t.SetValue(filepath.Join(userHomeDir, ".ssh", "id_ed25519"))
			}
		case KnownHostsFile:
			m.Inputs[i].Label = "Known Hosts File (optional)"
			t.CharLimit = 128
			if context.KnownHostsFile != "" {
				t.SetValue(context.KnownHostsFile)
			} else if userHomeDir != "" {
				t.SetValue(filepath.Join(userHomeDir, ".ssh", "known_hosts"))
			}
		case GcloudProject:
			m.Inputs[i].Label = "gcloud: Project ID"
			t.SetValue(context.GCloud.Project)
		case GcloudAccount:
			m.Inputs[i].Label = "gcloud: Account (email, optional)"
			t.SetValue(context.GCloud.Account)
		case GcloudRegion:
			m.Inputs[i].Label = "gcloud: Region"
			t.SetValue(context.GCloud.Region)
		case GcloudCluster:
			m.Inputs[i].Label = "gcloud: Cluster"
			t.SetValue(context.GCloud.Cluster)
		case GcloudConfig:
			m.Inputs[i].Label = "gcloud: Config"
			t.SetValue(context.GCloud.Config)
		case GcloudName:
			m.Inputs[i].Label = "gcloud: Workstation ID/Name"
			t.SetValue(context.GCloud.Name)
		default:
			// This should not be reached as maxFocusable defines the number of inputs.
		}
		t.SetStyles(s)
		t.Placeholder = m.Inputs[i].Label
		m.Inputs[i].Style = m.Styles.Label
		m.Inputs[i].Model = t
	}

	return m
}
