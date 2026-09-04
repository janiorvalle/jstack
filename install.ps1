# The whole installer runs in its own scope, so `irm ... | iex` in a terminal
# leaves that terminal's preferences, functions, and variables as they were.
& {
$ErrorActionPreference = "Stop"

function Fail([string]$Message) {
  throw "jstack installer: $Message"
}

function Publish-EnvironmentChange {
  if (-not ([System.Management.Automation.PSTypeName]"Jstack.EnvironmentChange").Type) {
    Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

namespace Jstack {
  public static class EnvironmentChange {
    [DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    public static extern IntPtr SendMessageTimeout(
      IntPtr window,
      uint message,
      UIntPtr messageParameter,
      string environmentName,
      uint flags,
      uint timeout,
      out UIntPtr result
    );
  }
}
"@
  }

  $result = [UIntPtr]::Zero
  [void][Jstack.EnvironmentChange]::SendMessageTimeout(
    [IntPtr]0xffff,
    [uint32]0x001A,
    [UIntPtr]::Zero,
    "Environment",
    [uint32]0x0002,
    [uint32]5000,
    [ref]$result
  )
}

$repo = if ($env:JSTACK_INSTALL_REPO) { $env:JSTACK_INSTALL_REPO } else { "janiorvalle/jstack" }
# The folder goes into the user PATH, so a relative override is made absolute
# against the current location first.
$installDir = if ($env:JSTACK_INSTALL_DIR) {
  $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($env:JSTACK_INSTALL_DIR)
} else {
  Join-Path $env:LOCALAPPDATA "Programs\jstack"
}
$version = $env:JSTACK_INSTALL_VERSION
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { Fail "unsupported architecture: $env:PROCESSOR_ARCHITECTURE; jstack ships for amd64 and arm64" }
}

# A token lifts the GitHub API's limit of sixty unauthenticated requests an
# hour per address, which shared machines such as CI runners hit.
$githubToken = if ($env:JSTACK_GITHUB_TOKEN) {
  $env:JSTACK_GITHUB_TOKEN
} elseif ($env:GH_TOKEN) {
  $env:GH_TOKEN
} else {
  $env:GITHUB_TOKEN
}
$apiHeaders = @{
  Accept = "application/vnd.github+json"
  "User-Agent" = "jstack-installer"
}
if ($githubToken) {
  $apiHeaders.Authorization = "Bearer $githubToken"
}

if (-not $version) {
  try {
    $release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest" -Headers $apiHeaders -ErrorAction Stop
  } catch {
    $status = if ($_.Exception.Response) { [int]$_.Exception.Response.StatusCode } else { 0 }
    if ($status -eq 403 -or $status -eq 429) {
      Fail "GitHub answered HTTP $status to the latest-release lookup for $repo, its rate limit for requests without a token. Set GH_TOKEN or JSTACK_GITHUB_TOKEN to a GitHub token, or set JSTACK_INSTALL_VERSION to skip the lookup, then retry."
    }
    if ($status -eq 0) {
      Fail "could not reach api.github.com for the latest release of $repo. Check the network, or set JSTACK_INSTALL_VERSION to skip the lookup, then retry."
    }
    Fail "GitHub answered HTTP $status to the latest-release lookup for $repo; see https://github.com/$repo/releases"
  }
  $version = [string]$release.tag_name
}
$version = $version.TrimStart("v")
$archiveName = "jstack_${version}_windows_${arch}.zip"
$baseUrl = if ($env:JSTACK_INSTALL_BASE_URL) {
  $env:JSTACK_INSTALL_BASE_URL.TrimEnd("/")
} else {
  "https://github.com/$repo/releases/download/v$version"
}

$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) "jstack-install-$PID"
New-Item -ItemType Directory -Force $temporaryDirectory | Out-Null
$archive = Join-Path $temporaryDirectory $archiveName
$checksums = Join-Path $temporaryDirectory "checksums.txt"
$extracted = Join-Path $temporaryDirectory "extracted"

function Download([string]$Url, [string]$Destination) {
  try {
    Invoke-WebRequest $Url -OutFile $Destination -UseBasicParsing -ErrorAction Stop
  } catch {
    Fail "could not download $Url"
  }
}

function Assert-JstackVersion([string]$Executable, [string]$Stage) {
  $reported = (& $Executable --version | Out-String).Trim()
  if ($LASTEXITCODE -ne 0) {
    Fail "$Stage jstack failed its version smoke test (exit $LASTEXITCODE): $reported. Confirm Windows security tools allow the release executable, then retry the installer."
  }
  if ($reported -ne $version) {
    Fail "$Stage jstack reported '$reported' instead of '$version'. Retry the installer; if it still fails, report both versions."
  }
}

try {
  if ($env:JSTACK_INSTALL_ARCHIVE -or $env:JSTACK_INSTALL_CHECKSUMS) {
    if (-not $env:JSTACK_INSTALL_ARCHIVE -or -not $env:JSTACK_INSTALL_CHECKSUMS) {
      Fail "JSTACK_INSTALL_ARCHIVE and JSTACK_INSTALL_CHECKSUMS must be set together"
    }
    Copy-Item $env:JSTACK_INSTALL_ARCHIVE $archive
    Copy-Item $env:JSTACK_INSTALL_CHECKSUMS $checksums
  } else {
    Write-Output "Downloading jstack $version for windows/$arch..."
    Download "$baseUrl/$archiveName" $archive
    Download "$baseUrl/checksums.txt" $checksums
  }

  $escapedArchiveName = [regex]::Escape($archiveName)
  $checksumLine = Get-Content $checksums |
    Where-Object { $_ -match "^[0-9a-fA-F]{64}\s+\*?$escapedArchiveName$" } |
    Select-Object -First 1
  if (-not $checksumLine) {
    Fail "checksums.txt has no entry for $archiveName"
  }
  $expected = ($checksumLine -split "\s+")[0].ToLowerInvariant()
  $actual = (Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant()
  if ($actual -ne $expected) {
    Fail "checksum mismatch for $archiveName"
  }

  Expand-Archive -LiteralPath $archive -DestinationPath $extracted -Force
  $downloaded = Get-ChildItem -LiteralPath $extracted -Recurse -Filter "jstack.exe" | Select-Object -First 1
  if (-not $downloaded) {
    Fail "archive did not contain jstack.exe"
  }
  Assert-JstackVersion $downloaded.FullName "downloaded"

  New-Item -ItemType Directory -Force $installDir | Out-Null
  $destination = Join-Path $installDir "jstack.exe"
  $stage = Join-Path $installDir ".jstack.new.$PID.exe"
  $previous = Join-Path $temporaryDirectory ".jstack.previous.exe"
  $hadPrevious = Test-Path -LiteralPath $destination
  Copy-Item $downloaded.FullName $stage
  if ($hadPrevious) {
    Copy-Item $destination $previous
    Remove-Item -Force $destination
  }
  try {
    Move-Item -Force $stage $destination
    Assert-JstackVersion $destination "installed"
  } catch {
    # Whatever failed, the folder ends up as it was: the staged and the
    # unusable new file go, the previous binary comes back.
    Remove-Item -Force $stage, $destination -ErrorAction SilentlyContinue
    if ($hadPrevious -and -not (Test-Path -LiteralPath $destination)) {
      Copy-Item $previous $destination -ErrorAction SilentlyContinue
    }
    throw
  }

  Write-Output "Installed jstack $version to $destination"
  if (-not $env:JSTACK_INSTALL_SKIP_PATH) {
    $environmentKey = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment", $true)
    if (-not $environmentKey) {
      Fail "could not open the user environment registry key"
    }
    try {
      $rawUserPath = [string]$environmentKey.GetValue(
        "Path",
        "",
        [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames
      )
      $expandedUserPath = [string]$environmentKey.GetValue("Path", "")
      $expandedEntries = @($expandedUserPath -split ";" | Where-Object { $_ })
      if ($installDir -notin $expandedEntries) {
        $rawEntries = @($rawUserPath -split ";" | Where-Object { $_ })
        $updatedPath = (@($rawEntries) + $installDir) -join ";"
        $pathKind = [Microsoft.Win32.RegistryValueKind]::ExpandString
        if ($environmentKey.GetValueNames() -contains "Path") {
          $pathKind = $environmentKey.GetValueKind("Path")
        }
        if (
          $pathKind -ne [Microsoft.Win32.RegistryValueKind]::String -and
          $pathKind -ne [Microsoft.Win32.RegistryValueKind]::ExpandString
        ) {
          Fail "the user PATH registry value is not a string"
        }
        $environmentKey.SetValue("Path", $updatedPath, $pathKind)
        Publish-EnvironmentChange
        Write-Output "Added $installDir to your user PATH. Open a new terminal before running jstack."
      }
    } finally {
      $environmentKey.Dispose()
    }
  }

  # Same as install.sh: setup runs once. With a terminal it asks its
  # questions; without one it prints the plan and the flags and changes nothing.
  & $destination setup
  if ($LASTEXITCODE -ne 0) {
    Fail "jstack setup exited with $LASTEXITCODE; the binary is installed, rerun jstack setup"
  }
} finally {
  Remove-Item -Recurse -Force $temporaryDirectory -ErrorAction SilentlyContinue
}
}
