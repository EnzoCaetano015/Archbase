# Planning-contract checks only. This does not implement audit or doctor.
$ErrorActionPreference = 'Stop'
function Confirm-Contract([string] $Kind, [hashtable] $Document) {
    $json = $Document | ConvertTo-Json -Depth 40
    $schema = Join-Path $PSScriptRoot "schemas/$Kind-result.schema.json"
    if (!(Test-Json -Json $json -SchemaFile $schema -ErrorAction SilentlyContinue)) { return $false }
    if ($Kind -eq 'audit') {
        foreach ($finding in $Document.findings) {
            if ($finding.kind -eq 'architecture' -and $finding.ruleId -cnotin @($Document.baseline.ruleId)) { return $false }
            foreach ($evidence in $finding.evidence) {
                if ($evidence.endLine -lt $evidence.startLine) { return $false }
            }
        }
        if (@($Document.baseline.ruleId | Select-Object -Unique).Count -ne $Document.baseline.Count) { return $false }
        if ($Document.state -eq 'complete' -and ($Document.unassessed.Count -or $Document.errors.Count)) { return $false }
        if ($Document.state -eq 'not-applicable' -and $Document.findings.Count) { return $false }
        if ($Document.state -eq 'partial' -and !$Document.unassessed.Count -and !$Document.errors.Count) { return $false }
    } else {
        if (@($Document.checks.id | Select-Object -Unique).Count -ne $Document.checks.Count) { return $false }
        $blocked = @($Document.checks | Where-Object status -eq 'not-run').Count
        if ($Document.state -eq 'complete' -and $blocked) { return $false }
        if ($Document.state -eq 'partial' -and !$blocked) { return $false }
    }
    return $true
}
$examples = @{}
foreach ($kind in @('audit', 'doctor')) {
    $examples[$kind] = Get-Content (Join-Path $PSScriptRoot "examples/$kind.json") -Raw | ConvertFrom-Json -AsHashtable
    if (!(Confirm-Contract $kind $examples[$kind])) { throw "Invalid proposed $kind example" }
}
$unconfigured = $examples.doctor | ConvertTo-Json -Depth 40 | ConvertFrom-Json -AsHashtable
$unconfigured.state = 'complete'
$unconfigured.checks[1].status = 'warning'
$unconfigured.checks[1].message = 'Workspace not configured'
$unconfigured.checks[1].reason = 'No root index exists'
$unconfigured.checks[2].status = 'not-applicable'
$unconfigured.checks[2].reason = 'No configured bundles to inspect'
if (!(Confirm-Contract doctor $unconfigured)) { throw 'Valid unconfigured workspace example was rejected' }
$rejections = 0
function Must-Reject([string] $Kind, [string] $Name, [scriptblock] $Mutate) {
    $candidate = $examples[$Kind] | ConvertTo-Json -Depth 40 | ConvertFrom-Json -AsHashtable
    & $Mutate $candidate
    if (Confirm-Contract $Kind $candidate) { throw "Accepted invalid planning contract: $Name" }
    $script:rejections++
}
Must-Reject audit 'unknown schema version' { param($d) $d.schemaVersion = 2 }
Must-Reject audit 'absolute evidence path' { param($d) $d.findings[0].evidence[0].path = 'C:/outside/example.ts' }
Must-Reject audit 'parent traversal' { param($d) $d.findings[0].evidence[0].path = 'features/../outside.ts' }
Must-Reject audit 'backslash path' { param($d) $d.findings[0].evidence[0].path = 'features\outside.ts' }
Must-Reject audit 'alternate stream path' { param($d) $d.findings[0].evidence[0].path = 'features/example.ts:stream' }
Must-Reject audit 'noncanonical path' { param($d) $d.findings[0].evidence[0].path = 'features//example.ts' }
Must-Reject audit 'architecture finding without RuleID' { param($d) $d.findings[0].ruleId = $null }
Must-Reject audit 'unadopted RuleID' { param($d) $d.baseline = @() }
Must-Reject audit 'missing hash' { param($d) $d.findings[0].evidence[0].Remove('sha256') | Out-Null }
Must-Reject audit 'reversed lines' { param($d) $d.findings[0].evidence[0].endLine = 1; $d.findings[0].evidence[0].startLine = 3 }
Must-Reject audit 'unknown numeric score' { param($d) $d.score = 100 }
Must-Reject audit 'unassessed called complete' { param($d) $d.state = 'complete' }
Must-Reject doctor 'missing blocked reason' { param($d) $d.checks[2].Remove('reason') | Out-Null }
Must-Reject doctor 'duplicate check ID' { param($d) $d.checks[1].id = $d.checks[0].id }
Must-Reject doctor 'blocked called complete' { param($d) $d.state = 'complete' }
Must-Reject doctor 'absolute root leak' { param($d) $d.projectRoot = 'C:/customer/project' }
Must-Reject doctor 'missing not-applicable reason' { param($d) $d.checks[2].status = 'not-applicable'; $d.checks[2].Remove('reason') | Out-Null }
Write-Output "Planning contracts passed: 3 valid examples; $rejections rejected mutations. No runtime scanner tested."
