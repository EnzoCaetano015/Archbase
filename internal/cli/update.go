package cli

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/EnzoCaetano015/Archbase/internal/update"
	"github.com/spf13/cobra"
)

func newUpdateCommand(deps Dependencies, current string) *cobra.Command {
	var options update.Options
	command := &cobra.Command{Use: "update", Short: "Update the running Archbase installation from a verified stable release", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()
		result, err := deps.UpdateRunner(ctx, current, options)
		for _, warning := range result.Warnings {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning: "+safeText(warning))
		}
		if err != nil {
			return err
		}
		switch {
		case result.Updated:
			fmt.Fprintf(cmd.OutOrStdout(), "Updated Archbase: %s -> %s\nExecutable: %s\nRun arc version to verify. Restart running arc/MCP processes to use the new version.\n", safeText(result.Current), safeText(result.Latest), safeText(result.Executable))
		case result.Available:
			hint := "arc update"
			if options.Version != "" {
				hint += " --version " + options.Version
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Update available: %s -> %s\nRun %s to install.\n", safeText(result.Current), safeText(result.Latest), safeText(hint))
		default:
			fmt.Fprintf(cmd.OutOrStdout(), "Archbase %s is up to date (latest stable: %s).\n", safeText(result.Current), safeText(result.Latest))
		}
		return nil
	}}
	command.Flags().BoolVar(&options.Check, "check", false, "check for a release without downloading or replacing the executable")
	command.Flags().StringVar(&options.Version, "version", "", "select a stable vMAJOR.MINOR.PATCH release (no downgrades)")
	command.Flags().BoolVar(&options.Force, "force", false, "explicitly replace a development build or reinstall the same stable version")
	return command
}
