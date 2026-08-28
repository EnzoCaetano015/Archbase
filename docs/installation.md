# Installing Archbase

Archbase publishes one self-contained `arc` executable for each supported operating system and architecture. Version `v0.1.0` is the first binary release.

| Operating system | Architecture | Asset |
| --- | --- | --- |
| Linux | x86-64 | `arc_v0.1.0_linux_amd64.tar.gz` |
| Linux | ARM64 | `arc_v0.1.0_linux_arm64.tar.gz` |
| macOS | Intel | `arc_v0.1.0_darwin_amd64.tar.gz` |
| macOS | Apple silicon | `arc_v0.1.0_darwin_arm64.tar.gz` |
| Windows | x86-64 | `arc_v0.1.0_windows_amd64.zip` |
| Windows | ARM64 | `arc_v0.1.0_windows_arm64.zip` |

All assets and `arc_v0.1.0_SHA256SUMS.txt` are published at the [v0.1.0 release](https://github.com/EnzoCaetano015/Archbase/releases/tag/v0.1.0). Verify the archive before extracting it.

## Linux

Replace `amd64` with `arm64` when appropriate.

```bash
version=0.1.0
asset="arc_v${version}_linux_amd64.tar.gz"
base="https://github.com/EnzoCaetano015/Archbase/releases/download/v${version}"
curl -fLO "$base/$asset"
curl -fLO "$base/arc_v${version}_SHA256SUMS.txt"
grep -F "  $asset" "arc_v${version}_SHA256SUMS.txt" | sha256sum --check -
tar -xzf "$asset"
install -d "$HOME/.local/bin"
test ! -e "$HOME/.local/bin/arc" || { echo "arc already exists in $HOME/.local/bin" >&2; exit 1; }
install -m 0755 arc "$HOME/.local/bin/arc"
```

Ensure `$HOME/.local/bin` is in `PATH`, open a new shell, and run `arc version`.

## macOS

Use `darwin_arm64` on Apple silicon and `darwin_amd64` on Intel Macs.

```bash
version=0.1.0
asset="arc_v${version}_darwin_arm64.tar.gz"
base="https://github.com/EnzoCaetano015/Archbase/releases/download/v${version}"
curl -fLO "$base/$asset"
curl -fLO "$base/arc_v${version}_SHA256SUMS.txt"
expected="$(grep -F "  $asset" "arc_v${version}_SHA256SUMS.txt" | cut -d ' ' -f 1)"
actual="$(shasum -a 256 "$asset" | cut -d ' ' -f 1)"
test "$actual" = "$expected"
tar -xzf "$asset"
mkdir -p "$HOME/.local/bin"
test ! -e "$HOME/.local/bin/arc" || { echo "arc already exists in $HOME/.local/bin" >&2; exit 1; }
install -m 0755 arc "$HOME/.local/bin/arc"
```

Ensure `$HOME/.local/bin` is in `PATH`, open a new shell, and run `arc version`.

## Windows PowerShell

The example uses x86-64. Replace `amd64` with `arm64` on Windows ARM devices.

```powershell
$version = "0.1.0"
$asset = "arc_v${version}_windows_amd64.zip"
$base = "https://github.com/EnzoCaetano015/Archbase/releases/download/v${version}"
Invoke-WebRequest "$base/$asset" -OutFile $asset
Invoke-WebRequest "$base/arc_v${version}_SHA256SUMS.txt" -OutFile "arc_v${version}_SHA256SUMS.txt"
$line = Get-Content "arc_v${version}_SHA256SUMS.txt" | Where-Object { $_ -match "  $([regex]::Escape($asset))$" }
$expected = ($line -split '\s+')[0].ToLowerInvariant()
$actual = (Get-FileHash $asset -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $expected) { throw "SHA-256 checksum mismatch for $asset" }
Expand-Archive $asset -DestinationPath .\archbase-v$version
$install = Join-Path $env:LOCALAPPDATA "Programs\Archbase"
New-Item -ItemType Directory -Force $install | Out-Null
$target = Join-Path $install "arc.exe"
if (Test-Path -LiteralPath $target) { throw "arc.exe already exists at $target" }
Copy-Item ".\archbase-v$version\arc.exe" $target
```

Add `%LOCALAPPDATA%\Programs\Archbase` to the user `PATH`, open a new terminal, and run:

```powershell
arc version
```

The expected output for these packages is `arc 0.1.0`.

## Build from source

Building from source requires Go 1.26 or newer:

```bash
git clone https://github.com/EnzoCaetano015/Archbase.git
cd Archbase
go build -trimpath -o arc ./cmd/arc
```

Source builds report `arc dev` unless a version is injected with linker flags.
