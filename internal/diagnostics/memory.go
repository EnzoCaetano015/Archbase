package diagnostics

import (
	"bytes"
	"io/fs"
)

// boundedFS adapts the safe reader to the existing contract/bundle validators.
// It never opens an unrestricted DirFS or provides write operations.
type boundedFS struct{ reader Reader }

func (b boundedFS) ReadFile(name string) ([]byte, error)       { return b.reader.ReadFile(name) }
func (b boundedFS) ReadDir(name string) ([]fs.DirEntry, error) { return b.reader.ReadDir(name) }
func (b boundedFS) Stat(name string) (fs.FileInfo, error)      { return b.reader.Lstat(name) }
func (b boundedFS) Open(name string) (fs.File, error) {
	info, err := b.reader.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fs.ErrInvalid
	}
	data, err := b.reader.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return &memoryFile{Reader: bytes.NewReader(data), info: info}, nil
}

type memoryFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *memoryFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memoryFile) Close() error               { return nil }
