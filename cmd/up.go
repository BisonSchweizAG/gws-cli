package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/ssh"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

// upCmd represents the up command.
var upCmd = &cobra.Command{
	Use:   types.DirectionUp,
	Short: "Upload files and dirs",
	RunE: func(_ *cobra.Command, args []string) error {
		cfg, err := readConfig(args...)
		if err != nil {
			return err
		}
		log.Logf("Running context %s", cfg.CurrentContextName)

		sshCtx := cfg.CurrentContext()
		cl, err := ssh.NewClient(sshCtx.HostAddr(), sshCtx.User, sshCtx.PrivateKeyFile, cfg.SSHTimeout())
		if err != nil {
			return err
		}

		defer cl.Close()

		if len(sshCtx.Dirs) > 0 {
			log.Log("Creating directories")
			for _, dir := range sshCtx.Dirs {
				if dir.Permissions != "" {
					log.Logf("Creating directory %q with permissions %s", dir.Path, dir.Permissions)
					_, err = cl.Execute(fmt.Sprintf("mkdir -p %s; chmod %s /home/user/.ssh", dir.Path, dir.Permissions))
				} else {
					log.Logf("Creating directory %q", dir.Path)
					_, err = cl.Execute("mkdir -p " + dir.Path)
				}
				if err != nil {
					return err
				}
			}
		}

		if len(sshCtx.Files) > 0 {
			log.Log("Uploading files")
			for _, file := range sshCtx.Files {
				if file.Direction != types.DirectionUp {
					continue
				}

				if file.Permissions == "0400" {
					log.Logf(
						"Add writable file permission for upload  %q with permissions %s",
						file.Path,
						file.Permissions,
					)
					_, err := cl.Execute(fmt.Sprintf("if [ -f %s ]; then chmod u+w %s; fi", file.Path, file.Path))
					if err != nil {
						return err
					}
				}
				log.Logf(
					"Uploading file from %q to %q with permissions %s [%s]",
					file.SourcePath,
					file.Path,
					file.Permissions,
					file.Direction,
				)
				err = cl.UploadFile(file.SourcePath, file.Path, file.Permissions)
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
