package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DefaultGitCacheTTL = 15 * time.Minute

var gitRefExpression = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

type GitSourceConfig struct {
	URL          string
	Ref          string
	Subdirectory string
	CacheRoot    string
	TTL          time.Duration
}

func (config GitSourceConfig) Validate() error {
	if config.URL == "" {
		return fmt.Errorf("git registry URL is required")
	}
	parsed, err := url.Parse(config.URL)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("git registry URL must use https, git, or file: %q", config.URL)
	}
	if parsed.User != nil {
		return fmt.Errorf("git registry URL must not contain credentials")
	}
	switch parsed.Scheme {
	case "https", "git":
		if parsed.Host == "" {
			return fmt.Errorf("git registry URL must include a host: %q", config.URL)
		}
	case "file":
		if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
			return fmt.Errorf("file registry URL must include an absolute path")
		}
	default:
		return fmt.Errorf("unsupported git registry URL scheme %q", parsed.Scheme)
	}
	if !gitRefExpression.MatchString(config.Ref) || strings.Contains(config.Ref, "..") || strings.HasSuffix(config.Ref, "/") {
		return fmt.Errorf("invalid git registry ref %q", config.Ref)
	}
	if config.CacheRoot == "" || !filepath.IsAbs(config.CacheRoot) {
		return fmt.Errorf("git registry cache root must be absolute")
	}
	if config.TTL < 0 {
		return fmt.Errorf("git registry TTL must not be negative")
	}
	if config.Subdirectory != "" {
		normalized := strings.ReplaceAll(config.Subdirectory, "\\", "/")
		if _, err := safeRegistryPath(normalized); err != nil {
			return fmt.Errorf("invalid git registry subdirectory: %w", err)
		}
	}
	return nil
}

func (config GitSourceConfig) cacheTTL() time.Duration {
	if config.TTL == 0 {
		return DefaultGitCacheTTL
	}
	return config.TTL
}

func (config GitSourceConfig) cacheKey() string {
	hash := sha256.Sum256([]byte(config.URL + "\x00" + config.Ref))
	return hex.EncodeToString(hash[:])
}

func (config GitSourceConfig) checkoutPath() string {
	return filepath.Join(config.CacheRoot, config.cacheKey(), "checkout")
}

func (config GitSourceConfig) registryPath() string {
	root := config.checkoutPath()
	if config.Subdirectory == "" {
		return root
	}
	return filepath.Join(root, filepath.FromSlash(config.Subdirectory))
}
