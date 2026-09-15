param(
    [string]$ClientConfig,
    [string]$MysqlPath = "D:\cppsoft\mysql\bin\mysql.exe",
    [string]$DumpPath = "D:\cppsoft\mysql\bin\mysqldump.exe"
)
$ErrorActionPreference = "Stop"
$projectDir = Split-Path -Parent $PSScriptRoot
if (-not $ClientConfig) {
    $ClientConfig = Join-Path $projectDir ".runtime\mysql\root-client.ini"
}
foreach ($requiredPath in @($ClientConfig, $MysqlPath, $DumpPath)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required file was not found: $requiredPath"
    }
}

function Invoke-ContentSql([string]$Sql) {
    $result = & $MysqlPath "--defaults-extra-file=$ClientConfig" "--default-character-set=utf8mb4" --batch --skip-column-names --execute=$Sql
    if ($LASTEXITCODE -ne 0) {
        throw "MySQL command failed with exit code $LASTEXITCODE"
    }
    return $result
}

# Run only one migration process at a time, with application writes stopped.
# Existing untracked installations can adopt 001/002 because those files are
# repeatable against the legacy schema.
$bootstrap = @'
CREATE DATABASE IF NOT EXISTS content_server CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS content_server.schema_migration (
    migration_name VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
    checksum CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB;
'@
Invoke-ContentSql $bootstrap | Out-Null

$migrations = Get-ChildItem -LiteralPath (Join-Path $projectDir "migrations") -Filter "*.sql" | Sort-Object Name
$backedUp = $false
foreach ($migration in $migrations) {
    if ($migration.Name -notmatch '^\d{3}_[a-z0-9_]+\.sql$') {
        throw "Unexpected migration name: $($migration.Name)"
    }
    $sql = (Get-Content -Raw -LiteralPath $migration.FullName).Replace("`r`n", "`n")
    $hasher = [System.Security.Cryptography.SHA256]::Create()
    try {
        $digest = $hasher.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($sql))
        $checksum = ([BitConverter]::ToString($digest)).Replace("-", "").ToLowerInvariant()
    } finally {
        $hasher.Dispose()
    }
    $applied = Invoke-ContentSql "SELECT checksum FROM content_server.schema_migration WHERE migration_name='$($migration.Name)'"
    if ($applied) {
        if ($applied -ne $checksum) {
            throw "Applied migration has changed: $($migration.Name)"
        }
        Write-Output "Already applied: $($migration.Name)"
        continue
    }
    if (-not $backedUp) {
        $backupDir = Join-Path $projectDir ".runtime\mysql\backups"
        New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
        $backupPath = Join-Path $backupDir ("content_server-before-migrate-" + (Get-Date -Format "yyyyMMdd-HHmmss-fff") + ".sql")
        & $DumpPath "--defaults-extra-file=$ClientConfig" --default-character-set=utf8mb4 --single-transaction --routines --triggers --events --set-gtid-purged=OFF "--result-file=$backupPath" --databases content_server
        if ($LASTEXITCODE -ne 0) {
            throw "Backup failed; migration stopped"
        }
        Write-Output "Backup: $backupPath"
        $backedUp = $true
    }

    $previousEncoding = $OutputEncoding
    try {
        $OutputEncoding = [System.Text.UTF8Encoding]::new($false)
        $sql | & $MysqlPath "--defaults-extra-file=$ClientConfig" --default-character-set=utf8mb4
        if ($LASTEXITCODE -ne 0) {
            throw "Migration failed: $($migration.Name). Resolve the cause and rerun."
        }
    } finally {
        $OutputEncoding = $previousEncoding
    }
    Invoke-ContentSql "INSERT INTO content_server.schema_migration (migration_name, checksum) VALUES ('$($migration.Name)', '$checksum')" | Out-Null
    Write-Output "Applied: $($migration.Name)"
}
Write-Output "Content database migration completed."
