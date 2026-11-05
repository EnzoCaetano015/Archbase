package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/EnzoCaetano015/Archbase/internal/diagnostics"
	"github.com/spf13/cobra"
)

type diagnosticExit struct{ code int }

func (e diagnosticExit) Error() string { return "diagnostic returned a nonzero exit status" }

func newDoctorCommand(options *RegistryOptions, version string) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "doctor", Short: "Inspect local Archbase configuration offline without changing files", Args: cobra.ArbitraryArgs, RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()
		// Do not initialize registry resolvers: doctor is always offline even
		// when global registry flags are present.
		var result diagnostics.Result
		if len(args) != 0 {
			result = diagnostics.InvalidInvocation(version)
		} else {
			result = diagnostics.Run(ctx, options.ProjectRoot, version, diagnostics.Options{})
		}
		if jsonOutput {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				return err
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Archbase doctor — %s\n", safeText(result.State))
			for _, check := range result.Checks {
				p := ""
				if check.Path != nil {
					p = " (" + safeText(*check.Path) + ")"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s%s: %s\n", check.Status, check.ID, p, safeText(check.Message))
				if check.Reason != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Reason: %s\n", check.Reason)
				}
				if check.Recommendation != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Next: %s\n", check.Recommendation)
				}
			}
		}
		if code := result.ExitCode(); code != 0 {
			return diagnosticExit{code}
		}
		return nil
	}}
	command.Flags().BoolVar(&jsonOutput, "json", false, "emit one versioned JSON object instead of human output")
	return command
}
