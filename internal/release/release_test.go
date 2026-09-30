package release

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestVersionFromTag(t *testing.T) {
	for _, test := range []struct {
		tag, version string
		valid        bool
	}{
		{tag: "v0.1.0", version: "0.1.0", valid: true},
		{tag: "v12.34.56", version: "12.34.56", valid: true},
		{tag: "0.1.0"},
		{tag: "v01.2.3"},
		{tag: "v1.2"},
		{tag: "v1.2.3-rc.1"},
	} {
		t.Run(test.tag, func(t *testing.T) {
			version, err := VersionFromTag(test.tag)
			if test.valid && (err != nil || version != test.version) {
				t.Fatalf("VersionFromTag(%q) = %q, %v", test.tag, version, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("VersionFromTag(%q) unexpectedly succeeded", test.tag)
			}
		})
	}
}

func TestAssetNames(t *testing.T) {
	names, err := AssetNames("v0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"arc_v0.3.0_darwin_amd64.tar.gz",
		"arc_v0.3.0_darwin_arm64.tar.gz",
		"arc_v0.3.0_linux_amd64.tar.gz",
		"arc_v0.3.0_linux_arm64.tar.gz",
		"arc_v0.3.0_windows_amd64.zip",
		"arc_v0.3.0_windows_arm64.zip",
		"arc_v0.3.0_SHA256SUMS.txt",
	}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("unexpected assets:\n%q\nwant:\n%q", names, expected)
	}
}

func TestWindowsPEVersion(t *testing.T) {
	version, components, err := windowsPEVersion("12.34.56")
	if err != nil {
		t.Fatal(err)
	}
	if version != "12.34.56.0" || components != [3]string{"12", "34", "56"} {
		t.Fatalf("unexpected PE version: %q %#v", version, components)
	}
	for _, invalid := range []string{"1.2", "1.2.x", "1.2.65536"} {
		if _, _, err := windowsPEVersion(invalid); err == nil {
			t.Fatalf("windowsPEVersion(%q) unexpectedly succeeded", invalid)
		}
	}
}

func TestWindowsResourceFailureCleansTemporaryPackage(t *testing.T) {
	moduleRoot := windowsBuildFixture(t)
	runner := func(context.Context, string, []string, string, ...string) ([]byte, error) {
		return []byte("fixture generator failure"), errors.New("exit status 1")
	}
	_, _, err := prepareWindowsBuildPackage(context.Background(), Options{ModuleRoot: moduleRoot, GoBinary: "go"}, Target{OS: "windows", Arch: "amd64"}, "1.2.3", runner)
	if err == nil || !strings.Contains(err.Error(), "fixture generator failure") {
		t.Fatalf("unexpected generator error: %v", err)
	}
	assertInternalDirectoryEmpty(t, moduleRoot)
}

func TestWindowsBuildFailureCleansTemporaryPackage(t *testing.T) {
	moduleRoot := windowsBuildFixture(t)
	call := 0
	runner := func(context.Context, string, []string, string, ...string) ([]byte, error) {
		call++
		if call == 2 {
			return []byte("fixture build failure"), errors.New("exit status 1")
		}
		return nil, nil
	}
	err := buildBinary(context.Background(), Options{ModuleRoot: moduleRoot, GoBinary: "go"}, Target{OS: "windows", Arch: "arm64"}, "1.2.3", filepath.Join(t.TempDir(), "arc.exe"), runner)
	if err == nil || !strings.Contains(err.Error(), "fixture build failure") {
		t.Fatalf("unexpected build error: %v", err)
	}
	assertInternalDirectoryEmpty(t, moduleRoot)
}

func windowsBuildFixture(t *testing.T) string {
	t.Helper()
	moduleRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(moduleRoot, "cmd", "arc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(moduleRoot, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "cmd", "arc", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return moduleRoot
}

func assertInternalDirectoryEmpty(t *testing.T, moduleRoot string) {
	t.Helper()
	entries, readErr := os.ReadDir(filepath.Join(moduleRoot, "internal"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary package was not removed: %v", entries)
	}
}

func TestReleaseBuildIsReproducible(t *testing.T) {
	if os.Getenv("ARCHBASE_RELEASE_INTEGRATION") != "1" {
		t.Skip("set ARCHBASE_RELEASE_INTEGRATION=1 to run the cross-platform release build")
	}
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goBinary := os.Getenv("ARCHBASE_GO_BINARY")
	if goBinary == "" {
		goBinary = "go"
	}
	timestamp := time.Date(2026, time.September, 30, 12, 34, 56, 0, time.UTC)
	outputRoot := os.Getenv("ARCHBASE_RELEASE_OUTPUT")
	if outputRoot == "" {
		outputRoot = t.TempDir()
	} else if err := os.Mkdir(outputRoot, 0o755); err != nil {
		t.Fatalf("create requested integration output %q: %v", outputRoot, err)
	}
	first := filepath.Join(outputRoot, "release-a")
	second := filepath.Join(outputRoot, "release-b")
	for _, output := range []string{first, second} {
		if _, err := Build(context.Background(), Options{
			Tag: "v9.8.7", ModuleRoot: moduleRoot, OutputDir: output,
			GoBinary: goBinary, Timestamp: timestamp,
		}); err != nil {
			t.Fatal(err)
		}
	}
	assertDirectoriesEqual(t, first, second)
}

func assertDirectoriesEqual(t *testing.T, left, right string) {
	t.Helper()
	leftEntries, err := os.ReadDir(left)
	if err != nil {
		t.Fatal(err)
	}
	rightEntries, err := os.ReadDir(right)
	if err != nil {
		t.Fatal(err)
	}
	if len(leftEntries) != len(rightEntries) {
		t.Fatalf("release directories contain different file counts: %d and %d", len(leftEntries), len(rightEntries))
	}
	differences := make([]string, 0)
	for index, leftEntry := range leftEntries {
		if leftEntry.Name() != rightEntries[index].Name() {
			t.Fatalf("release file names differ: %q and %q", leftEntry.Name(), rightEntries[index].Name())
		}
		leftContent, err := os.ReadFile(filepath.Join(left, leftEntry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		rightContent, err := os.ReadFile(filepath.Join(right, rightEntries[index].Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(leftContent, rightContent) {
			differences = append(differences, leftEntry.Name())
		}
	}
	if len(differences) > 0 {
		t.Fatalf("release artifacts are not reproducible: %s", strings.Join(differences, ", "))
	}
}

func TestArchivesAreDeterministicAndContainOnlyBinary(t *testing.T) {
	timestamp := time.Date(2026, time.August, 27, 12, 34, 56, 0, time.UTC)
	for _, target := range []Target{{OS: "linux", Arch: "amd64"}, {OS: "windows", Arch: "amd64"}} {
		t.Run(target.OS, func(t *testing.T) {
			name := "arc"
			if target.OS == "windows" {
				name += ".exe"
			}
			first := filepath.Join(t.TempDir(), "first")
			second := filepath.Join(t.TempDir(), "second")
			content := []byte("deterministic binary")
			if err := writeArchive(first, target, name, content, timestamp); err != nil {
				t.Fatal(err)
			}
			if err := writeArchive(second, target, name, content, timestamp); err != nil {
				t.Fatal(err)
			}
			left, _ := os.ReadFile(first)
			right, _ := os.ReadFile(second)
			if !bytes.Equal(left, right) {
				t.Fatal("archive bytes differ between identical runs")
			}
			if target.OS == "windows" {
				assertZIP(t, left, name, content, timestamp)
			} else {
				assertTarGzip(t, left, name, content, timestamp)
			}
		})
	}
}

func TestChecksumsVerifyAndDetectChanges(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "b.zip"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "a.tar.gz"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteChecksums(directory, []string{"b.zip", "a.tar.gz"}, "SHA256SUMS.txt"); err != nil {
		t.Fatal(err)
	}
	manifest, _ := os.ReadFile(filepath.Join(directory, "SHA256SUMS.txt"))
	if strings.Index(string(manifest), "a.tar.gz") > strings.Index(string(manifest), "b.zip") {
		t.Fatalf("checksum manifest is not sorted:\n%s", manifest)
	}
	if err := VerifyChecksums(directory, "SHA256SUMS.txt"); err != nil {
		t.Fatalf("valid checksums failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "a.tar.gz"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksums(directory, "SHA256SUMS.txt"); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("changed artifact was not rejected: %v", err)
	}
}

func assertTarGzip(t *testing.T, archive []byte, name string, expected []byte, timestamp time.Time) {
	t.Helper()
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	tarReader := tar.NewReader(gzipReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != name || header.Mode != 0o755 || !header.ModTime.Equal(timestamp) || !bytes.Equal(content, expected) {
		t.Fatalf("unexpected tar entry: %#v content=%q", header, content)
	}
	if _, err := tarReader.Next(); err != io.EOF {
		t.Fatalf("archive contains another entry: %v", err)
	}
}

func assertZIP(t *testing.T, archive []byte, name string, expected []byte, timestamp time.Time) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("zip contains %d entries", len(reader.File))
	}
	entry := reader.File[0]
	stream, err := entry.Open()
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	stream.Close()
	if entry.Name != name || entry.Mode().Perm() != 0o755 || !entry.Modified.Equal(timestamp) || !bytes.Equal(content, expected) {
		t.Fatalf("unexpected zip entry: %#v content=%q", entry.FileHeader, content)
	}
}
