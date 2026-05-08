param(
  [string]$RepoRoot = "",
  [string]$CliOutput = "dist/icoo.exe",
  [string]$AgentTarget = "apps/agent",
  [string]$AgentOutput = "dist/icoo-agent.exe",
  [string]$PackageVersion = "0.1.2",
  [string]$PackageRoot = "",
  [switch]$SkipExecutable,
  [switch]$SkipVerify,
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

function Resolve-RelativePath {
  param(
    [string]$Base,
    [string]$Child
  )

  return [System.IO.Path]::GetFullPath((Join-Path $Base $Child))
}

function Invoke-Icoo {
  param(
    [string]$CliPath,
    [string[]]$Arguments
  )

  & $CliPath @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "icoo command failed: $CliPath $($Arguments -join ' ')"
  }
}

function New-AgentVerifyWorkspace {
  param(
    [string]$AgentWorkspace
  )

  $verifyRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("icoo-agent-verify-" + [System.Guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $verifyRoot -Force | Out-Null
  $runtimeRoot = Join-Path $verifyRoot "runtime"
  New-Item -ItemType Directory -Path $runtimeRoot -Force | Out-Null

  $configToml = @"
workspace = "./runtime"
session_dir = ".agent/sessions"
log_path = ".agent/agent.log"
"@
  Set-Content -Path (Join-Path $verifyRoot "config.toml") -Value $configToml -Encoding UTF8

  Set-Content -Path (Join-Path $runtimeRoot "config.toml") -Value "# verify workspace`n" -Encoding UTF8

  return $verifyRoot
}

if ($RunTests) {
  Write-Host "==> Running tests"
  go test ./...
}

# Always rebuild the host CLI first. The agent packager depends on the latest
# build logic, and using a stale icoo.exe will silently package old behavior.
Write-Host "==> Building host CLI: $CliOutput"
go build -o $CliOutput ./cmd/icoo

$cliPath = Resolve-RelativePath -Base $moduleRoot -Child $CliOutput
$agentPath = Resolve-RelativePath -Base $root -Child $AgentTarget
$packageDir = $PackageRoot
if (-not $packageDir -or $packageDir.Trim() -eq "") {
  $packageDir = Join-Path $agentPath ".icoo\packages\issueye"
} elseif (-not [System.IO.Path]::IsPathRooted($packageDir)) {
  $packageDir = Resolve-RelativePath -Base $root -Child $PackageRoot
}

$configPackagePath = Join-Path $packageDir "agent\pkg\config.icpkg"
$sessionPackagePath = Join-Path $packageDir "agent\pkg\session.icpkg"
$toolsPackagePath = Join-Path $packageDir "agent\pkg\tools.icpkg"
$agentPackagePath = Join-Path $packageDir "agent.icpkg"
$smokePath = Join-Path $agentPath "smoke_package.ic"

Write-Host "==> Packaging reusable modules into $packageDir"
Invoke-Icoo -CliPath $cliPath -Arguments @(
  "package",
  (Join-Path $agentPath "pkg\config"),
  $configPackagePath,
  "--version", $PackageVersion
)
Invoke-Icoo -CliPath $cliPath -Arguments @(
  "package",
  (Join-Path $agentPath "pkg\session"),
  $sessionPackagePath,
  "--version", $PackageVersion
)
Invoke-Icoo -CliPath $cliPath -Arguments @(
  "package",
  (Join-Path $agentPath "pkg\tools"),
  $toolsPackagePath,
  "--version", $PackageVersion
)
Invoke-Icoo -CliPath $cliPath -Arguments @(
  "package",
  $agentPath,
  $agentPackagePath,
  "--name", "issueye/agent",
  "--version", $PackageVersion,
  "--export", "src/app/app.ic"
)

if (-not $SkipExecutable) {
  Write-Host "==> Building standalone agent executable: $AgentTarget -> $AgentOutput"
  Invoke-Icoo -CliPath $cliPath -Arguments @("build", $agentPath, $AgentOutput)
}

if (-not $SkipVerify) {
  $verifyWorkspace = New-AgentVerifyWorkspace -AgentWorkspace $agentPath
  try {
    Push-Location $verifyWorkspace

    Write-Host "==> Verifying source project"
    Invoke-Icoo -CliPath $cliPath -Arguments @("run", $agentPath)

    Write-Host "==> Verifying packaged agent archive"
    Invoke-Icoo -CliPath $cliPath -Arguments @("run", $agentPackagePath)

    if (Test-Path $smokePath) {
      Write-Host "==> Verifying pkg: import smoke test"
      Invoke-Icoo -CliPath $cliPath -Arguments @("run", $smokePath)
    }
  } finally {
    Pop-Location
    Remove-Item -LiteralPath $verifyWorkspace -Recurse -Force -ErrorAction SilentlyContinue
  }
}

Write-Host "==> Done"
$outputs = @($cliPath, $configPackagePath, $sessionPackagePath, $toolsPackagePath, $agentPackagePath)
if (-not $SkipExecutable) {
  $outputs += (Resolve-RelativePath -Base $moduleRoot -Child $AgentOutput)
}
Get-Item $outputs | Select-Object FullName, Length, LastWriteTime | Format-Table -AutoSize
