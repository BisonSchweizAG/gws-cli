package types

import (
	"fmt"
)

type Context struct {
	Host           string `validate:"required"       yaml:"host"`
	Port           int    `validate:"required"       yaml:"port"`
	User           string `yaml:"user"`
	Icon           string `yaml:"icon,omitempty"`
	PrivateKeyFile string `validate:"omitempty,file" yaml:"privateKeyFile"`
	KnownHostsFile string `validate:"omitempty,file" yaml:"knownHostsFile"`

	PostConnectCommand []string `yaml:"postConnectCommand,omitempty"`

	GCloud *GCloud `yaml:"gcloud"`

	Dirs  []Dir  `yaml:"dirs,omitempty"`
	Files []File `validate:"dive,required" yaml:"files,omitempty"`
}

type GCloud struct {
	Project string `yaml:"project"`
	Account string `yaml:"account,omitempty"`
	Region  string `yaml:"region"`
	Cluster string `yaml:"cluster"`
	Config  string `yaml:"config"`
	Name    string `yaml:"name"`
}

func (c Context) HostAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// FormatName returns the context name optionally prefixed with its icon emoji.
// If the icon is set, it appears before the name separated by a space.
func (c Context) FormatName(name string) string {
	if c.Icon != "" {
		return c.Icon + " " + name
	}
	return name
}

type Dir struct {
	Path        string `yaml:"path"`
	Permissions string `yaml:"permissions,omitempty"`
}

type File struct {
	SourcePath  string `validate:"required"                yaml:"sourcePath"`
	Path        string `validate:"required"                yaml:"path"`
	Permissions string `yaml:"permissions"`
	Direction   string `validate:"omitempty,oneof=up down" yaml:"direction"`
}
