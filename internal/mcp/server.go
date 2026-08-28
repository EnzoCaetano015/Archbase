package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewServer(service *Service, version string) (*mcpsdk.Server, error) {
	if service == nil {
		return nil, errNilService
	}
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name: "archbase", Title: "Archbase", Version: version,
		Description: "Resolve structural patterns and architecture rules for a local project.",
	}, &mcpsdk.ServerOptions{Instructions: "Patterns define code structure; rules define architecture. Resolve the nearest project scope before creating files."})
	annotation := readOnlyAnnotations()
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "search_patterns", Description: "Search available structural patterns by ID, name, or description.", Annotations: annotation}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input SearchPatternsInput) (*mcpsdk.CallToolResult, SearchPatternsOutput, error) {
		output, err := service.SearchPatterns(ctx, input)
		return nil, output, err
	})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_pattern", Description: "Get the validated manifest for a registry pattern ID.", Annotations: annotation}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input PatternIDInput) (*mcpsdk.CallToolResult, PatternOutput, error) {
		output, err := service.GetPattern(ctx, input)
		return nil, output, err
	})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "resolve_pattern", Description: "Resolve the nearest active Archbase pattern for a project path.", Annotations: annotation}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input PathInput) (*mcpsdk.CallToolResult, ResolvedPatternOutput, error) {
		output, err := service.ResolvePattern(ctx, input)
		return nil, output, err
	})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_pattern_files", Description: "Read every declared file from a registry pattern ID or resolved local project path.", Annotations: annotation}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input PatternFilesInput) (*mcpsdk.CallToolResult, PatternFilesOutput, error) {
		output, err := service.GetPatternFiles(ctx, input)
		return nil, output, err
	})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "get_scope_rules", Description: "Find architecture rules compatible with the pattern and glob for a project path.", Annotations: annotation}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input PathInput) (*mcpsdk.CallToolResult, ScopeRulesOutput, error) {
		output, err := service.GetScopeRules(ctx, input)
		return nil, output, err
	})
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "list_project_scopes", Description: "List every valid Archbase scope below the configured project root.", Annotations: annotation}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input EmptyInput) (*mcpsdk.CallToolResult, ProjectScopesOutput, error) {
		output, err := service.ListProjectScopes(ctx, input)
		return nil, output, err
	})
	return server, nil
}

func RunStdio(ctx context.Context, service *Service, version string) error {
	server, err := NewServer(service, version)
	if err != nil {
		return err
	}
	return server.Run(ctx, &mcpsdk.StdioTransport{})
}

var errNilService = &configurationError{"MCP server service is required"}

type configurationError struct{ message string }

func (e *configurationError) Error() string { return e.message }

func readOnlyAnnotations() *mcpsdk.ToolAnnotations {
	value := false
	return &mcpsdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: &value, OpenWorldHint: &value}
}
