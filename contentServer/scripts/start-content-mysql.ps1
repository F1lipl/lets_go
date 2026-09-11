$ErrorActionPreference = "Stop"

$projectDir = Split-Path -Parent $PSScriptRoot
$configPath = Join-Path $projectDir "etc\mysql-content.ini"
$mysqlPath = "D:\cppsoft\mysql\bin\mysqld.exe"

if (Get-NetTCPConnection -LocalPort 3309 -State Listen -ErrorAction SilentlyContinue) {
    Write-Output "Content MySQL is already listening on 127.0.0.1:3309."
    exit 0
}

if (-not (Test-Path -LiteralPath $mysqlPath)) {
    throw "mysqld.exe was not found at $mysqlPath"
}

$arguments = "--defaults-file=`"$configPath`" --console"
$process = Start-Process -FilePath $mysqlPath -ArgumentList $arguments -WindowStyle Hidden -PassThru
Start-Sleep -Seconds 2

$listener = Get-NetTCPConnection -LocalPort 3309 -State Listen -ErrorAction SilentlyContinue
if (-not $listener) {
    throw "Content MySQL did not start. Process id: $($process.Id)"
}

Write-Output "Content MySQL started on 127.0.0.1:3309. Process id: $($listener.OwningProcess)"
