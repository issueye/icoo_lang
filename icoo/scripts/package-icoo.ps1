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

function Resolve-ModuleRoot {
  param([string]$Root)

  if (Test-Path (Join-Path $Root "go.mod")) {
    return $Root
  }

  $candidate = Join-Path $Root "icoo"
  if (Test-Path (Join-Path $candidate "go.mod")) {
    return $candidate
  }

  throw "Go module root not found from: $Root"
}

$root = Resolve-RepoRoot -InputRoot $RepoRoot
$moduleRoot = Resolve-ModuleRoot -Root $root
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
