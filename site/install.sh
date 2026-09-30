#!/bin/sh

set -eu

repository="EnzoCaetano015/Archbase"
release_api_url="${ARCHBASE_RELEASE_API_URL:-https://api.github.com/repos/${repository}/releases/latest}"
release_base_url="${ARCHBASE_RELEASE_BASE_URL:-https://github.com/${repository}/releases/download}"
requested_version=""
install_dir="${HOME}/.local/bin"
force=0
temporary_dir=""
staged_target=""

usage() {
    cat <<'EOF'
Install the Archbase CLI on Linux. macOS support is an unsigned preview.

Usage:
  install.sh [--version <vMAJOR.MINOR.PATCH>] [--install-dir <path>] [--force]

Options:
  --version       Install a specific stable version. Defaults to the latest release.
  --install-dir   Install arc in this directory. Defaults to $HOME/.local/bin.
  --force         Replace an existing arc executable.
  --help          Show this help.
EOF
}

fail() {
    printf 'Archbase install error: %s\n' "$*" >&2
    exit 1
}

cleanup() {
    if [ -n "$staged_target" ]; then
        rm -f -- "$staged_target"
    fi
    if [ -n "$temporary_dir" ]; then
        rm -rf -- "$temporary_dir"
    fi
}

trap cleanup EXIT
trap 'exit 1' HUP INT TERM

while [ "$#" -gt 0 ]; do
    case "$1" in
        --version)
            [ "$#" -ge 2 ] || fail "--version requires a value"
            requested_version=$2
            shift 2
            ;;
        --install-dir)
            [ "$#" -ge 2 ] || fail "--install-dir requires a value"
            install_dir=$2
            shift 2
            ;;
        --force)
            force=1
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            fail "unknown option: $1"
            ;;
    esac
done

[ -n "$install_dir" ] || fail "install directory must not be empty"

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

detected_os=${ARCHBASE_TEST_OS:-$(uname -s)}
case "$detected_os" in
    Linux|linux) target_os=linux ;;
    Darwin|darwin) target_os=darwin ;;
    *) fail "unsupported operating system: $detected_os" ;;
esac

if [ "$target_os" = darwin ]; then
    printf '%s\n' 'WARNING: macOS support is an unsigned preview.' >&2
    printf '%s\n' 'The binary is not signed or notarized, so macOS security controls may warn or block it.' >&2
    printf '%s\n' 'Do not disable system security protections; continue only if you trust this source.' >&2
fi

detected_arch=${ARCHBASE_TEST_ARCH:-$(uname -m)}
case "$detected_arch" in
    x86_64|amd64) target_arch=amd64 ;;
    arm64|aarch64) target_arch=arm64 ;;
    *) fail "unsupported architecture: $detected_arch" ;;
esac

if [ -n "$requested_version" ]; then
    case "$requested_version" in
        v*) release_tag=$requested_version ;;
        *) release_tag="v${requested_version}" ;;
    esac
else
    release_json=$(curl -fsSL \
        -H 'Accept: application/vnd.github+json' \
        -H 'X-GitHub-Api-Version: 2022-11-28' \
        -H 'User-Agent: archbase-installer' \
        "$release_api_url") || fail "could not resolve the latest Archbase release"
    release_tag=$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | sed -n '1p')
    [ -n "$release_tag" ] || fail "latest release response did not contain tag_name"
fi

printf '%s\n' "$release_tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' || \
    fail "version must match vMAJOR.MINOR.PATCH"
version=${release_tag#v}

asset="arc_v${version}_${target_os}_${target_arch}.tar.gz"
checksum="arc_v${version}_SHA256SUMS.txt"
download_url="${release_base_url}/${release_tag}"
target="${install_dir}/arc"

if [ -d "$target" ]; then
    fail "installation target is a directory: $target"
fi
if { [ -e "$target" ] || [ -L "$target" ]; } && [ "$force" -ne 1 ]; then
    fail "arc already exists at $target; rerun with --force to replace it"
fi

temporary_dir=$(mktemp -d "${TMPDIR:-/tmp}/archbase-install.XXXXXX") || fail "could not create a temporary directory"
archive_path="${temporary_dir}/${asset}"
checksum_path="${temporary_dir}/${checksum}"
extract_dir="${temporary_dir}/extract"

printf 'Downloading Archbase %s for %s/%s...\n' "$version" "$target_os" "$target_arch"
curl -fsSL "$download_url/$asset" -o "$archive_path" || fail "could not download $asset"
curl -fsSL "$download_url/$checksum" -o "$checksum_path" || fail "could not download $checksum"

expected_hash=$(awk -v name="$asset" '$2 == name { print $1 }' "$checksum_path")
case "$expected_hash" in
    *[!0-9A-Fa-f]*|'') fail "checksum manifest does not contain a valid hash for $asset" ;;
esac
[ "${#expected_hash}" -eq 64 ] || fail "checksum manifest contains an invalid SHA-256 hash for $asset"

if command -v sha256sum >/dev/null 2>&1; then
    actual_hash=$(sha256sum "$archive_path" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
    actual_hash=$(shasum -a 256 "$archive_path" | awk '{ print $1 }')
else
    fail "sha256sum or shasum is required"
fi

expected_hash=$(printf '%s' "$expected_hash" | tr 'A-F' 'a-f')
actual_hash=$(printf '%s' "$actual_hash" | tr 'A-F' 'a-f')
[ "$actual_hash" = "$expected_hash" ] || fail "SHA-256 checksum mismatch for $asset"

mkdir -p "$extract_dir"
tar -xzf "$archive_path" -C "$extract_dir" || fail "could not extract $asset"
[ -f "$extract_dir/arc" ] || fail "release archive does not contain arc"
chmod 0755 "$extract_dir/arc"

installed_version=$($extract_dir/arc version 2>/dev/null) || fail "downloaded arc executable could not be run"
[ "$installed_version" = "arc $version" ] || fail "downloaded executable reported '$installed_version', expected 'arc $version'"

mkdir -p "$install_dir" || fail "could not create $install_dir"
staged_target="${install_dir}/.arc.$$.tmp"
cp "$extract_dir/arc" "$staged_target" || fail "could not stage arc in $install_dir"
chmod 0755 "$staged_target"

if [ -d "$target" ]; then
    fail "installation target became a directory during installation: $target"
fi
if { [ -e "$target" ] || [ -L "$target" ]; } && [ "$force" -ne 1 ]; then
    fail "arc appeared at $target during installation; rerun with --force to replace it"
fi
mv -f "$staged_target" "$target" || fail "could not install arc at $target"
staged_target=""

path_contains_install_dir=0
if printf '%s' "${PATH:-}" | tr ':' '\n' | grep -Fx -- "$install_dir" >/dev/null 2>&1; then
    path_contains_install_dir=1
fi

profile_changed=0
if [ "$path_contains_install_dir" -ne 1 ]; then
    shell_name=$(basename "${SHELL:-sh}")
    quoted_install_dir=$(printf '%s' "$install_dir" | sed "s/'/'\\\\''/g")
    case "$shell_name" in
        fish)
            profile_file="${HOME}/.config/fish/config.fish"
            profile_line="fish_add_path '${quoted_install_dir}'"
            mkdir -p "$(dirname "$profile_file")"
            ;;
        bash)
            profile_file="${HOME}/.bashrc"
            profile_line="export PATH='${quoted_install_dir}':\"\$PATH\""
            ;;
        zsh)
            profile_file="${HOME}/.zshrc"
            profile_line="export PATH='${quoted_install_dir}':\"\$PATH\""
            ;;
        *)
            profile_file="${HOME}/.profile"
            profile_line="export PATH='${quoted_install_dir}':\"\$PATH\""
            ;;
    esac
    if ! grep -F -- "$install_dir" "$profile_file" >/dev/null 2>&1; then
        {
            printf '\n# Archbase CLI\n'
            printf '%s\n' "$profile_line"
        } >> "$profile_file" || fail "could not update PATH in $profile_file"
        profile_changed=1
    fi
fi

printf 'Archbase %s installed at %s\n' "$version" "$target"
if [ "$profile_changed" -eq 1 ]; then
    printf 'PATH was updated. Open a new terminal before running arc.\n'
elif [ "$path_contains_install_dir" -ne 1 ]; then
    printf 'Open a new terminal before running arc.\n'
fi
