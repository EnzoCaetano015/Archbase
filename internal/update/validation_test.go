package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
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

func TestValidateRealVersionExecutable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	output := filepath.Join(t.TempDir(), "arc")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
		output += ".exe"
	}
	command := exec.CommandContext(ctx, goBinary, "build", "-o", output, "./testdata/version")
	command.Env = append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build version fixture: %v %s", err, data)
	}
	t.Setenv("ARC_UPDATE_TEST_VERSION", "2.0.0")
	if err := validateVersion(ctx, output, "2.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := validateVersion(ctx, output, "3.0.0"); err == nil {
		t.Fatal("mismatched version accepted")
	}
	t.Setenv("ARC_UPDATE_TEST_VERSION", "oversized")
	if err := validateVersion(ctx, output, "2.0.0"); err == nil {
		t.Fatal("oversized version accepted")
	}
	cancel()
	if err := validateVersion(ctx, output, "2.0.0"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestLockAndCancelledStaging(t *testing.T) {
	u, target, _ := fixture(t, "linux")
	if err := os.WriteFile(target+".update.lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Run(context.Background(), "1.0.0", Options{}); err == nil || !strings.Contains(err.Error(), "lock update") {
		t.Fatal(err)
	}
	if err := os.Remove(target + ".update.lock"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	u.validate = func(context.Context, string, string) error { cancel(); return nil }
	if _, err := u.Run(ctx, "1.0.0", Options{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "old" {
		t.Fatal("cancelled operation replaced executable")
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatal("cancelled operation leaked files", entries)
	}
}

func TestChangedInstallationIsNotOverwritten(t *testing.T) {
	u, target, _ := fixture(t, "linux")
	u.validate = func(context.Context, string, string) error {
		return os.WriteFile(target, []byte("changed by another installer"), 0755)
	}
	if _, err := u.Run(context.Background(), "1.0.0", Options{}); err == nil || !strings.Contains(err.Error(), "installation changed") {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "changed by another installer" {
		t.Fatal("concurrent change overwritten")
	}
}

func TestHTTPFailureBoundsAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/large":
			fmt.Fprint(w, "too large")
		case "/redirect":
			http.Redirect(w, r, "http://example.com/insecure", http.StatusFound)
		default:
			http.Error(w, "unavailable", http.StatusForbidden)
		}
	}))
	defer server.Close()
	u := New()
	for _, p := range []string{"/missing", "/large", "/redirect"} {
		if _, err := u.get(context.Background(), server.URL+p, 2); err == nil {
			t.Fatal("invalid response accepted", p)
		}
	}
}

func TestArchivesRejectLinksAndExtraFiles(t *testing.T) {
	var zipped bytes.Buffer
	z := zip.NewWriter(&zipped)
	header := &zip.FileHeader{Name: "arc.exe"}
	header.SetMode(os.ModeSymlink | 0777)
	f, err := z.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("outside"))
	z.Close()
	if _, err := extractBinary(zipped.Bytes(), "windows"); err == nil {
		t.Fatal("ZIP symlink accepted")
	}
	zipped.Reset()
	z = zip.NewWriter(&zipped)
	for _, name := range []string{"arc.exe", "extra"} {
		f, _ := z.Create(name)
		f.Write([]byte("x"))
	}
	z.Close()
	if _, err := extractBinary(zipped.Bytes(), "windows"); err == nil {
		t.Fatal("extra ZIP file accepted")
	}
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "arc", Typeflag: tar.TypeSymlink, Linkname: "outside"})
	tw.Close()
	gz.Close()
	if _, err := extractBinary(compressed.Bytes(), "linux"); err == nil {
		t.Fatal("TAR symlink accepted")
	}
	compressed.Reset()
	gz = gzip.NewWriter(&compressed)
	tw = tar.NewWriter(gz)
	for _, name := range []string{"arc", "extra"} {
		tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Size: 1})
		tw.Write([]byte("x"))
	}
	tw.Close()
	gz.Close()
	if _, err := extractBinary(compressed.Bytes(), "linux"); err == nil {
		t.Fatal("extra TAR file accepted")
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTamperedDownloadNeverRunsOrReplacesBinary(t *testing.T) {
	u, target, _ := fixture(t, "linux")
	u.client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		response, err := http.DefaultTransport.RoundTrip(r)
		if err == nil && strings.HasSuffix(r.URL.Path, ".tar.gz") {
			data, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil {
				return nil, readErr
			}
			data[0] ^= 1
			response.Body = io.NopCloser(bytes.NewReader(data))
		}
		return response, err
	})
	u.validate = func(context.Context, string, string) error { t.Fatal("unverified binary executed"); return nil }
	if _, err := u.Run(context.Background(), "1.0.0", Options{}); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(target); string(data) != "old" {
		t.Fatal("tampered release installed")
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatal("failed download leaked staging/lock")
	}
}

func TestConcurrentUpdatesAreRejected(t *testing.T) {
	u, _, _ := fixture(t, "linux")
	started := make(chan struct{})
	proceed := make(chan struct{})
	done := make(chan error, 1)
	u.validate = func(context.Context, string, string) error { close(started); <-proceed; return nil }
	go func() { _, err := u.Run(context.Background(), "1.0.0", Options{}); done <- err }()
	<-started
	_, err := u.Run(context.Background(), "1.0.0", Options{})
	close(proceed)
	if firstErr := <-done; firstErr != nil {
		t.Fatal(firstErr)
	}
	if err == nil || !strings.Contains(err.Error(), "lock update") {
		t.Fatalf("concurrent update not blocked: %v", err)
	}
}
