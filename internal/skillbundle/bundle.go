package skillbundle

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
)

// The conservative delivery profile fits both Agensi submission channels.
const MaxFiles = 100
const MaxBytes = 5_000_000
const MaxFileBytes = 1_000_000

type Options struct {
	Root, Manifest, Lock, CatalogRoot string
}

type Bundle struct {
	Manifest Manifest
	Files    map[string][]byte
}

type frontmatter struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	License     string            `yaml:"license"`
	Metadata    map[string]string `yaml:"metadata"`
}

var localLink = regexp.MustCompile(`\[[^\]]*\]\(([^\s)]+)(?:\s+"[^"]*")?\)`)
var referenceLink = regexp.MustCompile(`(?m)^\s{0,3}\[[^\]]+\]:\s*<?([^\s>]+)>?`)
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{40,}\b`),
	regexp.MustCompile(`(?i)\b(?:api[_-]?key|access[_-]?token|password|client[_-]?secret)\s*[:=]\s*["']?[a-z0-9_+/=-]{20,}`),
}

func validateText(name string, data []byte) error {
	if err := safePath(name); err != nil {
		return err
	}
	for _, part := range strings.Split(strings.ToLower(name), "/") {
		if strings.HasPrefix(part, ".") || part == "node_modules" || part == "site-packages" || part == "__pycache__" || part == "evals" || part == "testdata" || part == "releases" || part == "dist" || part == "venv" || strings.HasSuffix(part, "_test.go") || part == "go.mod" || part == "go.sum" || part == "bundle.json" {
			return fmt.Errorf("%s: development or hidden file is forbidden", name)
		}
	}
	ext := strings.ToLower(path.Ext(name))
	allowed := map[string]bool{".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true, ".py": true, ".sh": true, ".ps1": true, ".ts": true, ".tsx": true, ".js": true}
	if !allowed[ext] && name != "LICENSE" && name != "NOTICE" {
		return fmt.Errorf("%s: file type is not allowed", name)
	}
	if len(data) == 0 || len(data) > MaxFileBytes || !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
		return fmt.Errorf("%s: expected nonempty UTF-8 text <= %d bytes", name, MaxFileBytes)
	}
	if strings.HasPrefix(name, "scripts/") && len(data) > 20_000 {
		return fmt.Errorf("%s: script exceeds reviewable 20 KB profile", name)
	}
	for _, pattern := range secretPatterns {
		if pattern.Match(data) {
			return fmt.Errorf("%s: possible secret detected", name)
		}
	}
	// Normalize checkout line endings before validation, hashing and packaging.
	return nil
}

func validateFrontmatter(data []byte, manifest Manifest) error {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return fmt.Errorf("SKILL.md: YAML frontmatter is required")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return fmt.Errorf("SKILL.md: frontmatter terminator is missing")
	}
	var front frontmatter
	if err := yaml.UnmarshalWithOptions([]byte(text[4:4+end]), &front, yaml.Strict()); err != nil {
		return fmt.Errorf("SKILL.md: invalid frontmatter: %w", err)
	}
	if front.Name != manifest.Name || !skillName.MatchString(front.Name) || len(front.Name) > 64 || strings.TrimSpace(front.Description) == "" || len(front.Description) > 1024 || strings.ContainsAny(front.Description, "<>") {
		return fmt.Errorf("SKILL.md: invalid name or description")
	}
	if front.License != manifest.License || front.Metadata["version"] != manifest.Version {
		return fmt.Errorf("SKILL.md: license/version must match manifest")
	}
	body := strings.TrimSpace(text[4+end+5:])
	if len(body) < 80 || strings.Contains(body, "[TODO]") || strings.Contains(body, "{{") {
		return fmt.Errorf("SKILL.md: unfinished or empty instructions")
	}
	return nil
}

func validateLinks(files map[string][]byte) error {
	for name, data := range files {
		if path.Ext(name) != ".md" {
			continue
		}
		links := localLink.FindAllSubmatch(data, -1)
		links = append(links, referenceLink.FindAllSubmatch(data, -1)...)
		for _, match := range links {
			value := strings.Trim(string(match[1]), "<>")
			if strings.HasPrefix(value, "#") {
				continue
			}
			u, err := url.Parse(value)
			if err != nil {
				return fmt.Errorf("%s: malformed link %s", name, value)
			}
			if u.IsAbs() {
				if u.Scheme != "https" && u.Scheme != "mailto" {
					return fmt.Errorf("%s: unsupported link %s", name, value)
				}
				continue
			}
			if u.Host != "" || strings.HasPrefix(u.Path, "/") || strings.Contains(u.Path, "\\") || u.RawQuery != "" {
				return fmt.Errorf("%s: unsafe local link %s", name, value)
			}
			target := path.Join(path.Dir(name), u.Path)
			if err = safePath(target); err != nil {
				return fmt.Errorf("%s: escaping link %s", name, value)
			}
			if _, ok := files[target]; !ok {
				return fmt.Errorf("%s: missing local reference %s", name, value)
			}
		}
	}
	return nil
}

func Prepare(ctx context.Context, options Options) (*Bundle, error) {
	var manifest Manifest
	if err := readContract(options.Manifest, "skill-manifest.schema.json", &manifest); err != nil {
		return nil, err
	}
	if filepath.Base(filepath.Clean(options.Root)) != manifest.Name {
		return nil, fmt.Errorf("skill directory must be named %s", manifest.Name)
	}
	var lock CatalogLock
	if err := readContract(options.Lock, "skill-catalog-lock.schema.json", &lock); err != nil {
		return nil, err
	}
	files, err := generatedCatalog(ctx, options.CatalogRoot, lock, manifest.Rules)
	if err != nil {
		return nil, err
	}
	lockData, _ := json.MarshalIndent(lock, "", "  ")
	files["catalog.lock.json"] = append(lockData, '\n')
	generated := map[string]bool{}
	for name := range files {
		generated[name] = true
	}
	allowed := map[string]bool{}
	caseNames := map[string]bool{}
	for _, name := range manifest.Files {
		if err = safePath(name); err != nil {
			return nil, err
		}
		if caseNames[strings.ToLower(name)] {
			return nil, fmt.Errorf("case-insensitive file collision: %s", name)
		}
		caseNames[strings.ToLower(name)] = true
		allowed[name] = true
		if _, generated := files[name]; generated {
			continue
		}
		file, err := confinedFile(options.Root, name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		info, err := os.Stat(file)
		if err != nil || !info.Mode().IsRegular() || info.Size() > MaxFileBytes {
			return nil, fmt.Errorf("%s: invalid input file", name)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		files[name] = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	}
	// Unexpected source files indicate a packaging mistake, even if a whitelist
	// would have hidden them. The manifest itself is the sole development exception.
	manifestAbs, _ := filepath.Abs(options.Manifest)
	err = filepath.WalkDir(options.Root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink: %s", file)
		}
		if entry.IsDir() {
			return nil
		}
		abs, _ := filepath.Abs(file)
		if abs == manifestAbs {
			return nil
		}
		rel, err := filepath.Rel(options.Root, file)
		if err != nil {
			return err
		}
		if generated[filepath.ToSlash(rel)] {
			return fmt.Errorf("generated catalog file must not be hand-maintained: %s", rel)
		}
		if !allowed[filepath.ToSlash(rel)] {
			return fmt.Errorf("unexpected skill source file: %s", rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	total := 0
	for name, data := range files {
		if !allowed[name] {
			return nil, fmt.Errorf("generated file is missing from explicit manifest: %s", name)
		}
		if err = validateText(name, data); err != nil {
			return nil, err
		}
		total += len(data)
	}
	if len(files) > MaxFiles || total > MaxBytes {
		return nil, fmt.Errorf("package exceeds delivery profile: %d files / %d bytes", len(files), total)
	}
	if _, ok := files["LICENSE"]; !ok {
		return nil, fmt.Errorf("LICENSE is required")
	}
	if err = validateFrontmatter(files["SKILL.md"], manifest); err != nil {
		return nil, err
	}
	if err = validateLinks(files); err != nil {
		return nil, err
	}
	return &Bundle{Manifest: manifest, Files: files}, nil
}

func (bundle *Bundle) Archive() ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	names := make([]string, 0, len(bundle.Files))
	for name := range bundle.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(0644)
		file, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err = file.Write(bundle.Files[name]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if output.Len() > MaxBytes {
		return nil, fmt.Errorf("ZIP exceeds %d byte delivery profile", MaxBytes)
	}
	return output.Bytes(), nil
}

// Write uses an exclusively created directory. A collision never overwrites an
// earlier delivery; failure removes only the directory created by this call.
func (bundle *Bundle) Write(destination string) (err error) {
	archive, err := bundle.Archive()
	if err != nil {
		return err
	}
	if err = os.Mkdir(destination, 0755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(destination)
		}
	}()
	name := bundle.Manifest.Name + "-" + bundle.Manifest.Version + ".zip"
	if err = writeNew(filepath.Join(destination, name), archive); err != nil {
		return err
	}
	if err = writeNew(filepath.Join(destination, "SHA256SUMS"), []byte(digest(archive)+"  "+name+"\n")); err != nil {
		return err
	}
	for file, data := range bundle.Files {
		if err = writeNew(filepath.Join(destination, "unpacked", filepath.FromSlash(file)), data); err != nil {
			return err
		}
	}
	return nil
}
