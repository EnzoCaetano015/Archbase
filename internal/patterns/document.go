package patterns

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/EnzoCaetano015/Archbase/internal/contracts"
)

func LanguageFor(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".jsx":
		return "jsx"
	case ".cs":
		return "csharp"
	case ".java":
		return "java"
	case ".py":
		return "python"
	case ".astro":
		return "astro"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".go":
		return "go"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".sql":
		return "sql"
	case ".sh":
		return "bash"
	default:
		return "text"
	}
}

func EncodeDocument(name string, code []byte) []byte {
	code = bytes.ReplaceAll(code, []byte("\r\n"), []byte("\n"))
	code = bytes.TrimRight(code, "\n")
	fence := "```"
	for bytes.Contains(code, []byte(fence)) {
		fence += "`"
	}
	return []byte(fence + LanguageFor(name) + "\n" + string(code) + "\n" + fence + "\n")
}

func DecodeDocument(name string, document []byte) ([]byte, string, error) {
	fail := func() ([]byte, string, error) {
		return nil, "", fmt.Errorf("%s: expected one closed Markdown code block with the language matching the source extension", name)
	}
	if !utf8.Valid(document) || !strings.HasSuffix(name, ".md") {
		return fail()
	}
	text := strings.TrimSpace(strings.ReplaceAll(string(document), "\r\n", "\n"))
	first := strings.IndexByte(text, '\n')
	if first < 0 {
		return fail()
	}
	header := text[:first]
	n := len(header) - len(strings.TrimLeft(header, "`"))
	if n < 3 {
		return fail()
	}
	lang := header[n:]
	aliases := map[string]string{"ts": "typescript", "js": "javascript", "cs": "csharp", "yml": "yaml", "md": "markdown", "sh": "bash", "plaintext": "text"}
	if canonical, ok := aliases[lang]; ok {
		lang = canonical
	}
	if lang != LanguageFor(strings.TrimSuffix(name, ".md")) {
		return fail()
	}
	fence := strings.Repeat("`", n)
	lines := strings.Split(text[first+1:], "\n")
	if len(lines) < 1 || lines[len(lines)-1] != fence {
		return fail()
	}
	for _, line := range lines[:len(lines)-1] {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= n && strings.Trim(trimmed, "`") == "" {
			return fail()
		}
	}
	return []byte(strings.Join(lines[:len(lines)-1], "\n") + "\n"), lang, nil
}

// DocumentBundle converts textual raw sources without changing destinations or binary data.
func DocumentBundle(bundle Bundle) (Bundle, error) {
	bundle.Manifest.Structure.Files = append([]contracts.PatternFile(nil), bundle.Manifest.Structure.Files...)
	bundle.Files = append([]File(nil), bundle.Files...)
	for i := range bundle.Files {
		f := &bundle.Files[i]
		if f.Spec.Format == "markdown-code" {
			continue
		}
		if f.Present && (!utf8.Valid(f.Content) || bytes.IndexByte(f.Content, 0) >= 0) {
			continue
		}
		old := f.Spec.Source
		f.Spec.Source += ".md"
		f.Spec.Format = "markdown-code"
		bundle.Manifest.Structure.Files[i] = f.Spec
		if f.Present {
			f.DocumentContent = EncodeDocument(old, f.Content)
			code, lang, err := DecodeDocument(f.Spec.Source, f.DocumentContent)
			if err != nil {
				return Bundle{}, err
			}
			f.Content, f.Language = code, lang
		}
	}
	data, err := contracts.EncodeManifest(bundle.Manifest)
	bundle.ManifestContent = data
	return bundle, err
}
