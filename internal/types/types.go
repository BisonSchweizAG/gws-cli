package types

import (
	"fmt"
)

type Context struct {
	Host           string `validate:"required"       yaml:"host"`
	Port           int    `validate:"required"       yaml:"port"`
	User           string `yaml:"user"`
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
