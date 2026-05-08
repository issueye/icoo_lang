param(
  [string]$RepoRoot = "",
  [string]$CliOutput = "dist/icoo.exe",
  [switch]$RunTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-RepoRoot {
  param([string]$InputRoot)

  if ($InputRoot -and $InputRoot.Trim() -ne "") {
    return (Resolve-Path $InputRoot).Path
  }

  return (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
}

$root = Resolve-RepoRoot -InputRoot $RepoRoot
$moduleRoot = Join-Path $root "icoo"
if (-not (Test-Path (Join-Path $moduleRoot "go.mod"))) {
  throw "Go module root not found: $moduleRoot"
}
Set-Location $moduleRoot

if ($RunTests) {
  Write-Host "==> Running tests"
  go test ./...
}

# Always rebuild the host CLI first. The agent packager depends on the latest
# build logic, and using a stale icoo.exe will silently package old behavior.
Write-Host "==> Building host CLI: $CliOutput"
go build -o $CliOutput ./cmd/icoo

# upx
Write-Host "==> Compressing $CliOutput"
upx $CliOutput

Write-Host "==> Done"
Get-Item $CliOutput | Select-Object FullName, Length, LastWriteTime | Format-Table -AutoSize
