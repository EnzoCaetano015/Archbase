package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
	"github.com/spf13/cobra"
)

const longDescription = `Archbase standardizes how AI agents structure code.

Available commands:
  arc add        Install a pattern in a local scope
  arc create     Create a customizable local pattern
  arc resolve    Resolve the active pattern for a path
  arc inspect    Inspect a local or registry pattern
  arc rules      List, inspect, and export architecture rules
  arc version    Show the CLI version

Planned commands:
  arc mcp serve  Start the MCP server`

type RegistryOptions struct {
	URL          string
	Ref          string
	Subdirectory string
	CacheRoot    string
	TTL          time.Duration
}

type Dependencies struct {
	FileSystem          workspace.FileSystem
	RuleFileSystem      archrules.ExportFileSystem
	ResolverFactory     func(context.Context, RegistryOptions) (*registry.Resolver, error)
	RuleResolverFactory func(context.Context, RegistryOptions, *registry.Resolver) (*archrules.Resolver, error)
}

func defaultDependencies() Dependencies {
	return Dependencies{
		FileSystem: workspace.OSFileSystem{}, RuleFileSystem: archrules.OSExportFileSystem{},
		ResolverFactory: defaultResolver, RuleResolverFactory: defaultRuleResolver,
	}
}

func defaultRuleResolver(_ context.Context, options RegistryOptions, patterns *registry.Resolver) (*archrules.Resolver, error) {
	embedded, err := archrules.NewEmbeddedSource()
	if err != nil {
		return nil, err
	}
	sources := []archrules.Source{embedded}
	if options.URL != "" {
		remote, err := archrules.NewGitSource(registry.GitSourceConfig{
			URL: options.URL, Ref: options.Ref, Subdirectory: options.Subdirectory,
			CacheRoot: options.CacheRoot, TTL: options.TTL,
		})
		if err != nil {
			return nil, err
		}
		sources = append([]archrules.Source{remote}, sources...)
	}
	return archrules.NewResolver(patterns, sources...)
}

func defaultResolver(_ context.Context, options RegistryOptions) (*registry.Resolver, error) {
	embedded, err := registry.NewEmbeddedSource()
	if err != nil {
		return nil, err
	}
	sources := []registry.Source{embedded}
	if options.URL != "" {
		remote, err := registry.NewGitSource(registry.GitSourceConfig{
			URL: options.URL, Ref: options.Ref, Subdirectory: options.Subdirectory,
			CacheRoot: options.CacheRoot, TTL: options.TTL,
		})
		if err != nil {
			return nil, err
		}
		sources = append([]registry.Source{remote}, sources...)
	}
	return registry.NewResolver(sources...)
}

// NewRootCommand builds the CLI command tree without coupling it to os.Exit.
func NewRootCommand(cliVersion string, supplied ...Dependencies) *cobra.Command {
	dependencies := defaultDependencies()
	if len(supplied) > 0 {
		dependencies = supplied[0]
		if dependencies.FileSystem == nil {
			dependencies.FileSystem = workspace.OSFileSystem{}
		}
		if dependencies.ResolverFactory == nil {
			dependencies.ResolverFactory = defaultResolver
		}
		if dependencies.RuleFileSystem == nil {
			dependencies.RuleFileSystem = archrules.OSExportFileSystem{}
		}
		if dependencies.RuleResolverFactory == nil {
			dependencies.RuleResolverFactory = defaultRuleResolver
		}
	}
	options := RegistryOptions{Ref: "main", CacheRoot: defaultCacheRoot(), TTL: registry.DefaultGitCacheTTL}
	root := &cobra.Command{
		Use: "arc", Short: "Structural patterns for AI-generated code", Long: longDescription,
		SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	root.CompletionOptions.DisableDefaultCmd = true
	flags := root.PersistentFlags()
	flags.StringVar(&options.URL, "registry-url", "", "public Git registry URL (https, git, or file)")
	flags.StringVar(&options.Ref, "registry-ref", "main", "Git registry branch or tag")
	flags.StringVar(&options.Subdirectory, "registry-subdir", "", "registry subdirectory inside the checkout")
	flags.StringVar(&options.CacheRoot, "registry-cache-dir", options.CacheRoot, "absolute registry cache directory")
	flags.DurationVar(&options.TTL, "registry-ttl", registry.DefaultGitCacheTTL, "registry cache time to live")
	root.AddCommand(newAddCommand(dependencies, &options), newCreateCommand(dependencies, &options), newResolveCommand(dependencies, &options), newInspectCommand(dependencies, &options), newRulesCommand(dependencies, &options))
	root.AddCommand(&cobra.Command{Use: "version", Short: "Show the CLI version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "arc %s\n", cliVersion)
	}})
	return root
}

func ruleService(ctx context.Context, dependencies Dependencies, options RegistryOptions) (*archrules.Resolver, error) {
	patterns, err := dependencies.ResolverFactory(ctx, options)
	if err != nil {
		return nil, err
	}
	return dependencies.RuleResolverFactory(ctx, options, patterns)
}

func newRulesCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	command := &cobra.Command{Use: "rules", Short: "Manage architecture rules", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}}
	command.AddCommand(newRulesListCommand(dependencies, options), newRulesInspectCommand(dependencies, options), newRulesAddCommand(dependencies, options))
	return command
}

func newRulesListCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List available architecture rules", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		resolver, err := ruleService(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		result, err := resolver.List(cmd.Context())
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), result.Warnings)
		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "ID\tVERSION\tSOURCE\tDESCRIPTION")
		for _, entry := range result.Entries {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", entry.ID, entry.Version, entry.Source, entry.Description)
		}
		return writer.Flush()
	}}
}

func newRulesInspectCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "inspect <rule-id>", Short: "Inspect an architecture rule", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		resolver, err := ruleService(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		resolved, err := resolver.Resolve(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), resolved.Warnings)
		printRule(cmd.OutOrStdout(), resolved.Rule)
		return nil
	}}
}

func newRulesAddCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	var rawFormat, destination string
	var overwrite, merge bool
	command := &cobra.Command{Use: "add <rule-id>", Short: "Export an architecture rule", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if rawFormat == "" {
			return errors.New("--format is required")
		}
		format, err := archrules.ParseFormat(rawFormat)
		if err != nil {
			return err
		}
		resolver, err := ruleService(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		resolved, err := resolver.Resolve(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), resolved.Warnings)
		exporter, err := archrules.NewExporter(dependencies.RuleFileSystem)
		if err != nil {
			return err
		}
		result, err := exporter.Export(resolved.Rule, format, archrules.ExportOptions{Destination: destination, Overwrite: overwrite, Merge: merge})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Exported: %s\nFormat: %s\nFiles:\n", resolved.Rule.Definition.ID, result.Format)
		for _, file := range result.Paths {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", file)
		}
		return nil
	}}
	command.Flags().StringVar(&rawFormat, "format", "", "export format: cursor, copilot, or agents")
	command.Flags().StringVar(&destination, "destination", ".", "project root receiving exported files")
	command.Flags().BoolVar(&overwrite, "overwrite", false, "replace an existing Cursor or Copilot file")
	command.Flags().BoolVar(&merge, "merge", false, "merge the managed block into an existing AGENTS.md")
	return command
}

func printRule(output io.Writer, rule archrules.Rule) {
	definition := rule.Definition
	fmt.Fprintf(output, "Rule: %s\nName: %s\nVersion: %s\nSource: %s\nDescription: %s\n", definition.ID, definition.Name, definition.Version, rule.Source, definition.Description)
	fmt.Fprintln(output, "Scopes:")
	for _, scope := range definition.Scopes {
		fmt.Fprintf(output, "  %s -> %s\n", scope.Path, scope.Pattern)
	}
	printValues(output, "Rules", definition.Rules)
}

func defaultCacheRoot() string {
	root, err := os.UserCacheDir()
	if err != nil || root == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "archbase", "registries")
}

func service(ctx context.Context, dependencies Dependencies, options RegistryOptions) (*workspace.Service, *registry.Resolver, error) {
	resolver, err := dependencies.ResolverFactory(ctx, options)
	if err != nil {
		return nil, nil, err
	}
	workspaceService, err := workspace.NewService(dependencies.FileSystem, resolver)
	return workspaceService, resolver, err
}

func newAddCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "add <pattern-id> <scope-path>", Short: "Install a pattern in a local scope", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		workspaceService, _, err := service(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		installed, err := workspaceService.Add(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), installed.Warnings)
		fmt.Fprintf(cmd.OutOrStdout(), "Installed: %s\nScope: %s\nPattern root: %s\n", installed.PatternID, installed.ScopeDirectory, installed.PatternDirectory)
		return nil
	}}
}

func newCreateCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	var from string
	command := &cobra.Command{Use: "create <name> <scope-path>", Short: "Create a customizable local pattern", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		workspaceService, _, err := service(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		installed, err := workspaceService.Create(cmd.Context(), args[0], args[1], from)
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), installed.Warnings)
		fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\nScope: %s\nPattern root: %s\n", installed.PatternID, installed.ScopeDirectory, installed.PatternDirectory)
		return nil
	}}
	command.Flags().StringVar(&from, "from", "", "derive the local pattern from a registry pattern ID")
	return command
}

func newResolveCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "resolve [path]", Short: "Resolve the active pattern for a path", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) == 1 {
			target = args[0]
		}
		workspaceService, _, err := service(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		resolved, err := workspaceService.Resolve(cmd.Context(), target)
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), resolved.Warnings)
		printResolution(cmd.OutOrStdout(), resolved)
		return nil
	}}
}

func newInspectCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "inspect <path-or-pattern-id>", Short: "Inspect a local or registry pattern", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		workspaceService, resolver, err := service(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		if _, parseErr := registry.ParsePatternID(args[0]); parseErr == nil {
			resolved, err := resolver.Resolve(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printWarnings(cmd.ErrOrStderr(), resolved.Warnings)
			printPattern(cmd.OutOrStdout(), resolved.Pattern, resolved.Pattern.Source, nil)
			return nil
		}
		resolved, err := workspaceService.Resolve(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		printWarnings(cmd.ErrOrStderr(), resolved.Warnings)
		printPattern(cmd.OutOrStdout(), resolved.Pattern, resolved.Scope.Pattern.Source, resolved.Scope.Origin)
		return nil
	}}
}

func printResolution(output io.Writer, resolved workspace.Resolution) {
	fmt.Fprintf(output, "Scope: %s\nPattern: %s\nSource: %s\n", resolved.ScopeDirectory, resolved.Pattern.Entry.ID, resolved.Scope.Pattern.Source)
	if resolved.PatternRoot != "" {
		fmt.Fprintf(output, "Root: %s\n", resolved.PatternRoot)
	}
	printOrigin(output, resolved.Scope.Origin)
	fmt.Fprintln(output, "Files:")
	for _, file := range resolved.Pattern.Bundle.Files {
		fmt.Fprintf(output, "  %s -> %s (required=%t, present=%t)\n", file.Spec.Source, file.Spec.Destination, file.Spec.Required, file.Present)
	}
}

func printPattern(output io.Writer, pattern registry.Pattern, source string, origin *contracts.Origin) {
	manifest := pattern.Bundle.Manifest
	fmt.Fprintf(output, "Pattern: %s\nName: %s\nVersion: %s\nType: %s\nSource: %s\n", manifest.ID, manifest.Name, manifest.Version, manifest.Type, source)
	printOrigin(output, origin)
	fmt.Fprintln(output, "Files:")
	for _, file := range pattern.Bundle.Files {
		fmt.Fprintf(output, "  %s -> %s (required=%t, present=%t)\n", file.Spec.Source, file.Spec.Destination, file.Spec.Required, file.Present)
	}
	printValues(output, "Allowed changes", manifest.AllowedChanges)
	printValues(output, "Preserve", manifest.Preserve)
}

func printOrigin(output io.Writer, origin *contracts.Origin) {
	if origin == nil {
		fmt.Fprintln(output, "Origin: none")
		return
	}
	fmt.Fprintf(output, "Origin: %s (%s, version %s)\n", origin.Registry, origin.ID, origin.Version)
}

func printValues(output io.Writer, label string, values []string) {
	fmt.Fprintf(output, "%s:\n", label)
	for _, value := range values {
		fmt.Fprintf(output, "  %s\n", value)
	}
}

func printWarnings(output io.Writer, warnings []error) {
	for _, warning := range warnings {
		if warning != nil {
			fmt.Fprintf(output, "warning: %s\n", strings.TrimSpace(warning.Error()))
		}
	}
}

// Execute runs the CLI and returns a process exit code.
func Execute(args []string, stdout, stderr io.Writer, cliVersion string, dependencies ...Dependencies) int {
	cmd := NewRootCommand(cliVersion, dependencies...)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
