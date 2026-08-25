package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
	"github.com/spf13/cobra"
)

const longDescription = `Archbase standardizes how AI agents structure code.

Available commands:
  arc add        Install a pattern in a local scope
  arc create     Create a customizable local pattern
  arc resolve    Resolve the active pattern for a path
  arc inspect    Inspect a local or registry pattern
  arc version    Show the CLI version

Planned commands:
  arc rules      List, inspect, and export architecture rules
  arc mcp serve  Start the MCP server`

type RegistryOptions struct {
	URL          string
	Ref          string
	Subdirectory string
	CacheRoot    string
	TTL          time.Duration
}

type Dependencies struct {
	FileSystem      workspace.FileSystem
	ResolverFactory func(context.Context, RegistryOptions) (*registry.Resolver, error)
}

func defaultDependencies() Dependencies {
	return Dependencies{FileSystem: workspace.OSFileSystem{}, ResolverFactory: defaultResolver}
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
	root.AddCommand(newAddCommand(dependencies, &options), newCreateCommand(dependencies, &options), newResolveCommand(dependencies, &options), newInspectCommand(dependencies, &options))
	root.AddCommand(&cobra.Command{Use: "version", Short: "Show the CLI version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "arc %s\n", cliVersion)
	}})
	return root
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
