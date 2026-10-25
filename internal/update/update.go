// Package update installs verified stable release binaries in place.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	archfs "github.com/EnzoCaetano015/Archbase/internal/filesystem"
	"github.com/EnzoCaetano015/Archbase/internal/release"
)

const repository = "EnzoCaetano015/Archbase"
const maxArchive = 100 << 20

type Options struct {
	Version string
	Check   bool
	Force   bool
}

type Result struct {
	Current, Latest, Executable string
	Available, Updated          bool
	Warnings                    []string
}

type Updater struct {
	client                       *http.Client
	api, downloads, goos, goarch string
	executable                   func() (string, error)
	validate                     func(context.Context, string, string) error
	rename                       func(string, string) error
}

func New() *Updater {
	return &Updater{
		client: &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many release redirects")
			}
			if req.URL.Scheme != "https" || req.URL.User != nil {
				return errors.New("release redirects must use HTTPS without credentials")
			}
			return nil
		}},
		api:       "https://api.github.com/repos/" + repository + "/releases",
		downloads: "https://github.com/" + repository + "/releases/download",
		goos:      runtime.GOOS, goarch: runtime.GOARCH, executable: os.Executable,
		validate: validateVersion, rename: os.Rename,
	}
}

func (u *Updater) Run(ctx context.Context, current string, options Options) (result Result, err error) {
	result.Current = current
	if (u.goos != "linux" && u.goos != "darwin" && u.goos != "windows") || (u.goarch != "amd64" && u.goarch != "arm64") {
		return result, fmt.Errorf("self-update is not supported on %s/%s", u.goos, u.goarch)
	}
	endpoint := u.api + "/latest"
	if options.Version != "" {
		if _, err = release.VersionFromTag(options.Version); err != nil {
			return result, err
		}
		endpoint = u.api + "/tags/" + options.Version
	}
	data, err := u.get(ctx, endpoint, 2<<20)
	if err != nil {
		return result, err
	}
	var remote struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err = json.Unmarshal(data, &remote); err != nil {
		return result, fmt.Errorf("invalid GitHub release response: %w", err)
	}
	if remote.Draft || remote.Prerelease {
		return result, errors.New("only published stable releases may be installed")
	}
	latest, err := release.VersionFromTag(remote.Tag)
	if err != nil {
		return result, err
	}
	if options.Version != "" && options.Version != remote.Tag {
		return result, errors.New("GitHub returned a different release than requested")
	}
	result.Latest = latest
	currentVersion := strings.TrimPrefix(current, "v")
	_, versionErr := release.VersionFromTag("v" + currentVersion)
	comparison := 1
	if versionErr == nil {
		comparison = compareVersions(latest, currentVersion)
	}
	if comparison < 0 && options.Version != "" {
		return result, fmt.Errorf("refusing downgrade from %s to %s", current, latest)
	}
	result.Available = comparison > 0
	if options.Check || comparison < 0 || comparison == 0 && !options.Force {
		return result, nil
	}
	if versionErr != nil && !options.Force {
		return result, errors.New("current build has no stable version; use --force to explicitly replace this development build")
	}
	if u.goos != "linux" {
		result.Warnings = append(result.Warnings, "macOS and Windows releases are unsigned previews; security policies may block the new executable. Do not disable security protections.")
	}
	executable, err := u.executable()
	if err != nil {
		return result, fmt.Errorf("locate running executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return result, fmt.Errorf("resolve executable %q: %w", executable, err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return result, err
	}
	result.Executable = executable
	info, err := os.Lstat(executable)
	if err != nil || !info.Mode().IsRegular() {
		return result, fmt.Errorf("update target %q must be a regular executable", executable)
	}
	lock, err := os.OpenFile(executable+".update.lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, fmt.Errorf("lock update %q (another update may be running): %w", executable, err)
	}
	defer func() {
		_ = lock.Close()
		if cleanupErr := os.Remove(executable + ".update.lock"); cleanupErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not remove update lock %s.update.lock: %v", executable, cleanupErr))
		}
	}()
	stage, err := os.MkdirTemp(filepath.Dir(executable), ".arc-update-*")
	if err != nil {
		return result, fmt.Errorf("stage update beside %q; check installation-directory permissions: %w", executable, err)
	}
	defer func() {
		if stage != "" {
			if cleanupErr := os.RemoveAll(stage); cleanupErr != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Update backup/temporary files retained at %s; remove this directory after closing running arc processes.", stage))
			}
		}
	}()
	extension := ".tar.gz"
	if u.goos == "windows" {
		extension = ".zip"
	}
	archiveName := fmt.Sprintf("arc_%s_%s_%s%s", remote.Tag, u.goos, u.goarch, extension)
	checksums, err := u.get(ctx, u.downloads+"/"+remote.Tag+"/arc_"+remote.Tag+"_SHA256SUMS.txt", 1<<20)
	if err != nil {
		return result, err
	}
	archive, err := u.get(ctx, u.downloads+"/"+remote.Tag+"/"+archiveName, maxArchive)
	if err != nil {
		return result, err
	}
	if err = verifyChecksum(archive, checksums, archiveName); err != nil {
		return result, err
	}
	binary, err := extractBinary(archive, u.goos)
	if err != nil {
		return result, err
	}
	staged := filepath.Join(stage, "arc")
	if u.goos == "windows" {
		staged += ".exe"
	}
	if err = archfs.WriteFileAtomic(staged, binary, false); err != nil {
		return result, fmt.Errorf("write staged executable: %w", err)
	}
	if err = os.Chmod(staged, info.Mode().Perm()|0700); err != nil {
		return result, fmt.Errorf("set staged executable permissions: %w", err)
	}
	if err = u.validate(ctx, staged, latest); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	// Do not replace an installation modified by an installer during download.
	currentInfo, statErr := os.Lstat(executable)
	if statErr != nil || !os.SameFile(info, currentInfo) || !currentInfo.Mode().IsRegular() || info.Size() != currentInfo.Size() || !info.ModTime().Equal(currentInfo.ModTime()) {
		return result, errors.New("installation changed during download; retry the update")
	}
	if u.goos == "windows" {
		backup := filepath.Join(stage, "previous.exe")
		if err = u.rename(executable, backup); err != nil {
			return result, fmt.Errorf("move previous executable %q; close other arc processes and check permissions: %w", executable, err)
		}
		if err = u.rename(staged, executable); err != nil {
			if rollbackErr := u.rename(backup, executable); rollbackErr != nil {
				// Never clean up the sole remaining copy after a failed rollback.
				stage = "" // cleanup is guarded below by preserving the backup directory
				return result, fmt.Errorf("install failed: %v; rollback failed: %v; recover previous executable from %s", err, rollbackErr, backup)
			}
			return result, fmt.Errorf("install update (previous version restored): %w", err)
		}
	} else if err = u.rename(staged, executable); err != nil {
		return result, fmt.Errorf("replace executable %q (previous version preserved): %w", executable, err)
	}
	result.Updated = true
	return result, nil
}

func (u *Updater) get(ctx context.Context, address string, limit int64) ([]byte, error) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.User != nil || parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost")) {
		return nil, errors.New("release URL must use HTTPS")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Archbase-self-update")
	req.Header.Set("Accept", "application/vnd.github+json")
	response, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release request returned HTTP %d for %s", response.StatusCode, parsed.Path)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("release response exceeds size limit")
	}
	return data, nil
}

func verifyChecksum(archive, manifest []byte, name string) error {
	found := ""
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		if found != "" {
			return errors.New("duplicate archive checksum")
		}
		decoded, err := hex.DecodeString(fields[0])
		if err != nil || len(decoded) != sha256.Size {
			return errors.New("invalid archive checksum")
		}
		found = strings.ToLower(fields[0])
	}
	if found == "" {
		return fmt.Errorf("checksum missing for %s", name)
	}
	digest := sha256.Sum256(archive)
	if hex.EncodeToString(digest[:]) != found {
		return errors.New("archive SHA-256 checksum mismatch; executable was not changed")
	}
	return nil
}

func compareVersions(a, b string) int {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		if len(left[i]) > len(right[i]) {
			return 1
		}
		if len(left[i]) < len(right[i]) {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
		if left[i] < right[i] {
			return -1
		}
	}
	return 0
}

func validateVersion(ctx context.Context, executable, version string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var output boundedOutput
	command := exec.CommandContext(ctx, executable, "version")
	command.Stdout = &output
	err := command.Run()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("validate downloaded executable (security policies may block unsigned previews): %w", err)
	}
	if strings.TrimSpace(string(output.data)) != "arc "+version {
		return errors.New("downloaded executable reports a different version")
	}
	return nil
}

type boundedOutput struct{ data []byte }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(b.data)+len(p) > 1024 {
		return 0, errors.New("version output exceeds size limit")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
