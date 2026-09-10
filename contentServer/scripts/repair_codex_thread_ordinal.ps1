[CmdletBinding()]
param(
    [switch]$CheckOnly
)

$ErrorActionPreference = 'Stop'

$targetPath = 'C:\Users\29953\.codex\sessions\2026\09\08\rollout-2026-09-08T13-52-52-01a07eae-7dbe-78e0-9f0f-9b110e2c7bb9_01a07f93-91b2-7961-9dfc-47aac8527dc4.jsonl'
$expectedSessionsRoot = [System.IO.Path]::GetFullPath(
    (Join-Path $env:USERPROFILE '.codex\sessions')
).TrimEnd('\') + '\'
$resolvedTarget = [System.IO.Path]::GetFullPath($targetPath)

if (-not $resolvedTarget.StartsWith($expectedSessionsRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Target is outside the expected sessions directory: $resolvedTarget"
}

if (-not (Test-Path -LiteralPath $resolvedTarget -PathType Leaf)) {
    throw "Session file was not found: $resolvedTarget"
}

if (-not $CheckOnly) {
    $activeProcesses = @(
        Get-Process -Name 'ChatGPT', 'codex', 'codex-code-mode-host' -ErrorAction SilentlyContinue
    )
    if ($activeProcesses.Count -gt 0) {
        $names = ($activeProcesses | Select-Object -ExpandProperty ProcessName -Unique) -join ', '
        throw "Exit the ChatGPT desktop app completely before running the repair. Active processes: $names"
    }
}

$utf8NoBom = [System.Text.UTF8Encoding]::new($false)
function Read-SharedUtf8Text {
    param([Parameter(Mandatory)][string]$Path)

    $stream = [System.IO.FileStream]::new(
        $Path,
        [System.IO.FileMode]::Open,
        [System.IO.FileAccess]::Read,
        [System.IO.FileShare]::ReadWrite
    )
    try {
        $reader = [System.IO.StreamReader]::new($stream, $utf8NoBom, $true)
        try {
            return $reader.ReadToEnd()
        }
        finally {
            $reader.Dispose()
        }
    }
    finally {
        $stream.Dispose()
    }
}

$originalText = Read-SharedUtf8Text -Path $resolvedTarget
$ordinalPattern = [regex]'(?m)^(?<prefix>\{"timestamp":"[^"]+","ordinal":)(?<ordinal>\d+)'
$matches = $ordinalPattern.Matches($originalText)

if ($matches.Count -eq 0) {
    throw 'No top-level ordinal fields were found.'
}

$ordinals = @(
    foreach ($match in $matches) {
        [int64]$match.Groups['ordinal'].Value
    }
)

$duplicateGroups = @($ordinals | Group-Object | Where-Object Count -gt 1)
if ($duplicateGroups.Count -ne 1 -or $duplicateGroups[0].Name -ne '264' -or $duplicateGroups[0].Count -ne 2) {
    $description = ($duplicateGroups | ForEach-Object { "$($_.Name)x$($_.Count)" }) -join ', '
    throw "The file no longer has the expected single duplicate ordinal 264. Duplicates: $description"
}

$duplicateIndexes = @(
    for ($index = 0; $index -lt $ordinals.Count; $index++) {
        if ($ordinals[$index] -eq 264) {
            $index
        }
    }
)
$repairFromIndex = $duplicateIndexes[1]

$repairedOrdinals = [System.Collections.Generic.List[long]]::new()
for ($index = 0; $index -lt $ordinals.Count; $index++) {
    $value = $ordinals[$index]
    if ($index -ge $repairFromIndex) {
        $value++
    }
    $repairedOrdinals.Add($value)
}

for ($index = 1; $index -lt $repairedOrdinals.Count; $index++) {
    if ($repairedOrdinals[$index] -ne ($repairedOrdinals[$index - 1] + 1)) {
        throw "Validation failed between ordinals $($repairedOrdinals[$index - 1]) and $($repairedOrdinals[$index])."
    }
}

Write-Output "Session file: $resolvedTarget"
Write-Output "Records: $($ordinals.Count)"
Write-Output "Duplicate: ordinal 264 at record indexes $($duplicateIndexes -join ', ')"
Write-Output "Repair: increment ordinals from the second 264 through the final record"
Write-Output "Resulting range: $($repairedOrdinals[0])..$($repairedOrdinals[$repairedOrdinals.Count - 1])"

if ($CheckOnly) {
    Write-Output 'Check completed; no file was changed.'
    exit 0
}

$builder = [System.Text.StringBuilder]::new($originalText.Length + 32)
$cursor = 0
for ($index = 0; $index -lt $matches.Count; $index++) {
    $ordinalGroup = $matches[$index].Groups['ordinal']
    [void]$builder.Append($originalText, $cursor, $ordinalGroup.Index - $cursor)
    [void]$builder.Append($repairedOrdinals[$index])
    $cursor = $ordinalGroup.Index + $ordinalGroup.Length
}
[void]$builder.Append($originalText, $cursor, $originalText.Length - $cursor)
$updatedText = $builder.ToString()

$timestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$backupPath = "$resolvedTarget.backup-$timestamp"
$temporaryPath = "$resolvedTarget.repairing-$timestamp"

[System.IO.File]::WriteAllText($temporaryPath, $updatedText, $utf8NoBom)

$writtenText = [System.IO.File]::ReadAllText($temporaryPath, $utf8NoBom)
$writtenOrdinals = @(
    foreach ($match in $ordinalPattern.Matches($writtenText)) {
        [int64]$match.Groups['ordinal'].Value
    }
)
if ($writtenOrdinals.Count -ne $repairedOrdinals.Count) {
    Remove-Item -LiteralPath $temporaryPath
    throw 'Temporary file validation failed: record count changed.'
}
for ($index = 0; $index -lt $writtenOrdinals.Count; $index++) {
    if ($writtenOrdinals[$index] -ne $repairedOrdinals[$index]) {
        Remove-Item -LiteralPath $temporaryPath
        throw "Temporary file validation failed at record index $index."
    }
}

Copy-Item -LiteralPath $resolvedTarget -Destination $backupPath
try {
    Copy-Item -LiteralPath $temporaryPath -Destination $resolvedTarget -Force
    Remove-Item -LiteralPath $temporaryPath
}
catch {
    Copy-Item -LiteralPath $backupPath -Destination $resolvedTarget -Force
    if (Test-Path -LiteralPath $temporaryPath) {
        Remove-Item -LiteralPath $temporaryPath
    }
    throw
}

Write-Output "Repair completed. Backup: $backupPath"
Write-Output 'Start the ChatGPT desktop app and reopen the task.'
