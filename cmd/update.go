package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/bisonschweizag/gws-cli/internal/update"
	"github.com/bisonschweizag/gws-cli/version"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gws to the latest release from GitHub",
	Long: `Download the latest gws release from GitHub, verify its SHA-256 checksum
and replace the running binary. Use --check to only look for a newer version.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().Bool("check", false, "only check whether a newer version is available")
	updateCmd.Flags().Bool("force", false, "install the latest release even if it is not newer (or this is a dev build)")
	rootCmd.AddCommand(updateCmd) // adjust if your root command variable is named differently
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	force, _ := cmd.Flags().GetBool("force")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	out := cmd.OutOrStdout()
	u := update.New()

	latest, err := u.Latest(ctx)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	newer, err := update.IsNewer(version.Version, latest.Tag)
	switch {
	case errors.Is(err, update.ErrUnknownVersion):
		if !force {
			return fmt.Errorf("this build (%q) is not a release version; latest is %s, use --force to install it anyway",
				version.Version, latest.Tag)
		}
	case err != nil:
		return err
	case !newer && !force:
		fmt.Fprintf(out, "gws %s is up to date\n", version.Version)
		return nil
	}

	if checkOnly {
		fmt.Fprintf(out, "Update available: %s -> %s\n", version.Version, latest.Tag)
		return nil
	}

	fmt.Fprintf(out, "Updating %s -> %s ...\n", version.Version, latest.Tag)
	if err := u.Apply(ctx, latest); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	fmt.Fprintf(out, "Updated to %s\n", latest.Tag)
	return nil
}
