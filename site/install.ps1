<#
.SYNOPSIS
Installs the Archbase CLI on Windows.

.PARAMETER Version
Stable version to install, with or without the leading v. Defaults to the latest release.

.PARAMETER InstallDir
Directory that receives arc.exe. Defaults to %LOCALAPPDATA%\Programs\Archbase.

.PARAMETER Force
Replace an existing arc.exe.

.PARAMETER Help
Show usage information.
#>
[CmdletBinding()]
param(
    [string]$Version,
    [string]$InstallDir,
    [switch]$Force,
    [Alias('h')]
    [switch]$Help
)

$ErrorActionPreference = 'Stop'
$repository = 'EnzoCaetano015/Archbase'
$releaseApiUrl = if ($env:ARCHBASE_RELEASE_API_URL) { $env:ARCHBASE_RELEASE_API_URL } else { "https://api.github.com/repos/$repository/releases/latest" }
$releaseBaseUrl = if ($env:ARCHBASE_RELEASE_BASE_URL) { $env:ARCHBASE_RELEASE_BASE_URL.TrimEnd('/') } else { "https://github.com/$repository/releases/download" }
$temporaryDir = $null
$stagedTarget = $null

function Fail([string]$Message) {
    throw "Archbase install error: $Message"
}

if ($Help) {
    @'
Install the Archbase CLI.

Usage:
  install.ps1 [-Version <vMAJOR.MINOR.PATCH>] [-InstallDir <path>] [-Force]

Options:
  -Version       Install a specific stable version. Defaults to the latest release.
  -InstallDir    Install arc.exe in this directory.
  -Force         Replace an existing arc.exe.
  -Help          Show this help.
'@
    return
}

try {
    if ($env:OS -ne 'Windows_NT' -and -not $env:ARCHBASE_TEST_OS) {
        Fail 'install.ps1 only supports Windows'
    }

    if ([string]::IsNullOrWhiteSpace($InstallDir)) {
        if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
            Fail 'LOCALAPPDATA is not available; pass -InstallDir explicitly'
        }
        $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\Archbase'
    }
    if ($InstallDir.Contains(';')) {
        Fail 'install directory must not contain a semicolon'
    }
    $InstallDir = [System.IO.Path]::GetFullPath($InstallDir)

    if ([string]::IsNullOrWhiteSpace($Version)) {
        try {
            $release = Invoke-RestMethod -Uri $releaseApiUrl -Headers @{
                Accept = 'application/vnd.github+json'
                'X-GitHub-Api-Version' = '2022-11-28'
                'User-Agent' = 'archbase-installer'
            }
            $releaseTag = [string]$release.tag_name
        }
        catch {
            Fail "could not resolve the latest Archbase release: $($_.Exception.Message)"
        }
    }
    else {
        $releaseTag = if ($Version.StartsWith('v')) { $Version } else { "v$Version" }
    }

    if ($releaseTag -notmatch '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') {
        Fail 'version must match vMAJOR.MINOR.PATCH'
    }
    $resolvedVersion = $releaseTag.Substring(1)

    $detectedArchitecture = if ($env:ARCHBASE_TEST_ARCH) {
        $env:ARCHBASE_TEST_ARCH
    }
    elseif ($env:PROCESSOR_ARCHITEW6432) {
        $env:PROCESSOR_ARCHITEW6432
    }
    else {
        $env:PROCESSOR_ARCHITECTURE
    }
    $targetArchitecture = switch ($detectedArchitecture.ToUpperInvariant()) {
        'AMD64' { 'amd64' }
        'X86_64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { Fail "unsupported architecture: $detectedArchitecture" }
    }

    $asset = "arc_v${resolvedVersion}_windows_${targetArchitecture}.zip"
    $checksumName = "arc_v${resolvedVersion}_SHA256SUMS.txt"
    $downloadUrl = "$releaseBaseUrl/$releaseTag"
    $target = Join-Path $InstallDir 'arc.exe'

    if (Test-Path -LiteralPath $target -PathType Container) {
        Fail "installation target is a directory: $target"
    }
    if ((Test-Path -LiteralPath $target) -and -not $Force) {
        Fail "arc.exe already exists at $target; rerun with -Force to replace it"
    }

    $temporaryDir = Join-Path ([System.IO.Path]::GetTempPath()) ("archbase-install-" + [guid]::NewGuid().ToString('N'))
    $extractDir = Join-Path $temporaryDir 'extract'
    $archivePath = Join-Path $temporaryDir $asset
    $checksumPath = Join-Path $temporaryDir $checksumName
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null

    Write-Host "Downloading Archbase $resolvedVersion for windows/$targetArchitecture..."
    try {
        Invoke-WebRequest -Uri "$downloadUrl/$asset" -OutFile $archivePath -UseBasicParsing
        Invoke-WebRequest -Uri "$downloadUrl/$checksumName" -OutFile $checksumPath -UseBasicParsing
    }
    catch {
        Fail "could not download release files: $($_.Exception.Message)"
    }

    $assetPattern = [regex]::Escape($asset)
    $checksumLine = Get-Content -LiteralPath $checksumPath | Where-Object { $_ -match "^(?<hash>[0-9A-Fa-f]{64})\s{2}$assetPattern$" }
    if (@($checksumLine).Count -ne 1) {
        Fail "checksum manifest does not contain exactly one valid hash for $asset"
    }
    [void]($checksumLine -match '^(?<hash>[0-9A-Fa-f]{64})')
    $expectedHash = $Matches.hash.ToLowerInvariant()
    $actualHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        Fail "SHA-256 checksum mismatch for $asset"
    }

    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir
    $downloadedBinary = Join-Path $extractDir 'arc.exe'
    if (-not (Test-Path -LiteralPath $downloadedBinary -PathType Leaf)) {
        Fail 'release archive does not contain arc.exe'
    }
    $reportedVersion = (& $downloadedBinary version 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $reportedVersion -ne "arc $resolvedVersion") {
        Fail "downloaded executable reported '$reportedVersion', expected 'arc $resolvedVersion'"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $stagedTarget = Join-Path $InstallDir (".arc." + [guid]::NewGuid().ToString('N') + '.tmp.exe')
    Copy-Item -LiteralPath $downloadedBinary -Destination $stagedTarget
    if (Test-Path -LiteralPath $target -PathType Container) {
        Fail "installation target became a directory during installation: $target"
    }
    if ((Test-Path -LiteralPath $target) -and -not $Force) {
        Fail "arc.exe appeared at $target during installation; rerun with -Force to replace it"
    }
    Move-Item -LiteralPath $stagedTarget -Destination $target -Force
    $stagedTarget = $null

    $testPathFile = $env:ARCHBASE_TEST_USER_PATH_FILE
    if ($testPathFile) {
        $userPath = if (Test-Path -LiteralPath $testPathFile) { Get-Content -LiteralPath $testPathFile -Raw } else { '' }
    }
    else {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    }
    $pathEntries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $normalizedInstallDir = $InstallDir.TrimEnd('\')
    $pathContainsInstallDir = $false
    foreach ($entry in $pathEntries) {
        if ($entry.Trim().TrimEnd('\').Equals($normalizedInstallDir, [StringComparison]::OrdinalIgnoreCase)) {
            $pathContainsInstallDir = $true
            break
        }
    }
    if (-not $pathContainsInstallDir) {
        $newUserPath = (@($pathEntries) + $InstallDir) -join ';'
        if ($testPathFile) {
            Set-Content -LiteralPath $testPathFile -Value $newUserPath -NoNewline
        }
        else {
            [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
        }
        Write-Host 'PATH was updated. Open a new terminal before running arc.'
    }
    if (-not (($env:Path -split ';') | Where-Object { $_.TrimEnd('\').Equals($normalizedInstallDir, [StringComparison]::OrdinalIgnoreCase) })) {
        $env:Path = "$InstallDir;$env:Path"
    }

    Write-Host "Archbase $resolvedVersion installed at $target"
}
finally {
    if ($stagedTarget -and (Test-Path -LiteralPath $stagedTarget)) {
        Remove-Item -LiteralPath $stagedTarget -Force -ErrorAction SilentlyContinue
    }
    if ($temporaryDir -and (Test-Path -LiteralPath $temporaryDir)) {
        Remove-Item -LiteralPath $temporaryDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
