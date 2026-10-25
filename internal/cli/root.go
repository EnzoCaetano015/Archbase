package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	archmcp "github.com/EnzoCaetano015/Archbase/internal/mcp"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/update"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
	"github.com/spf13/cobra"
)

const longDescription = `Archbase standardizes how AI agents structure code.

Available commands:
  arc add        Install a pattern in the central project workspace
  arc create     Create a customizable local pattern
  arc resolve    Resolve the active pattern for a path
  arc inspect    Inspect a local or registry pattern
  arc rules      List, inspect, and export architecture rules
  arc remove     Remove one or all managed rule exports
  arc mcp serve  Start the project-scoped MCP server over stdio
  arc version    Show the CLI version
  arc update     Update Archbase to the latest stable release

Workspace: add/create use the current directory as the project root.
Use --project-root to select another root. Scopes live in .archbase/scopes.yaml;
pattern examples use typed Markdown (.ts.md, .tsx.md, .cs.md).
Rules live in .archbase/rules with synchronized native indices for
agents, claude, cursor, and copilot. External instructions are preserved.`

type RegistryOptions struct {
	ProjectRoot  string
	URL          string
	Ref          string
	Subdirectory string
	CacheRoot    string
	TTL          time.Duration
}

type Dependencies struct {
	FileSystem          workspace.FileSystem
	RuleFileSystem      archrules.ExportFileSystem
	MCPFileSystem       archmcp.FileSystem
	ResolverFactory     func(context.Context, RegistryOptions) (*registry.Resolver, error)
	RuleResolverFactory func(context.Context, RegistryOptions, *registry.Resolver) (*archrules.Resolver, error)
	MCPRunner           func(context.Context, *archmcp.Service, string) error
	UpdateRunner        func(context.Context, string, update.Options) (update.Result, error)
}

func defaultDependencies() Dependencies {
	return Dependencies{
		FileSystem: workspace.OSFileSystem{}, RuleFileSystem: archrules.OSExportFileSystem{}, MCPFileSystem: archmcp.OSFileSystem{},
		ResolverFactory: defaultResolver, RuleResolverFactory: defaultRuleResolver, MCPRunner: archmcp.RunStdio,
		UpdateRunner: update.New().Run,
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
		if dependencies.MCPFileSystem == nil {
			dependencies.MCPFileSystem = archmcp.OSFileSystem{}
		}
		if dependencies.MCPRunner == nil {
			dependencies.MCPRunner = archmcp.RunStdio
		}
		if dependencies.UpdateRunner == nil {
			dependencies.UpdateRunner = update.New().Run
		}
	}
	var plain bool
	options := RegistryOptions{Ref: "main", CacheRoot: defaultCacheRoot(), TTL: registry.DefaultGitCacheTTL}
	root := &cobra.Command{
		Use: "arc", Short: "Structural patterns for AI-generated code", Long: longDescription,
		SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !plainCommand(cmd) && terminalStream(cmd.InOrStdin()) && terminalStream(cmd.OutOrStdout()) {
				return runInteractive(cmd, dependencies, options, cliVersion)
			}
			if plainCommand(cmd) {
				return renderWelcome(cmd.OutOrStdout(), cliVersion, false)
			}
			return welcome(cmd.OutOrStdout(), cliVersion)
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	flags := root.PersistentFlags()
	flags.StringVar(&options.ProjectRoot, "project-root", "", "workspace root (add/create default to the current directory)")
	flags.BoolVar(&plain, "plain", false, "disable the interactive UI and rich terminal output")
	flags.StringVar(&options.URL, "registry-url", "", "public Git registry URL (https, git, or file)")
	flags.StringVar(&options.Ref, "registry-ref", "main", "Git registry branch or tag")
	flags.StringVar(&options.Subdirectory, "registry-subdir", "", "registry subdirectory inside the checkout")
	flags.StringVar(&options.CacheRoot, "registry-cache-dir", options.CacheRoot, "absolute registry cache directory")
	flags.DurationVar(&options.TTL, "registry-ttl", registry.DefaultGitCacheTTL, "registry cache time to live")
	root.AddCommand(newAddCommand(dependencies, &options), newCreateCommand(dependencies, &options), newResolveCommand(dependencies, &options), newInspectCommand(dependencies, &options), newRulesCommand(dependencies, &options), newRemoveCommand(dependencies), newMCPCommand(dependencies, &options, cliVersion))
	root.AddCommand(newUpdateCommand(dependencies, cliVersion))
	root.AddCommand(&cobra.Command{Use: "version", Short: "Show the CLI version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "arc %s\n", cliVersion)
	}})
	installHelp(root, cliVersion)
	return root
}

func newMCPCommand(dependencies Dependencies, options *RegistryOptions, cliVersion string) *cobra.Command {
	command := &cobra.Command{Use: "mcp", Short: "Run Model Context Protocol integrations", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}}
	serve := &cobra.Command{Use: "serve", Short: "Start the project-scoped MCP server over stdio", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		projectRoot := options.ProjectRoot
		if projectRoot == "" {
			projectRoot = "."
		}
		workspaceService, patterns, err := service(cmd.Context(), dependencies, *options)
		if err != nil {
			return err
		}
		rules, err := dependencies.RuleResolverFactory(cmd.Context(), *options, patterns)
		if err != nil {
			return err
		}
		mcpService, err := archmcp.NewService(projectRoot, patterns, rules, workspaceService, dependencies.MCPFileSystem)
		if err != nil {
			return err
		}
		return dependencies.MCPRunner(cmd.Context(), mcpService, cliVersion)
	}}
	command.AddCommand(serve)
	return command
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
		return executeOperation(cmd, dependencies, *options, request{Action: listRules})
	}}
}

func newRulesInspectCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "inspect <rule-id>", Short: "Inspect an architecture rule", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return executeOperation(cmd, dependencies, *options, request{Action: inspectRule, ID: args[0]})
	}}
}

func newRulesAddCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	var format, destination string
	var overwrite, merge bool
	command := &cobra.Command{Use: "add <rule-id>", Short: "Export an architecture rule", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if format == "" {
			return errors.New("--format is required")
		}
		return executeOperation(cmd, dependencies, *options, request{Action: exportRule, ID: args[0], Path: destination, Format: format, Overwrite: overwrite, Merge: merge})
	}}
	command.Flags().StringVar(&format, "format", "", "index format: agents, claude, cursor, or copilot")
	command.Flags().StringVar(&destination, "destination", ".", "project root receiving exported files")
	command.Flags().BoolVar(&overwrite, "overwrite", false, "replace a changed Archbase-owned rule document")
	command.Flags().BoolVar(&merge, "merge", false, "deprecated compatibility flag; managed indices are always merged")
	return command
}

func newRemoveCommand(dependencies Dependencies) *cobra.Command {
	var destination string
	var all bool
	command := &cobra.Command{Use: "remove [rule-id]", Short: "Remove managed architecture rule exports", Args: func(_ *cobra.Command, args []string) error {
		if all && len(args) != 0 {
			return errors.New("a rule ID cannot be combined with --all")
		}
		if !all && len(args) != 1 {
			return errors.New("provide one rule ID or use --all")
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		id := ""
		if len(args) == 1 {
			id = args[0]
		}
		return executeOperation(cmd, dependencies, RegistryOptions{}, request{Action: removeRule, ID: id, Path: destination, All: all})
	}}
	command.Flags().StringVar(&destination, "destination", ".", "project root containing managed rule exports")
	command.Flags().BoolVar(&all, "all", false, "remove every managed rule export in the project")
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
	if err == nil && options.ProjectRoot != "" {
		workspaceService = workspaceService.WithProjectRoot(options.ProjectRoot)
	}
	return workspaceService, resolver, err
}

func newAddCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "add <pattern-id> <scope-path>", Short: "Install a pattern in a local scope", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return executeOperation(cmd, dependencies, *options, request{Action: addPattern, ID: args[0], Path: args[1]})
	}}
}

func newCreateCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	var from string
	command := &cobra.Command{Use: "create <name> <scope-path>", Short: "Create a customizable local pattern", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return executeOperation(cmd, dependencies, *options, request{Action: createPattern, ID: args[0], Path: args[1], From: from})
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
		return executeOperation(cmd, dependencies, *options, request{Action: resolvePath, Path: target})
	}}
}

func newInspectCommand(dependencies Dependencies, options *RegistryOptions) *cobra.Command {
	return &cobra.Command{Use: "inspect <path-or-pattern-id>", Short: "Inspect a local or registry pattern", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return executeOperation(cmd, dependencies, *options, request{Action: inspectPattern, ID: args[0]})
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
		if !plainCommand(cmd) && terminalStream(stderr) {
			t := newTheme(os.Getenv("NO_COLOR") == "")
			fmt.Fprintln(stderr, t.danger.Render("Error  ")+safeText(err.Error()))
		} else {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	return 0
}
