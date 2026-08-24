package registry

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// GitSourceConfig describes a prepared Git checkout. Network and cache
// operations intentionally belong to the later registry resolution milestone.
type GitSourceConfig struct {
	URL          string
	Ref          string
	Subdirectory string
	CheckoutPath string
}

func (config GitSourceConfig) Validate() error {
	if config.URL == "" {
		return fmt.Errorf("git registry URL is required")
	}
	parsed, err := url.Parse(config.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("git registry URL must be an absolute URL: %q", config.URL)
	}
	switch parsed.Scheme {
	case "https", "ssh", "git":
	default:
		return fmt.Errorf("unsupported git registry URL scheme %q", parsed.Scheme)
	}
	if config.Ref == "" {
		return fmt.Errorf("git registry ref is required")
	}
	if config.CheckoutPath == "" || !filepath.IsAbs(config.CheckoutPath) {
		return fmt.Errorf("git registry checkout path must be absolute")
	}
	if config.Subdirectory != "" {
		normalized := strings.ReplaceAll(config.Subdirectory, "\\", "/")
		if _, err := safeRegistryPath(normalized); err != nil {
			return fmt.Errorf("invalid git registry subdirectory: %w", err)
		}
	}
	return nil
}

func (config GitSourceConfig) DirectoryRoot() (string, error) {
	if err := config.Validate(); err != nil {
		return "", err
	}
	if config.Subdirectory == "" {
		return filepath.Clean(config.CheckoutPath), nil
	}
	return filepath.Join(config.CheckoutPath, filepath.FromSlash(config.Subdirectory)), nil
}
