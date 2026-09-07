package setup

import (
	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

const (
	ContextName focusable = iota
	Port
	User
	PrivateKeyFile
	KnownHostsFile
	GcloudProject
	GcloudAccount
	GcloudRegion
	GcloudCluster
	GcloudConfig
	GcloudName
	Submit
	maxFocusable
)

type focusable int

type step int

const (
	stepLogin step = iota
	stepProject
	stepConfig
	stepForm
)

type Model struct {
	Step             step
	Inputs           []Input
	Focused          focusable
	Aborted          bool
	StatusMessage    string
	Config           *types.Config
	Styles           *Styles
	Help             string
	Fp               filepicker.Model
	FilePickerActive bool
	FilePickerField  focusable
	Width            int
	Height           int

	LoginDone     bool
	ListCursor    int
	FilterInput   textinput.Model
	Projects      []gcloud.Project
	Workstations  []gcloud.Workstation
	FilteredItems []listItem

	Logs    []string
	LogChan chan string
}

type listItem struct {
	title string
	value any
}

type (
	logMsg string
)

type Input struct {
	textinput.Model
	Label string
	Style lipgloss.Style
}
