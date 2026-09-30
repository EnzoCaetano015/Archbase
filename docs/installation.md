# Installing Archbase

Archbase publishes a self-contained `arc` executable for Linux, macOS, and Windows. The official installers detect the operating system and architecture, download only the matching release archive, verify its SHA-256 checksum, install the executable, and configure the user `PATH`.

## Linux and macOS

```bash
curl -fsSL https://archbase.caetanodev.com/install.sh | sh
```

The installer supports Linux on x86-64 and ARM64, macOS on Intel and Apple silicon, and installs to `$HOME/.local/bin` by default. Open a new terminal after the first installation, then verify it:

```bash
arc version
```

Install a specific stable version:

```bash
curl -fsSL https://archbase.caetanodev.com/install.sh | sh -s -- --version v0.3.0
```

Choose another installation directory:

```bash
curl -fsSL https://archbase.caetanodev.com/install.sh | sh -s -- --install-dir "$HOME/bin"
```

Replace an existing installation explicitly:

```bash
curl -fsSL https://archbase.caetanodev.com/install.sh | sh -s -- --force
```

Available options: `--version`, `--install-dir`, `--force`, and `--help`.

## Windows PowerShell

```powershell
powershell -ExecutionPolicy Bypass -c "irm https://archbase.caetanodev.com/install.ps1 | iex"
```

The installer supports Windows on x86-64 and ARM64 and installs to `%LOCALAPPDATA%\Programs\Archbase` by default. Open a new terminal after the first installation, then verify it:

```powershell
arc version
```

To install a specific version, choose another directory, or replace an existing installation, load the installer as a script block:

```powershell
$install = [scriptblock]::Create((irm https://archbase.caetanodev.com/install.ps1))
& $install -Version v0.3.0
& $install -InstallDir "$HOME\bin"
& $install -Force
```

Available parameters: `-Version`, `-InstallDir`, `-Force`, and `-Help`.

## Installer safety

Both installers:

- accept only stable `vMAJOR.MINOR.PATCH` releases;
- download release assets from the official GitHub repository;
- verify the selected archive against the published SHA-256 manifest;
- validate the embedded CLI version before modifying the destination;
- stage the executable in the destination directory before atomically promoting it;
- preserve an existing executable unless `--force` or `-Force` is provided;
- update the user `PATH` without duplicating entries;
- remove temporary files after success or failure.

The installer source is available at [`site/install.sh`](../site/install.sh) and [`site/install.ps1`](../site/install.ps1) for inspection before execution.

## System changes

On Linux and macOS, the default installation creates `$HOME/.local/bin` when needed and places the executable at `$HOME/.local/bin/arc`. If that directory is not already on `PATH`, the installer appends one marked `# Archbase CLI` entry to the configuration for the detected shell: `.bashrc`, `.zshrc`, `.config/fish/config.fish`, or `.profile`.

On Windows, the default installation creates `%LOCALAPPDATA%\Programs\Archbase`, places `arc.exe` there, and adds that directory to the current user's `PATH` only when it is absent. The installer does not modify the machine-wide `PATH` and does not require administrator privileges. Existing `PATH` entries are preserved on every platform.

## Uninstall

Before uninstalling, close terminals and tools that may be running `arc`.

On Linux or macOS, remove the executable:

```bash
rm "$HOME/.local/bin/arc"
```

If the installer added a `PATH` entry, open the shell file reported during installation and remove the `# Archbase CLI` line together with the following `export PATH=...` or `fish_add_path ...` line. Do not remove `$HOME/.local/bin` from `PATH` when other installed programs use that directory.

On Windows PowerShell, remove the executable and remove only the matching Archbase directory from the user `PATH`:

```powershell
$installDir = Join-Path $env:LOCALAPPDATA 'Programs\Archbase'
Remove-Item (Join-Path $installDir 'arc.exe') -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$entries = @($userPath -split ';' | Where-Object {
    $_ -and -not [string]::Equals(
        $_.TrimEnd('\'),
        $installDir.TrimEnd('\'),
        [StringComparison]::OrdinalIgnoreCase
    )
})
[Environment]::SetEnvironmentVariable('Path', ($entries -join ';'), 'User')
if ((Test-Path $installDir) -and -not (Get-ChildItem $installDir -Force)) {
    Remove-Item $installDir
}
```

Open a new terminal after changing `PATH`. For a custom installation directory, substitute that exact directory and keep it on `PATH` if it contains other programs.

## Manual installation

Manual archives and `SHA256SUMS.txt` are available on the [GitHub Releases page](https://github.com/EnzoCaetano015/Archbase/releases/latest).

| Operating system | Architecture | Asset suffix |
| --- | --- | --- |
| Linux | x86-64 | `linux_amd64.tar.gz` |
| Linux | ARM64 | `linux_arm64.tar.gz` |
| macOS | Intel | `darwin_amd64.tar.gz` |
| macOS | Apple silicon | `darwin_arm64.tar.gz` |
| Windows | x86-64 | `windows_amd64.zip` |
| Windows | ARM64 | `windows_arm64.zip` |

Verify the archive against the release checksum manifest before extracting it. Place `arc` or `arc.exe` in a directory on `PATH`, then run `arc version`.

## Build from source

Building from source requires Go 1.26 or newer:

```bash
git clone https://github.com/EnzoCaetano015/Archbase.git
cd Archbase
go build -trimpath -o arc ./cmd/arc
```

Source builds report `arc dev` unless a version is injected with linker flags.

## Code signing status

Free code signing provided by SignPath.io, certificate by SignPath Foundation.

Archbase is preparing its Windows release process for SignPath Foundation approval. Version `v0.3.0` and all earlier releases are unsigned. They remain protected by the published SHA-256 checksums but do not contain an Authenticode publisher signature. No release will be described as signed until the SignPath integration is approved, activated, and verified.

See the [code signing policy](code-signing-policy.md), [privacy policy](privacy.md), and [MIT license](../LICENSE).
