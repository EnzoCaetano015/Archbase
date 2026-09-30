package install

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const testVersion = "9.8.7"

func TestInstallerInstallsProtectsAndForceReplaces(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skipf("installer does not support test architecture %s", runtime.GOARCH)
	}
	root := repositoryRoot(t)
	assetName, archive := buildFixtureArchive(t, root)
	server := releaseServer(t, assetName, archive, "valid")

	home := t.TempDir()
	installDir := filepath.Join(home, "Archbase CLI")
	pathState := filepath.Join(home, "user-path.txt")

	output, err := runInstaller(t, root, server.URL, home, installDir, pathState, false, false, "")
	if err != nil {
		t.Fatalf("initial install failed: %v\n%s", err, output)
	}
	assertInstalledVersion(t, installDir)

	output, err = runInstaller(t, root, server.URL, home, installDir, pathState, false, true, testVersion)
	if err == nil || !strings.Contains(output, "already exists") {
		t.Fatalf("existing install was not protected: err=%v\n%s", err, output)
	}

	output, err = runInstaller(t, root, server.URL, home, installDir, pathState, true, true, "v"+testVersion)
	if err != nil {
		t.Fatalf("forced install failed: %v\n%s", err, output)
	}
	assertInstalledVersion(t, installDir)
	assertPathUpdatedOnce(t, home, installDir, pathState)
}

func TestInstallerRejectsInvalidOrMissingChecksumWithoutInstalling(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skipf("installer does not support test architecture %s", runtime.GOARCH)
	}
	root := repositoryRoot(t)
	assetName, archive := buildFixtureArchive(t, root)
	for _, test := range []struct {
		mode, expected string
	}{
		{mode: "invalid", expected: "checksum mismatch"},
		{mode: "missing", expected: "could not download"},
	} {
		t.Run(test.mode, func(t *testing.T) {
			server := releaseServer(t, assetName, archive, test.mode)
			home := t.TempDir()
			installDir := filepath.Join(home, "install")
			output, err := runInstaller(t, root, server.URL, home, installDir, filepath.Join(home, "path.txt"), false, true, testVersion)
			if err == nil || !strings.Contains(output, test.expected) {
				t.Fatalf("%s checksum was not rejected: err=%v\n%s", test.mode, err, output)
			}
			if _, statErr := os.Stat(installedBinary(installDir)); !os.IsNotExist(statErr) {
				t.Fatalf("invalid download left an installed binary: %v", statErr)
			}
		})
	}
}

func TestInstallerRejectsUnsupportedArchitecture(t *testing.T) {
	root := repositoryRoot(t)
	home := t.TempDir()
	installDir := filepath.Join(home, "install")
	environment := map[string]string{"ARCHBASE_TEST_ARCH": "unsupported"}
	var command *exec.Cmd
	if runtime.GOOS == "windows" {
		command = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "site", "install.ps1"), "-Version", testVersion, "-InstallDir", installDir)
	} else {
		command = exec.Command("sh", filepath.Join(root, "site", "install.sh"), "--version", testVersion, "--install-dir", installDir)
	}
	command.Env = mergedEnvironment(environment)
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "unsupported architecture") {
		t.Fatalf("unsupported architecture was not rejected: err=%v\n%s", err, output)
	}
}

func TestInstallerSelectsArm64Asset(t *testing.T) {
	requestedPath := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestedPath <- request.URL.Path
		http.NotFound(response, request)
	}))
	defer server.Close()

	root := repositoryRoot(t)
	installDir := filepath.Join(t.TempDir(), "install")
	environment := map[string]string{
		"ARCHBASE_RELEASE_BASE_URL": server.URL + "/download",
		"ARCHBASE_TEST_ARCH":        "arm64",
	}
	var command *exec.Cmd
	expectedAsset := "arc_v" + testVersion + "_windows_arm64.zip"
	if runtime.GOOS == "windows" {
		command = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "site", "install.ps1"), "-Version", testVersion, "-InstallDir", installDir)
	} else {
		targetOS := runtime.GOOS
		environment["ARCHBASE_TEST_OS"] = targetOS
		expectedAsset = "arc_v" + testVersion + "_" + targetOS + "_arm64.tar.gz"
		command = exec.Command("sh", filepath.Join(root, "site", "install.sh"), "--version", testVersion, "--install-dir", installDir)
	}
	command.Env = mergedEnvironment(environment)
	if output, err := command.CombinedOutput(); err == nil {
		t.Fatalf("installer unexpectedly succeeded without an asset: %s", output)
	}
	var path string
	select {
	case path = <-requestedPath:
	case <-time.After(5 * time.Second):
		t.Fatal("installer did not request a release asset")
	}
	if !strings.HasSuffix(path, "/"+expectedAsset) {
		t.Fatalf("installer requested %q, expected asset %q", path, expectedAsset)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func buildFixtureArchive(t *testing.T, root string) (string, []byte) {
	t.Helper()
	buildDir := t.TempDir()
	binaryName := "arc"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(buildDir, binaryName)
	ldflags := "-s -w -buildid= -X github.com/EnzoCaetano015/Archbase/internal/version.Value=" + testVersion
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", binaryPath, "./cmd/arc")
	command.Dir = root
	command.Env = mergedEnvironment(map[string]string{"CGO_ENABLED": "0"})
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fixture arc: %v\n%s", err, output)
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	assetName := fmt.Sprintf("arc_v%s_%s_%s.tar.gz", testVersion, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName = fmt.Sprintf("arc_v%s_windows_%s.zip", testVersion, runtime.GOARCH)
		return assetName, zipArchive(t, binaryName, binary)
	}
	return assetName, tarGzipArchive(t, binaryName, binary)
}

func tarGzipArchive(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func zipArchive(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o755)
	entry, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func releaseServer(t *testing.T, assetName string, archive []byte, checksumMode string) *httptest.Server {
	t.Helper()
	digest := fmt.Sprintf("%x", sha256.Sum256(archive))
	if checksumMode == "invalid" {
		digest = strings.Repeat("0", 64)
	}
	manifest := []byte(fmt.Sprintf("%s  %s\n", digest, assetName))
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/latest":
			response.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(response, `{"tag_name":"v%s"}`, testVersion)
		case "/download/v" + testVersion + "/" + assetName:
			response.Write(archive)
		case "/download/v" + testVersion + "/arc_v" + testVersion + "_SHA256SUMS.txt":
			if checksumMode == "missing" {
				http.NotFound(response, request)
				return
			}
			response.Write(manifest)
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func runInstaller(t *testing.T, root, serverURL, home, installDir, pathState string, force, explicitVersion bool, version string) (string, error) {
	t.Helper()
	environment := map[string]string{
		"ARCHBASE_RELEASE_API_URL":     serverURL + "/latest",
		"ARCHBASE_RELEASE_BASE_URL":    serverURL + "/download",
		"ARCHBASE_TEST_USER_PATH_FILE": pathState,
		"HOME":                          home,
		"SHELL":                         "/bin/sh",
	}
	var command *exec.Cmd
	if runtime.GOOS == "windows" {
		arguments := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(root, "site", "install.ps1"), "-InstallDir", installDir}
		if explicitVersion {
			arguments = append(arguments, "-Version", version)
		}
		if force {
			arguments = append(arguments, "-Force")
		}
		command = exec.Command("powershell.exe", arguments...)
	} else {
		arguments := []string{filepath.Join(root, "site", "install.sh"), "--install-dir", installDir}
		if explicitVersion {
			arguments = append(arguments, "--version", version)
		}
		if force {
			arguments = append(arguments, "--force")
		}
		command = exec.Command("sh", arguments...)
	}
	command.Env = mergedEnvironment(environment)
	output, err := command.CombinedOutput()
	return string(output), err
}

func assertInstalledVersion(t *testing.T, installDir string) {
	t.Helper()
	command := exec.Command(installedBinary(installDir), "version")
	output, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "arc "+testVersion {
		t.Fatalf("unexpected installed version: err=%v output=%q", err, output)
	}
}

func assertPathUpdatedOnce(t *testing.T, home, installDir, pathState string) {
	t.Helper()
	var content []byte
	var err error
	if runtime.GOOS == "windows" {
		content, err = os.ReadFile(pathState)
	} else {
		content, err = os.ReadFile(filepath.Join(home, ".profile"))
	}
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		matches := 0
		installInfo, statErr := os.Stat(installDir)
		for _, entry := range strings.Split(string(content), ";") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if strings.EqualFold(filepath.Clean(entry), filepath.Clean(installDir)) {
				matches++
				continue
			}
			if statErr == nil {
				entryInfo, entryErr := os.Stat(entry)
				if entryErr == nil && os.SameFile(installInfo, entryInfo) {
					matches++
				}
			}
		}
		if matches != 1 {
			t.Fatalf("PATH entry was not written exactly once: install directory %q, PATH %q", installDir, content)
		}
		return
	}
	if strings.Count(string(content), installDir) != 1 {
		t.Fatalf("PATH entry was not written exactly once: %q", content)
	}
}

func installedBinary(installDir string) string {
	name := "arc"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(installDir, name)
}

func mergedEnvironment(overrides map[string]string) []string {
	values := map[string]string{}
	for _, item := range os.Environ() {
		key, value, found := strings.Cut(item, "=")
		if found {
			values[strings.ToUpper(key)] = key + "=" + value
		}
	}
	for key, value := range overrides {
		values[strings.ToUpper(key)] = key + "=" + value
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
