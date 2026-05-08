param(
  [string]$RepoRoot = "",
  [string]$CliPath = "",
  [string]$Output = "../../.icoo/packages/issueye/agent/pkg/config.icpkg"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$packageRoot = (Resolve-Path $PSScriptRoot).Path

function Resolve-RepoRoot {
  param(
    [string]$InputRoot,
    [string]$StartPath
  )

  if ($InputRoot -and $InputRoot.Trim() -ne "") {
    return (Resolve-Path $InputRoot).Path
  }

  $current = (Resolve-Path $StartPath).Path
  while ($true) {
    if (Test-Path (Join-Path $current "icoo\go.mod")) {
      return $current
    }
    $parent = Split-Path $current -Parent
    if (-not $parent -or $parent -eq $current) {
      break
    }
    $current = $parent
  }
  throw "unable to locate repo root containing icoo\go.mod"
}

$root = Resolve-RepoRoot -InputRoot $RepoRoot -StartPath $packageRoot
$moduleRoot = Join-Path $root "icoo"

if (-not $CliPath -or $CliPath.Trim() -eq "") {
  $CliPath = Join-Path $moduleRoot "dist\icoo.exe"
} elseif (-not [System.IO.Path]::IsPathRooted($CliPath)) {
  $CliPath = [System.IO.Path]::GetFullPath((Join-Path $root $CliPath))
}

if (-not (Test-Path $CliPath)) {
  Push-Location $moduleRoot
  try {
    go build -o dist/icoo.exe ./cmd/icoo
  } finally {
    Pop-Location
  }
}

if (-not [System.IO.Path]::IsPathRooted($Output)) {
  $Output = [System.IO.Path]::GetFullPath((Join-Path $packageRoot $Output))
}
$outputDir = Split-Path $Output -Parent
if ($outputDir) {
  New-Item -ItemType Directory -Force $outputDir | Out-Null
}

& $CliPath package $packageRoot $Output
if ($LASTEXITCODE -ne 0) {
  throw "package build failed"
}

Get-Item $Output | Select-Object FullName, Length, LastWriteTime | Format-Table -AutoSize
