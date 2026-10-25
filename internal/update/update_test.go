package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func archiveFixture(t *testing.T, goos, name string, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if goos == "windows" {
		w := zip.NewWriter(&b)
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(data); err != nil {
			t.Fatal(err)
		}
		if err = w.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		gz := gzip.NewWriter(&b)
		w := tar.NewWriter(gz)
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return b.Bytes()
}

func fixture(t *testing.T, goos string, architectures ...string) (*Updater, string, *atomic.Int64) {
	t.Helper()
	root := t.TempDir()
	target := filepath.Join(root, "arc")
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	name, ext := "arc", ".tar.gz"
	if goos == "windows" {
		name = "arc.exe"
		ext = ".zip"
	}
	archive := archiveFixture(t, goos, name, []byte("new"))
	arch := "amd64"
	if len(architectures) > 0 {
		arch = architectures[0]
	}
	asset := "arc_v2.0.0_" + goos + "_" + arch + ext
	digest := sha256.Sum256(archive)
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/releases/latest", "/releases/tags/v2.0.0":
			fmt.Fprint(w, `{"tag_name":"v2.0.0"}`)
		case "/download/v2.0.0/arc_v2.0.0_SHA256SUMS.txt":
			fmt.Fprintf(w, "%x  %s\n", digest, asset)
		case "/download/v2.0.0/" + asset:
			w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	u := New()
	u.api = server.URL + "/releases"
	u.downloads = server.URL + "/download"
	u.goos = goos
	u.goarch = arch
	u.executable = func() (string, error) { return target, nil }
	u.validate = func(context.Context, string, string) error { return nil }
	return u, target, &requests
}

func TestUpdatePlatformsCheckAndNoDowngrade(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			t.Run(goos+"/"+arch, func(t *testing.T) {
				u, target, requests := fixture(t, goos, arch)
				checked, err := u.Run(context.Background(), "1.0.0", Options{Check: true})
				if err != nil || !checked.Available || requests.Load() != 1 {
					t.Fatalf("check: %#v %v", checked, err)
				}
				if b, _ := os.ReadFile(target); string(b) != "old" {
					t.Fatal("check wrote executable")
				}
				for _, version := range []string{"2.0.0", "3.0.0"} {
					result, err := u.Run(context.Background(), version, Options{})
					if err != nil || result.Updated || result.Available {
						t.Fatalf("%s: %#v %v", version, result, err)
					}
				}
				if _, err = u.Run(context.Background(), "3.0.0", Options{Version: "v2.0.0", Force: true}); err == nil {
					t.Fatal("downgrade allowed")
				}
				result, err := u.Run(context.Background(), "1.0.0", Options{Version: "v2.0.0"})
				if err != nil || !result.Updated {
					t.Fatalf("update: %#v %v", result, err)
				}
				if b, _ := os.ReadFile(target); string(b) != "new" {
					t.Fatal("new executable missing")
				}
				entries, _ := os.ReadDir(filepath.Dir(target))
				if len(entries) != 1 {
					t.Fatalf("temporary files remain: %v", entries)
				}
			})
		}
	}
}

func TestDevelopmentBuildAndForcedReinstall(t *testing.T) {
	u, target, _ := fixture(t, "linux")
	if _, err := u.Run(context.Background(), "dev", Options{}); err == nil {
		t.Fatal("dev build replaced without consent")
	}
	if _, err := u.Run(context.Background(), "dev", Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Run(context.Background(), "2.0.0", Options{Force: true}); err != nil {
		t.Fatal(err)
	}
}

func TestValidationAndPromotionFailuresPreserveExistingExecutable(t *testing.T) {
	for _, failure := range []string{"validation", "unix-rename", "windows-first", "windows-second"} {
		t.Run(failure, func(t *testing.T) {
			goos := "linux"
			if strings.HasPrefix(failure, "windows") {
				goos = "windows"
			}
			u, target, _ := fixture(t, goos)
			if failure == "validation" {
				u.validate = func(context.Context, string, string) error { return errors.New("blocked binary") }
			} else {
				count := 0
				u.rename = func(a, b string) error {
					count++
					if failure == "windows-second" && count == 2 || failure != "windows-second" && count == 1 {
						return errors.New("injected promotion error")
					}
					return os.Rename(a, b)
				}
			}
			if _, err := u.Run(context.Background(), "1.0.0", Options{}); err == nil {
				t.Fatal("expected failure")
			}
			if b, _ := os.ReadFile(target); string(b) != "old" {
				t.Fatal("previous executable lost")
			}
			entries, _ := os.ReadDir(filepath.Dir(target))
			if len(entries) != 1 {
				t.Fatalf("staging/lock leaked: %v", entries)
			}
		})
	}
}

func TestFailedRollbackRetainsRecoveryCopy(t *testing.T) {
	u, target, _ := fixture(t, "windows")
	count := 0
	u.rename = func(a, b string) error {
		count++
		if count > 1 {
			return errors.New("blocked")
		}
		return os.Rename(a, b)
	}
	_, err := u.Run(context.Background(), "1.0.0", Options{})
	if err == nil || !strings.Contains(err.Error(), "recover previous executable") {
		t.Fatal(err)
	}
	backups, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".arc-update-*", "previous.exe"))
	if len(backups) != 1 {
		t.Fatal("recovery copy deleted")
	}
	if b, _ := os.ReadFile(backups[0]); string(b) != "old" {
		t.Fatal("invalid recovery copy")
	}
}

func TestChecksumAndArchiveValidation(t *testing.T) {
	archive := []byte("payload")
	digest := sha256.Sum256(archive)
	valid := fmt.Sprintf("%x  asset.zip\n", digest)
	if err := verifyChecksum(archive, []byte(valid), "asset.zip"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", valid + valid, "bad asset.zip", strings.Repeat("0", 64) + " asset.zip"} {
		if err := verifyChecksum(archive, []byte(bad), "asset.zip"); err == nil {
			t.Fatal("bad checksum accepted", bad)
		}
	}
	for _, goos := range []string{"linux", "windows"} {
		name := "arc"
		if goos == "windows" {
			name += ".exe"
		}
		for _, entry := range []string{"../" + name, "folder/" + name, "/" + name} {
			if _, err := extractBinary(archiveFixture(t, goos, entry, []byte("binary")), goos); err == nil {
				t.Fatal("unsafe archive accepted", entry)
			}
		}
		if _, err := extractBinary(archiveFixture(t, goos, name, nil), goos); err == nil {
			t.Fatal("empty binary accepted")
		}
	}
}

func TestReleaseErrorsAndCancellation(t *testing.T) {
	for _, body := range []string{`not json`, `{"tag_name":"v2.0.0-rc1"}`, `{"tag_name":"v2.0.0","draft":true}`, `{"tag_name":"v2.0.0","prerelease":true}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		u := New()
		u.api = server.URL
		_, err := u.Run(context.Background(), "1.0.0", Options{})
		server.Close()
		if err == nil {
			t.Fatal("bad release accepted", body)
		}
	}
	u, _, _ := fixture(t, "linux")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := u.Run(ctx, "1.0.0", Options{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := u.Run(context.Background(), "1.0.0", Options{Version: "../../bad"}); err == nil {
		t.Fatal("invalid tag accepted")
	}
	u.goarch = "386"
	if _, err := u.Run(context.Background(), "1.0.0", Options{}); err == nil {
		t.Fatal("unsupported architecture accepted")
	}
	for _, address := range []string{"http://example.com", "https://user:secret@example.com", "file:///tmp/release"} {
		if _, err := u.get(context.Background(), address, 10); err == nil {
			t.Fatal("unsafe URL", address)
		}
	}
	for _, pair := range [][2]string{{"1.10.0", "1.9.99"}, {"2.0.0", "1.999999999999999999999999.0"}} {
		if compareVersions(pair[0], pair[1]) <= 0 {
			t.Fatal("version comparison", pair)
		}
	}
}

// This subprocess exercises replacement while the target executable is running,
// including the Windows rename/backup path. No installed user binary is touched.
func TestRunningExecutableReplacement(t *testing.T) {
	if os.Getenv("ARC_UPDATE_HELPER") == "1" {
		u := New()
		u.goarch = "amd64" // the fixture payload is architecture-neutral test data
		u.api = os.Getenv("ARC_UPDATE_FIXTURE") + "/releases"
		u.downloads = os.Getenv("ARC_UPDATE_FIXTURE") + "/download"
		u.validate = func(context.Context, string, string) error { return nil }
		_, err := u.Run(context.Background(), "1.0.0", Options{})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	u, target, _ := fixture(t, runtime.GOOS)
	current, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(target, data, 0755); err != nil {
		t.Fatal(err)
	}
	// Windows requires an executable suffix for launching the copied test process.
	if runtime.GOOS == "windows" {
		if err = os.Rename(target, target+".exe"); err != nil {
			t.Fatal(err)
		}
		target += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, target, "-test.run=^TestRunningExecutableReplacement$")
	cmd.Env = append(os.Environ(), "ARC_UPDATE_HELPER=1", "ARC_UPDATE_FIXTURE="+strings.TrimSuffix(u.api, "/releases"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("running replacement: %v %s", err, output)
	}
	if result, _ := os.ReadFile(target); string(result) != "new" {
		t.Fatal("running executable was not replaced")
	}
}
