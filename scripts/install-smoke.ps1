# The Windows half of install-smoke.sh. Builds jstack.exe from this checkout,
# zips it the way goreleaser does, then runs install.ps1 against that zip
# twice, refuses a bad checksum, and runs setup for real into a throwaway
# profile folder so the PowerShell lines in tools.md are exercised.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$version = "0.0.0-smoke"
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "unsupported smoke architecture: $env:PROCESSOR_ARCHITECTURE" }
}
$work = Join-Path ([System.IO.Path]::GetTempPath()) "jstack-install-smoke-$PID"
$savedProfile = $env:USERPROFILE

function Assert([bool]$Condition, [string]$Message) {
  if (-not $Condition) {
    throw "install smoke: $Message"
  }
}

function Invoke-Installer([string]$Dist, [string]$Bin) {
  $env:JSTACK_INSTALL_ARCHIVE = Join-Path $Dist $archiveName
  $env:JSTACK_INSTALL_CHECKSUMS = Join-Path $Dist "checksums.txt"
  $env:JSTACK_INSTALL_VERSION = $version
  $env:JSTACK_INSTALL_DIR = $Bin
  $env:JSTACK_INSTALL_SKIP_PATH = "1"
  & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root "install.ps1")
  return $LASTEXITCODE
}

try {
  New-Item -ItemType Directory -Force $work | Out-Null
  $dist = Join-Path $work "dist"
  $build = Join-Path $work "build"
  $profileHome = Join-Path $work "home"
  $bin = Join-Path $work "bin"
  New-Item -ItemType Directory -Force $dist, $build, (Join-Path $profileHome ".claude"), (Join-Path $profileHome ".codex") | Out-Null

  Push-Location $root
  try {
    & go build -buildvcs=false -ldflags "-s -w -X main.version=$version" -o (Join-Path $build "jstack.exe") ./cmd/jstack
    Assert ($LASTEXITCODE -eq 0) "go build failed"
  } finally {
    Pop-Location
  }
  $archiveName = "jstack_${version}_windows_${arch}.zip"
  $archive = Join-Path $dist $archiveName
  Compress-Archive -LiteralPath (Join-Path $build "jstack.exe") -DestinationPath $archive
  $hash = (Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant()
  Set-Content -Path (Join-Path $dist "checksums.txt") -Value "$hash  $archiveName" -Encoding ascii

  # A fresh profile folder, so the setup run at the end of install.ps1 finds
  # the empty harness folders above and, with no terminal, changes nothing.
  $env:USERPROFILE = $profileHome
  Assert ((Invoke-Installer $dist $bin) -eq 0) "first install failed"
  Assert ((Invoke-Installer $dist $bin) -eq 0) "second install over the first failed"
  $installed = Join-Path $bin "jstack.exe"
  $reported = (& $installed --version | Out-String).Trim()
  Assert ($reported -eq $version) "installed jstack reports '$reported', want '$version'"
  Assert (-not (Test-Path (Join-Path $profileHome ".claude\skills\jstack-mode"))) "install.ps1 changed the profile without a terminal"

  $report = & $installed setup --harness claude,codex --yes | Out-String
  Write-Output $report
  Assert ($LASTEXITCODE -eq 0) "jstack setup exited with $LASTEXITCODE"
  foreach ($path in @(".claude\skills\jstack-mode\SKILL.md", ".claude\CLAUDE.md", ".codex\skills\jstack-mode\SKILL.md", ".codex\AGENTS.md", ".jstack\config.json")) {
    Assert (Test-Path (Join-Path $profileHome $path)) "setup did not write $path"
  }
  Assert ((Get-Content (Join-Path $profileHome ".claude\CLAUDE.md") -Raw).Contains("<!-- jstack:start -->")) "the letter is not in CLAUDE.md"
  Assert ($report.Contains("git and gh")) "the report does not name git and gh"
  if ($env:GH_TOKEN) {
    # gh is on the runner and GH_TOKEN makes gh auth status pass, so the
    # Windows check line, Get-Command through PowerShell, is what says ok.
    Assert ($report.Contains("ok git and gh")) "the Windows check line for git and gh did not pass"
  }
  Assert ($report.Contains("irm https://raw.githubusercontent.com/janiorvalle/quest/main/install.ps1 | iex")) "the report does not show the Windows install line for quest"

  $badDist = Join-Path $work "bad-dist"
  New-Item -ItemType Directory -Force $badDist | Out-Null
  Copy-Item $archive (Join-Path $badDist $archiveName)
  Set-Content -Path (Join-Path $badDist "checksums.txt") -Value ("0" * 64 + "  $archiveName") -Encoding ascii
  $badBin = Join-Path $work "bad-bin"
  Assert ((Invoke-Installer $badDist $badBin) -ne 0) "installer accepted a checksum mismatch"
  Assert (-not (Test-Path (Join-Path $badBin "jstack.exe"))) "checksum failure left a binary behind"
  Write-Output "install smoke passed"
} finally {
  $env:USERPROFILE = $savedProfile
  Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}
