package release

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const versionVariable = "github.com/EnzoCaetano015/Archbase/internal/version.Value"

var stableTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// Target identifies one supported release build.
type Target struct {
	OS   string
	Arch string
}

// Targets is the deterministic public release matrix.
var Targets = []Target{
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
}

// Options controls a complete release build.
type Options struct {
	Tag        string
	ModuleRoot string
	OutputDir  string
	GoBinary   string
	Timestamp  time.Time
}

// VersionFromTag validates a stable release tag and removes its v prefix.
func VersionFromTag(tag string) (string, error) {
	if !stableTag.MatchString(tag) {
		return "", fmt.Errorf("release tag %q must match vMAJOR.MINOR.PATCH", tag)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

// AssetNames returns every expected asset, including the checksum manifest.
func AssetNames(tag string) ([]string, error) {
	version, err := VersionFromTag(tag)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(Targets)+1)
	for _, target := range Targets {
		extension := ".tar.gz"
		if target.OS == "windows" {
			extension = ".zip"
		}
		names = append(names, fmt.Sprintf("arc_v%s_%s_%s%s", version, target.OS, target.Arch, extension))
	}
	names = append(names, fmt.Sprintf("arc_v%s_SHA256SUMS.txt", version))
	return names, nil
}

// Build compiles, packages, checksums, and atomically promotes a release directory.
func Build(ctx context.Context, options Options) ([]string, error) {
	version, err := VersionFromTag(options.Tag)
	if err != nil {
		return nil, err
	}
	if options.ModuleRoot == "" {
		options.ModuleRoot = "."
	}
	if options.OutputDir == "" {
		return nil, errors.New("release output directory is required")
	}
	if options.GoBinary == "" {
		options.GoBinary = "go"
	}
	if options.Timestamp.IsZero() {
		return nil, errors.New("release timestamp is required")
	}

	output, err := filepath.Abs(options.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve release output %q: %w", options.OutputDir, err)
	}
	if _, err := os.Lstat(output); err == nil {
		return nil, fmt.Errorf("release output %q already exists", output)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect release output %q: %w", output, err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return nil, fmt.Errorf("create release output parent %q: %w", filepath.Dir(output), err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(output), ".arc-release-*")
	if err != nil {
		return nil, fmt.Errorf("create release staging directory: %w", err)
	}
	defer os.RemoveAll(staging)
	buildDirectory := filepath.Join(staging, ".build")
	if err := os.MkdirAll(buildDirectory, 0o755); err != nil {
		return nil, fmt.Errorf("create release build directory: %w", err)
	}

	archiveNames := make([]string, 0, len(Targets))
	for _, target := range Targets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		binaryName := "arc"
		if target.OS == "windows" {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(buildDirectory, target.OS+"_"+target.Arch, binaryName)
		if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
			return nil, fmt.Errorf("create build directory for %s/%s: %w", target.OS, target.Arch, err)
		}
		ldflags := fmt.Sprintf("-s -w -buildid= -X %s=%s", versionVariable, version)
		command := exec.CommandContext(ctx, options.GoBinary, "build", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", binaryPath, "./cmd/arc")
		command.Dir = options.ModuleRoot
		command.Env = releaseEnvironment(os.Environ(), target)
		if output, runErr := command.CombinedOutput(); runErr != nil {
			return nil, fmt.Errorf("build arc for %s/%s: %w: %s", target.OS, target.Arch, runErr, strings.TrimSpace(string(output)))
		}
		content, err := os.ReadFile(binaryPath)
		if err != nil {
			return nil, fmt.Errorf("read built binary %q: %w", binaryPath, err)
		}
		extension := ".tar.gz"
		if target.OS == "windows" {
			extension = ".zip"
		}
		archiveName := fmt.Sprintf("arc_v%s_%s_%s%s", version, target.OS, target.Arch, extension)
		if err := writeArchive(filepath.Join(staging, archiveName), target, binaryName, content, options.Timestamp); err != nil {
			return nil, err
		}
		archiveNames = append(archiveNames, archiveName)
	}
	if err := os.RemoveAll(buildDirectory); err != nil {
		return nil, fmt.Errorf("remove release build directory %q: %w", buildDirectory, err)
	}
	checksumName := fmt.Sprintf("arc_v%s_SHA256SUMS.txt", version)
	if err := WriteChecksums(staging, archiveNames, checksumName); err != nil {
		return nil, err
	}
	if err := os.Rename(staging, output); err != nil {
		return nil, fmt.Errorf("promote release output to %q: %w", output, err)
	}
	return append(archiveNames, checksumName), nil
}

func releaseEnvironment(environment []string, target Target) []string {
	blocked := map[string]bool{"GOOS": true, "GOARCH": true, "CGO_ENABLED": true}
	result := make([]string, 0, len(environment)+3)
	for _, item := range environment {
		key, _, _ := strings.Cut(item, "=")
		if !blocked[strings.ToUpper(key)] {
			result = append(result, item)
		}
	}
	return append(result, "GOOS="+target.OS, "GOARCH="+target.Arch, "CGO_ENABLED=0")
}

func writeArchive(destination string, target Target, binaryName string, content []byte, timestamp time.Time) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create release archive %q: %w", destination, err)
	}
	var writeErr error
	if target.OS == "windows" {
		writeErr = writeZIP(file, binaryName, content, timestamp)
	} else {
		writeErr = writeTarGzip(file, binaryName, content, timestamp)
	}
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write release archive %q: %w", destination, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close release archive %q: %w", destination, closeErr)
	}
	return nil
}

func writeTarGzip(output io.Writer, name string, content []byte, timestamp time.Time) error {
	gzipWriter, err := gzip.NewWriterLevel(output, gzip.BestCompression)
	if err != nil {
		return err
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{
		Name: name, Mode: 0o755, Size: int64(len(content)), ModTime: timestamp.UTC().Truncate(time.Second),
		Uid: 0, Gid: 0, Uname: "", Gname: "", Format: tar.FormatUSTAR,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write(content); err != nil {
		return err
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	return gzipWriter.Close()
}

func writeZIP(output io.Writer, name string, content []byte, timestamp time.Time) error {
	zipWriter := zip.NewWriter(output)
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o755)
	header.SetModTime(timestamp.UTC().Truncate(2 * time.Second))
	entry, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := entry.Write(content); err != nil {
		return err
	}
	return zipWriter.Close()
}

// WriteChecksums writes a sorted SHA-256 manifest for the named files.
func WriteChecksums(directory string, names []string, outputName string) error {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	output, err := os.OpenFile(filepath.Join(directory, outputName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create checksum manifest %q: %w", outputName, err)
	}
	for _, name := range sorted {
		if filepath.Base(name) != name {
			output.Close()
			return fmt.Errorf("checksum input %q must be a file name", name)
		}
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			output.Close()
			return fmt.Errorf("read checksum input %q: %w", name, err)
		}
		digest := sha256.Sum256(content)
		if _, err := fmt.Fprintf(output, "%x  %s\n", digest, name); err != nil {
			output.Close()
			return fmt.Errorf("write checksum for %q: %w", name, err)
		}
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close checksum manifest %q: %w", outputName, err)
	}
	return nil
}

// VerifyChecksums verifies every entry in a generated SHA-256 manifest.
func VerifyChecksums(directory, checksumName string) error {
	file, err := os.Open(filepath.Join(directory, checksumName))
	if err != nil {
		return fmt.Errorf("open checksum manifest %q: %w", checksumName, err)
	}
	defer file.Close()
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 || len(fields[0]) != sha256.Size*2 {
			return fmt.Errorf("invalid checksum line %q", scanner.Text())
		}
		name := fields[1]
		if filepath.Base(name) != name || seen[name] {
			return fmt.Errorf("invalid or duplicate checksum path %q", name)
		}
		seen[name] = true
		expected, err := hex.DecodeString(fields[0])
		if err != nil {
			return fmt.Errorf("invalid checksum for %q: %w", name, err)
		}
		content, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return fmt.Errorf("read checksummed file %q: %w", name, err)
		}
		actual := sha256.Sum256(content)
		if !equalBytes(expected, actual[:]) {
			return fmt.Errorf("checksum mismatch for %q", name)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read checksum manifest %q: %w", checksumName, err)
	}
	if len(seen) == 0 {
		return errors.New("checksum manifest is empty")
	}
	return nil
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}
