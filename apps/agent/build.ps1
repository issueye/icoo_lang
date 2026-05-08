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

function Get-RelativePathCompat {
  param(
    [string]$Base,
    [string]$Path
  )

  $resolvedBase = (Resolve-Path $Base).Path
  $resolvedPath = [System.IO.Path]::GetFullPath($Path)

  if ([type]::GetType("System.IO.Path") -and [System.IO.Path].GetMethod("GetRelativePath", [type[]]@([string], [string]))) {
    return [System.IO.Path]::GetRelativePath($resolvedBase, $resolvedPath)
  }

  $baseUri = New-Object System.Uri(($resolvedBase.TrimEnd('\') + '\'))
  $pathUri = New-Object System.Uri($resolvedPath)
  $relative = $baseUri.MakeRelativeUri($pathUri).ToString()
  return $relative.Replace('/', '\')
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
$moduleRelativeExe = Get-RelativePathCompat -Base $moduleRoot -Path $releaseExe

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

$runtimeConfigSource = Join-Path $projectRoot "runtime\\config.toml"
$runtimeOutputDir = Join-Path $OutputDir "runtime"
New-Item -ItemType Directory -Force $runtimeOutputDir | Out-Null
if (Test-Path $runtimeConfigSource) {
  Copy-Item -LiteralPath $runtimeConfigSource -Destination (Join-Path $runtimeOutputDir "config.toml") -Force
}

$manifest = @(
  "icoo-agent distributable",
  "build_date=$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss'))",
  "package_version=$PackageVersion",
  "executable=$ExecutableName",
  "packages_dir=.icoo/packages/issueye"
) -join [Environment]::NewLine
Set-Content -LiteralPath (Join-Path $OutputDir "BUILD_INFO.txt") -Value $manifest -Encoding utf8

Get-Item $releaseExe, (Join-Path $OutputDir "config.toml"), (Join-Path $runtimeOutputDir "config.toml"), (Join-Path $OutputDir "BUILD_INFO.txt") -ErrorAction SilentlyContinue |
  Select-Object FullName, Length, LastWriteTime |
  Format-Table -AutoSize
