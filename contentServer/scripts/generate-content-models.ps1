param([string]$DataSource = $env:CONTENT_DB_DSN)
$ErrorActionPreference = "Stop"
$projectDir = Split-Path -Parent $PSScriptRoot
if (-not $DataSource) {
    $config = Get-Content -Raw -LiteralPath (Join-Path $projectDir "etc\content-api.yaml")
    $match = [regex]::Match($config, '(?m)^DataSource:\s*"([^"]+)"')
    if (-not $match.Success) { throw "Set CONTENT_DB_DSN before generating models" }
    $DataSource = $match.Groups[1].Value
}
# goctl only inspects metadata. Its DSN handling cannot consume the app's
# encoded location option; do not pass application query options to it.
$modelDataSource = $DataSource.Split('?')[0]
Push-Location $projectDir
try {
    go tool goctl model mysql datasource --url $modelDataSource --table "post,post_draft,post_revision,post_card_projection,media_asset" --dir internal/model --style go_zero
    if ($LASTEXITCODE -ne 0) { throw "Model generation failed" }
} finally {
    Pop-Location
}
