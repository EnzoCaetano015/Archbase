package diagnostics

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrBoundary = errors.New("unsafe path or link")
	ErrBudget   = errors.New("diagnostic read budget exhausted")
	ErrChanged  = errors.New("filesystem changed during inspection")
	ErrExcluded = errors.New("credential or environment path excluded")
)

type Limits struct {
	Files      int
	FileBytes  int64
	TotalBytes int64
}

var DefaultLimits = Limits{Files: 10000, FileBytes: 1 << 20, TotalBytes: 50 << 20}

// Reader deliberately provides no write, repair, registry or network operation.
type Reader interface {
	ReadFile(string) ([]byte, error)
	Lstat(string) (fs.FileInfo, error)
	ReadDir(string) ([]fs.DirEntry, error)
	Verify() error
	Close() error
}

type rootReader struct {
	ctx         context.Context
	root        *os.Root
	absolute    string
	identity    fs.FileInfo
	limits      Limits
	files       int
	bytes       int64
	directories map[string][]fs.DirEntry
	snapshots   map[string]fs.FileInfo
	beforeOpen  func(string) // deterministic race injection in package tests only
}

func excluded(name string) bool {
	for _, component := range strings.Split(strings.ToLower(name), "/") {
		if component == ".ssh" || component == ".aws" || component == ".azure" || component == ".config" || component == ".gnupg" || component == ".npmrc" || component == ".pypirc" || component == ".netrc" || component == "credentials" || component == "secrets" || strings.HasPrefix(component, "credentials.") || strings.HasPrefix(component, "secrets.") || strings.HasPrefix(component, ".env") || strings.HasSuffix(component, ".pem") || strings.HasSuffix(component, ".key") || strings.HasSuffix(component, ".pfx") || strings.HasPrefix(component, "id_rsa") || strings.HasPrefix(component, "id_ed25519") {
			return true
		}
	}
	return false
}

func safeName(name string) bool {
	return utf8.ValidString(name) && fs.ValidPath(name) && path.Clean(name) == name && !strings.ContainsAny(name, "\\:\x00") && strings.IndexFunc(name, unicode.IsControl) < 0
}

// OpenRoot walks from an anchored volume root. Each component must be a stable
// directory, not a symlink/junction. os.Root confines subsequent opens even if
// a checked path is swapped concurrently; identities are checked before reads.
func OpenRoot(ctx context.Context, name string, limits Limits) (Reader, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return nil, ErrBoundary
	}
	volume := filepath.VolumeName(absolute) + string(filepath.Separator)
	current, err := os.OpenRoot(volume)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(volume, absolute)
	if err != nil {
		current.Close()
		return nil, ErrBoundary
	}
	if rel != "." {
		for _, component := range strings.Split(rel, string(filepath.Separator)) {
			if err := ctx.Err(); err != nil {
				current.Close()
				return nil, err
			}
			info, e := current.Lstat(component)
			if e != nil {
				current.Close()
				return nil, e
			}
			if !info.IsDir() || info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				current.Close()
				return nil, ErrBoundary
			}
			next, e := current.OpenRoot(component)
			if e != nil {
				current.Close()
				return nil, e
			}
			opened, e := next.Stat(".")
			after, e2 := current.Lstat(component)
			if e != nil || e2 != nil || !os.SameFile(info, opened) || !os.SameFile(info, after) || after.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				next.Close()
				current.Close()
				return nil, ErrChanged
			}
			current.Close()
			current = next
		}
	}
	identity, err := current.Stat(".")
	if err != nil {
		current.Close()
		return nil, err
	}
	return &rootReader{ctx: ctx, root: current, absolute: absolute, identity: identity, limits: limits, directories: map[string][]fs.DirEntry{}, snapshots: map[string]fs.FileInfo{}}, nil
}

func (r *rootReader) Close() error { return r.root.Close() }
func (r *rootReader) Lstat(name string) (fs.FileInfo, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	if !safeName(name) {
		return nil, ErrBoundary
	}
	current := "."
	var info fs.FileInfo
	for _, part := range strings.Split(name, "/") {
		current = path.Join(current, part)
		var err error
		info, err = r.root.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return nil, ErrBoundary
		}
	}
	return info, nil
}
func sameSnapshot(a, b fs.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime() == b.ModTime()
}
func (r *rootReader) ReadFile(name string) ([]byte, error) {
	if excluded(name) {
		return nil, ErrExcluded
	}
	info, err := r.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrBoundary
	}
	if previous, exists := r.snapshots[name]; exists && !sameSnapshot(previous, info) {
		return nil, ErrChanged
	}
	if r.files >= r.limits.Files || info.Size() > r.limits.FileBytes || r.bytes+info.Size() > r.limits.TotalBytes {
		return nil, ErrBudget
	}
	if r.beforeOpen != nil {
		r.beforeOpen(name)
	}
	f, err := r.root.OpenFile(name, os.O_RDONLY|nonblockFlag, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !sameSnapshot(info, actual) || !actual.Mode().IsRegular() {
		return nil, ErrChanged
	}
	if _, err = r.Lstat(name); err != nil {
		return nil, err
	}
	r.files++
	limit := min(r.limits.FileBytes, r.limits.TotalBytes-r.bytes)
	var data []byte
	buffer := make([]byte, 32<<10)
	for {
		if err := r.ctx.Err(); err != nil {
			return nil, err
		}
		n, e := f.Read(buffer[:min(int64(len(buffer)), limit-int64(len(data))+1)])
		data = append(data, buffer[:n]...)
		r.bytes += int64(n)
		if int64(len(data)) > limit {
			return nil, ErrBudget
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	latest, err := r.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !sameSnapshot(info, after) || !sameSnapshot(info, latest) {
		return nil, ErrChanged
	}
	r.snapshots[name] = after
	return data, nil
}
func (r *rootReader) ReadDir(name string) ([]fs.DirEntry, error) {
	info, err := r.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrBoundary
	}
	if entries, exists := r.directories[name]; exists {
		if !sameSnapshot(info, r.snapshots[name]) {
			return nil, ErrChanged
		}
		return append([]fs.DirEntry(nil), entries...), nil
	}
	if r.files >= r.limits.Files {
		return nil, ErrBudget
	}
	if r.beforeOpen != nil {
		r.beforeOpen(name)
	}
	f, err := r.root.OpenFile(name, os.O_RDONLY|nonblockFlag, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, ErrChanged
	}
	entries, err := f.ReadDir(r.limits.Files - r.files + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	r.files += len(entries)
	if r.files > r.limits.Files {
		return nil, ErrBudget
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	after, err := r.Lstat(name)
	if err != nil || !sameSnapshot(info, after) {
		return nil, ErrChanged
	}
	r.snapshots[name] = after
	r.directories[name] = append([]fs.DirEntry(nil), entries...)
	return entries, nil
}
func (r *rootReader) Verify() error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	// Re-open through the same no-link component walk to detect ancestor swaps.
	other, err := OpenRoot(r.ctx, r.absolute, r.limits)
	if err != nil {
		if r.ctx.Err() != nil {
			return r.ctx.Err()
		}
		return ErrChanged
	}
	defer other.Close()
	info, err := other.Lstat(".")
	if err != nil || !os.SameFile(r.identity, info) {
		if r.ctx.Err() != nil {
			return r.ctx.Err()
		}
		return ErrChanged
	}
	for name, previous := range r.snapshots {
		latest, err := r.Lstat(name)
		if err != nil || !sameSnapshot(previous, latest) {
			if r.ctx.Err() != nil {
				return r.ctx.Err()
			}
			return ErrChanged
		}
	}
	return nil
}
