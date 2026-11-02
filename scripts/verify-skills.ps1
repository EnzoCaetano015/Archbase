param(
    [string] $SkillRoot,
    [string] $Manifest,
    [string] $Lock,
    [string] $CatalogRoot,
    [Parameter(Mandatory = $true)] [string] $Output
)
$ErrorActionPreference = 'Stop'
$ossRoot = Split-Path $PSScriptRoot -Parent
if (!$SkillRoot) { $SkillRoot = Join-Path $ossRoot 'skills/archbase-architecture-starter' }
if (!$Manifest) { $Manifest = Join-Path $SkillRoot 'bundle.json' }
if (!$Lock) { $Lock = Join-Path $ossRoot 'skills/catalog.lock.json' }
if (!$CatalogRoot) { $CatalogRoot = $ossRoot }
$Output = [IO.Path]::GetFullPath($Output)
if (Test-Path -LiteralPath $Output) { throw "Output already exists: $Output" }

function Invoke-Bundle([string[]] $BundleArgs) {
    & go run ./internal/skillbundle/cmd/skillbundle @BundleArgs
    if ($LASTEXITCODE -ne 0) { throw "skillbundle failed: $($BundleArgs[0])" }
}

Push-Location $ossRoot
try {
    & go test ./internal/skillbundle/...
    if ($LASTEXITCODE -ne 0) { throw 'skill tooling tests failed' }
    $common = @('-root', $SkillRoot, '-manifest', $Manifest, '-lock', $Lock, '-catalog-root', $CatalogRoot)
    Invoke-Bundle (@('validate') + $common)
    New-Item -ItemType Directory -Path $Output | Out-Null
    Invoke-Bundle (@('package') + $common + @('-output', (Join-Path $Output 'first')))
    Invoke-Bundle (@('package') + $common + @('-output', (Join-Path $Output 'second')))
    $first = @(Get-ChildItem -LiteralPath (Join-Path $Output 'first') -Filter '*.zip')
    $second = @(Get-ChildItem -LiteralPath (Join-Path $Output 'second') -Filter '*.zip')
    if ($first.Count -ne 1 -or $second.Count -ne 1) { throw 'Expected one skill ZIP per build' }
    $hash = (Get-FileHash -LiteralPath $first[0].FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($hash -ne (Get-FileHash -LiteralPath $second[0].FullName -Algorithm SHA256).Hash.ToLowerInvariant()) {
        throw 'Skill ZIPs are not byte-for-byte reproducible'
    }
    $checksum = Get-Content -LiteralPath (Join-Path $Output 'first/SHA256SUMS') -Raw
    if ($checksum.Trim() -ne "$hash  $($first[0].Name)") { throw 'Incorrect checksum manifest' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [IO.Compression.ZipFile]::OpenRead($first[0].FullName)
    try {
        foreach ($entry in $archive.Entries) {
            $expected = Join-Path $Output ('first/unpacked/' + $entry.FullName)
            $stream = $entry.Open()
            try { $memory = [IO.MemoryStream]::new(); $stream.CopyTo($memory); $actual = $memory.ToArray() }
            finally { $stream.Dispose(); if ($memory) { $memory.Dispose() } }
            $bytes = [IO.File]::ReadAllBytes($expected)
            if ([Convert]::ToBase64String($actual) -ne [Convert]::ToBase64String($bytes)) {
                throw "ZIP extraction mismatch: $($entry.FullName)"
            }
        }
    } finally { $archive.Dispose() }
    Write-Output "Verified skill ZIP: $hash  $($first[0].Name)"
} finally { Pop-Location }
