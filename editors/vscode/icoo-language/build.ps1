param(
  [switch]$Validate
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$scriptPath = Join-Path $root "scripts\\build.js"
$args = @($scriptPath)

if ($Validate) {
  $args += "--validate"
}

& node @args
if ($LASTEXITCODE -ne 0) {
  exit $LASTEXITCODE
}
