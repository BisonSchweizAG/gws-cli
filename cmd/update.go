package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/version"
)

// updateCmd represents the update command.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gws",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return update(cmd.Context(), version.Version)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func update(ctx context.Context, v string) error {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("BisonSchweizAG/gws-cli"))
	if err != nil {
		return fmt.Errorf("error occurred while detecting version: %w", err)
	}
	if !found {
		return fmt.Errorf("latest version for %s/%s could not be found from github repository", runtime.GOOS, runtime.GOARCH)
	}

	if v != version.DevelVersion && latest.LessOrEqual(v) {
		log.Printf("Current version (%s) is the latest", v)
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return errors.New("could not locate executable path")
	}
	if err := selfupdate.UpdateTo(ctx, latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}
	log.Printf("Successfully updated to version %s", latest.Version())
	return nil
}
