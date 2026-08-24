package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewDirectorySource(root string) (Source, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve registry directory %q: %w", root, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("open registry directory %q: %w", absolute, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("open registry directory %q: not a directory", absolute)
	}
	return newCatalogSource("directory:"+absolute, os.DirFS(absolute))
}
