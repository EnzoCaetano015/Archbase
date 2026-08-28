package mcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
	"github.com/EnzoCaetano015/Archbase/internal/registry"
	archrules "github.com/EnzoCaetano015/Archbase/internal/rules"
	"github.com/EnzoCaetano015/Archbase/internal/workspace"
	"github.com/bmatcuk/doublestar/v4"
)

type FileSystem interface {
	Lstat(string) (fs.FileInfo, error)
	ReadDir(string) ([]fs.DirEntry, error)
}

type OSFileSystem struct{}

func (OSFileSystem) Lstat(value string) (fs.FileInfo, error)     { return os.Lstat(value) }
func (OSFileSystem) ReadDir(value string) ([]fs.DirEntry, error) { return os.ReadDir(value) }

type Service struct {
	root      string
	patterns  *registry.Resolver
	rules     *archrules.Resolver
	workspace *workspace.Service
	fs        FileSystem
}

func NewService(root string, patterns *registry.Resolver, rules *archrules.Resolver, workspaceService *workspace.Service, fsys FileSystem) (*Service, error) {
	if patterns == nil || rules == nil || workspaceService == nil {
		return nil, errors.New("MCP service requires pattern, rule, and workspace resolvers")
	}
	if fsys == nil {
		return nil, errors.New("MCP service filesystem is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve MCP project root %q: %w", root, err)
	}
	info, err := fsys.Lstat(absolute)
	if err != nil {
		return nil, fmt.Errorf("inspect MCP project root %q: %w", absolute, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("MCP project root %q must be a regular directory and not a symlink", absolute)
	}
	return &Service{root: absolute, patterns: patterns, rules: rules, workspace: workspaceService, fs: fsys}, nil
}

func (s *Service) SearchPatterns(ctx context.Context, input SearchPatternsInput) (SearchPatternsOutput, error) {
	listed, err := s.patterns.List(ctx)
	if err != nil {
		return SearchPatternsOutput{}, err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	output := SearchPatternsOutput{Patterns: make([]PatternSummary, 0), Stale: listed.Stale, Warnings: warningStrings(listed.Warnings)}
	for _, entry := range listed.Entries {
		resolved, err := s.patterns.Resolve(ctx, entry.ID.String())
		if err != nil {
			return SearchPatternsOutput{}, err
		}
		manifest := resolved.Pattern.Bundle.Manifest
		if query != "" && !strings.Contains(strings.ToLower(manifest.ID), query) && !strings.Contains(strings.ToLower(manifest.Name), query) && !strings.Contains(strings.ToLower(manifest.Description), query) {
			continue
		}
		output.Patterns = append(output.Patterns, PatternSummary{ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, Type: manifest.Type, Description: manifest.Description, Source: resolved.Pattern.Source})
		output.Stale = output.Stale || resolved.Stale
		output.Warnings = append(output.Warnings, warningStrings(resolved.Warnings)...)
	}
	output.Warnings = deduplicateStrings(output.Warnings)
	return output, nil
}

func (s *Service) GetPattern(ctx context.Context, input PatternIDInput) (PatternOutput, error) {
	resolved, err := s.patterns.Resolve(ctx, input.PatternID)
	if err != nil {
		return PatternOutput{}, err
	}
	return PatternOutput{Manifest: resolved.Pattern.Bundle.Manifest, Source: resolved.Pattern.Source, Stale: resolved.Stale, Warnings: warningStrings(resolved.Warnings)}, nil
}

func (s *Service) ResolvePattern(ctx context.Context, input PathInput) (ResolvedPatternOutput, error) {
	target, err := s.projectPath(input.Path)
	if err != nil {
		return ResolvedPatternOutput{}, err
	}
	resolved, err := s.workspace.Resolve(ctx, target)
	if err != nil {
		return ResolvedPatternOutput{}, err
	}
	files := make([]FileMetadata, 0, len(resolved.Pattern.Bundle.Files))
	for _, file := range resolved.Pattern.Bundle.Files {
		files = append(files, FileMetadata{Source: file.Spec.Source, Destination: file.Spec.Destination, Required: file.Spec.Required, Present: file.Present})
	}
	root := ""
	if resolved.PatternRoot != "" {
		root, err = s.relativePath(resolved.PatternRoot)
		if err != nil {
			return ResolvedPatternOutput{}, err
		}
	}
	scope, err := s.relativePath(resolved.ScopeDirectory)
	if err != nil {
		return ResolvedPatternOutput{}, err
	}
	return ResolvedPatternOutput{ScopePath: scope, PatternRoot: root, Manifest: resolved.Pattern.Bundle.Manifest, Source: resolved.Pattern.Source, Origin: resolved.Scope.Origin, Files: files, Stale: resolved.Stale, Warnings: warningStrings(resolved.Warnings)}, nil
}

func (s *Service) GetPatternFiles(ctx context.Context, input PatternFilesInput) (PatternFilesOutput, error) {
	byID := strings.TrimSpace(input.PatternID) != ""
	byPath := strings.TrimSpace(input.Path) != ""
	if byID == byPath {
		return PatternFilesOutput{}, errors.New("exactly one of patternId or path is required")
	}
	var pattern registry.Pattern
	var stale bool
	var warnings []error
	if byID {
		resolved, err := s.patterns.Resolve(ctx, input.PatternID)
		if err != nil {
			return PatternFilesOutput{}, err
		}
		pattern, stale, warnings = resolved.Pattern, resolved.Stale, resolved.Warnings
	} else {
		target, err := s.projectPath(input.Path)
		if err != nil {
			return PatternFilesOutput{}, err
		}
		resolved, err := s.workspace.Resolve(ctx, target)
		if err != nil {
			return PatternFilesOutput{}, err
		}
		pattern, stale, warnings = resolved.Pattern, resolved.Stale, resolved.Warnings
	}
	files := make([]PatternFileContent, 0, len(pattern.Bundle.Files))
	for _, file := range pattern.Bundle.Files {
		item := PatternFileContent{Source: file.Spec.Source, Destination: file.Spec.Destination, Required: file.Spec.Required, Present: file.Present}
		if file.Present {
			if utf8.Valid(file.Content) {
				item.Encoding, item.Content = "utf-8", string(file.Content)
			} else {
				item.Encoding, item.Content = "base64", base64.StdEncoding.EncodeToString(file.Content)
			}
		}
		files = append(files, item)
	}
	return PatternFilesOutput{PatternID: pattern.Bundle.Manifest.ID, Source: pattern.Source, Files: files, Stale: stale, Warnings: warningStrings(warnings)}, nil
}

func (s *Service) GetScopeRules(ctx context.Context, input PathInput) (ScopeRulesOutput, error) {
	target, err := s.projectPath(input.Path)
	if err != nil {
		return ScopeRulesOutput{}, err
	}
	resolvedPattern, err := s.workspace.Resolve(ctx, target)
	if err != nil {
		return ScopeRulesOutput{}, err
	}
	targetRelative, err := s.relativePath(target)
	if err != nil {
		return ScopeRulesOutput{}, err
	}
	compatible := map[string]struct{}{resolvedPattern.Pattern.Bundle.Manifest.ID: {}}
	originID := ""
	if resolvedPattern.Scope.Origin != nil {
		originID = resolvedPattern.Scope.Origin.ID
		compatible[originID] = struct{}{}
	}
	listed, err := s.rules.List(ctx)
	if err != nil {
		return ScopeRulesOutput{}, err
	}
	output := ScopeRulesOutput{PatternID: resolvedPattern.Pattern.Bundle.Manifest.ID, OriginID: originID, Rules: make([]RuleMatch, 0), Stale: resolvedPattern.Stale || listed.Stale}
	output.Warnings = append(output.Warnings, warningStrings(resolvedPattern.Warnings)...)
	output.Warnings = append(output.Warnings, warningStrings(listed.Warnings)...)
	for _, entry := range listed.Entries {
		resolvedRule, err := s.rules.Resolve(ctx, entry.ID.String())
		if err != nil {
			return ScopeRulesOutput{}, err
		}
		matching := make([]structScope, 0)
		for _, scope := range resolvedRule.Rule.Definition.Scopes {
			if _, ok := compatible[scope.Pattern]; !ok {
				continue
			}
			matches, err := doublestar.Match(scope.Path, targetRelative)
			if err != nil {
				return ScopeRulesOutput{}, fmt.Errorf("match rule %s scope %q: %w", entry.ID, scope.Path, err)
			}
			if matches {
				matching = append(matching, structScope{Path: scope.Path, Pattern: scope.Pattern})
			}
		}
		if len(matching) == 0 {
			continue
		}
		scopes := makeRuleScopes(matching)
		definition := resolvedRule.Rule.Definition
		output.Rules = append(output.Rules, RuleMatch{ID: definition.ID, Name: definition.Name, Version: definition.Version, Description: definition.Description, Source: resolvedRule.Rule.Source, MatchingScopes: scopes, Restrictions: append([]string(nil), definition.Rules...)})
		output.Stale = output.Stale || resolvedRule.Stale
		output.Warnings = append(output.Warnings, warningStrings(resolvedRule.Warnings)...)
	}
	output.Warnings = deduplicateStrings(output.Warnings)
	return output, nil
}

type structScope struct{ Path, Pattern string }

func makeRuleScopes(values []structScope) []contracts.RuleScope {
	result := make([]contracts.RuleScope, 0, len(values))
	for _, value := range values {
		result = append(result, contracts.RuleScope{Path: value.Path, Pattern: value.Pattern})
	}
	return result
}

func (s *Service) ListProjectScopes(ctx context.Context, _ EmptyInput) (ProjectScopesOutput, error) {
	result := ProjectScopesOutput{Scopes: make([]ProjectScope, 0)}
	var walk func(string) error
	walk = func(directory string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, err := s.fs.ReadDir(directory)
		if err != nil {
			return fmt.Errorf("list project directory %q: %w", directory, err)
		}
		for _, entry := range entries {
			if entry.Name() == ".git" {
				continue
			}
			current := filepath.Join(directory, entry.Name())
			info, err := s.fs.Lstat(current)
			if err != nil {
				return fmt.Errorf("inspect project path %q: %w", current, err)
			}
			if info.Mode()&fs.ModeSymlink != 0 {
				if entry.Name() == ".archbase" {
					return fmt.Errorf("project scope %q must not be a symlink", current)
				}
				continue
			}
			if entry.Name() == ".archbase" {
				if !info.IsDir() {
					return fmt.Errorf("project scope %q must be a regular directory", current)
				}
				resolved, err := s.workspace.Resolve(ctx, directory)
				if err != nil {
					return err
				}
				relative, err := s.relativePath(resolved.ScopeDirectory)
				if err != nil {
					return err
				}
				result.Scopes = append(result.Scopes, ProjectScope{Path: relative, Pattern: resolved.Scope.Pattern, Origin: resolved.Scope.Origin})
				result.Stale = result.Stale || resolved.Stale
				result.Warnings = append(result.Warnings, warningStrings(resolved.Warnings)...)
				continue
			}
			if !info.IsDir() {
				continue
			}
			if err := walk(current); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(s.root); err != nil {
		return ProjectScopesOutput{}, err
	}
	sort.Slice(result.Scopes, func(i, j int) bool { return result.Scopes[i].Path < result.Scopes[j].Path })
	result.Warnings = deduplicateStrings(result.Warnings)
	return result, nil
}

func (s *Service) projectPath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("path is required")
	}
	for _, part := range strings.Split(strings.ReplaceAll(value, "\\", "/"), "/") {
		if part == ".." {
			return "", fmt.Errorf("path %q contains traversal", value)
		}
	}
	target := value
	if !filepath.IsAbs(target) {
		target = filepath.Join(s.root, target)
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve project path %q: %w", value, err)
	}
	if _, err := s.relativePath(absolute); err != nil {
		return "", err
	}
	current := s.root
	relative, _ := filepath.Rel(s.root, absolute)
	if relative == "." {
		return absolute, nil
	}
	components := strings.Split(relative, string(filepath.Separator))
	for index, component := range components {
		current = filepath.Join(current, component)
		info, err := s.fs.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("inspect project path %q: %w", current, err)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return "", fmt.Errorf("project path %q must not contain symlinks", current)
		}
		if index < len(components)-1 && !info.IsDir() {
			return "", fmt.Errorf("project path component %q is not a directory", current)
		}
	}
	return absolute, nil
}

func (s *Service) relativePath(value string) (string, error) {
	relative, err := filepath.Rel(s.root, value)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("path %q escapes MCP project root %q", value, s.root)
	}
	return filepath.ToSlash(relative), nil
}

func warningStrings(values []error) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, strings.TrimSpace(value.Error()))
		}
	}
	return result
}

func deduplicateStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
