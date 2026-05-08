param(
  [string]$RepoRoot = "",
  [string]$CliOutput = "dist/icoo.exe",
  [string]$PackageVersion = "0.1.2",
  [string]$OutputDir = "dist/icoo-agent",
  [string]$ExecutableName = "icoo-agent.exe",
  [switch]$RunTests,
  [switch]$SkipVerify
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$projectRoot = (Resolve-Path $PSScriptRoot).Path

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

function Resolve-RelativePath {
  param(
    [string]$Base,
    [string]$Child
  )

  return [System.IO.Path]::GetFullPath((Join-Path $Base $Child))
}

$root = Resolve-RepoRoot -InputRoot $RepoRoot -StartPath $projectRoot
$moduleRoot = Join-Path $root "icoo"
$packagerScript = Join-Path $moduleRoot "scripts\package-agent.ps1"

if (-not (Test-Path $packagerScript)) {
  throw "packager script not found: $packagerScript"
}

if (-not [System.IO.Path]::IsPathRooted($OutputDir)) {
  $OutputDir = Resolve-RelativePath -Base $projectRoot -Child $OutputDir
}
New-Item -ItemType Directory -Force $OutputDir | Out-Null

$releaseExe = Join-Path $OutputDir $ExecutableName
$moduleRelativeExe = [System.IO.Path]::GetRelativePath($moduleRoot, $releaseExe)

Write-Host "==> Building distributable agent into $OutputDir"

$invokeArgs = @{
  RepoRoot = $root
  CliOutput = $CliOutput
  AgentTarget = "apps/agent"
  AgentOutput = $moduleRelativeExe
  PackageVersion = $PackageVersion
}
if ($RunTests) {
  $invokeArgs.RunTests = $true
}
if ($SkipVerify) {
  $invokeArgs.SkipVerify = $true
}

& $packagerScript @invokeArgs
if ($LASTEXITCODE -ne 0) {
  throw "agent package script failed"
}

$configSource = Join-Path $projectRoot "config.toml"
if (Test-Path $configSource) {
  Copy-Item -LiteralPath $configSource -Destination (Join-Path $OutputDir "config.toml") -Force
}

$manifest = @(
  "icoo-agent distributable",
  "build_date=$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss'))",
  "package_version=$PackageVersion",
  "executable=$ExecutableName",
  "packages_dir=.icoo/packages/issueye"
) -join [Environment]::NewLine
Set-Content -LiteralPath (Join-Path $OutputDir "BUILD_INFO.txt") -Value $manifest -Encoding utf8

Get-Item $releaseExe, (Join-Path $OutputDir "config.toml"), (Join-Path $OutputDir "BUILD_INFO.txt") -ErrorAction SilentlyContinue |
  Select-Object FullName, Length, LastWriteTime |
  Format-Table -AutoSize
