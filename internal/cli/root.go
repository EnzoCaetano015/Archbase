package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const longDescription = `Archbase standardizes how AI agents structure code.

Available in this foundation release:
  arc help       Show CLI help
  arc version    Show the CLI version

Planned commands:
  arc add        Install a pattern in a local scope
  arc create     Create a customizable local pattern
  arc resolve    Resolve the active pattern for a path
  arc inspect    Inspect a local or registry pattern
  arc rules      List, inspect, and export architecture rules
  arc mcp serve  Start the MCP server`

// NewRootCommand builds the CLI command tree without coupling it to os.Exit.
func NewRootCommand(cliVersion string) *cobra.Command {
	root := &cobra.Command{
		Use:           "arc",
		Short:         "Structural patterns for AI-generated code",
		Long:          longDescription,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show the CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "arc %s\n", cliVersion)
		},
	})
	return root
}

// Execute runs the CLI and returns a process exit code.
func Execute(args []string, stdout, stderr io.Writer, cliVersion string) int {
	cmd := NewRootCommand(cliVersion)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
