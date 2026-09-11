$ErrorActionPreference = "Stop"

$projectDir = Split-Path -Parent $PSScriptRoot
$clientConfig = Join-Path $projectDir ".runtime\mysql\root-client.ini"
$migrationPath = Join-Path $projectDir "migrations\001_initial_schema.sql"
$mysqlPath = "D:\cppsoft\mysql\bin\mysql.exe"

foreach ($requiredPath in @($clientConfig, $migrationPath, $mysqlPath)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required file was not found: $requiredPath"
    }
}

Get-Content -Raw -LiteralPath $migrationPath |
    & $mysqlPath "--defaults-extra-file=$clientConfig"

if ($LASTEXITCODE -ne 0) {
    throw "Content database migration failed with exit code $LASTEXITCODE"
}

Write-Output "Content database migration completed."
