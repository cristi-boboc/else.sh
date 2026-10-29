# else.sh installer for Windows.
#
#   irm https://cristi-boboc.github.io/else.sh/install.ps1 | iex
#
# Environment:
#   ELSE_SH_VERSION       tag to install (default: latest release)
#   ELSE_SH_INSTALL_DIR   where to put the binary (default: %LOCALAPPDATA%\Programs\else.sh)
#   ELSE_SH_NO_HOOK=1     only install the binary, don't touch the PowerShell profile
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$repo = 'cristi-boboc/else.sh'
$dir = if ($env:ELSE_SH_INSTALL_DIR) { $env:ELSE_SH_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\else.sh' }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$version = $env:ELSE_SH_VERSION
if (-not $version) {
    $version = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
}

$asset = "else.sh_windows_$arch.zip"
$base = "https://github.com/$repo/releases/download/$version"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("else.sh-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "==> downloading else.sh $version (windows/$arch)"
    Invoke-WebRequest "$base/$asset" -OutFile "$tmp\$asset" -UseBasicParsing
    Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt" -UseBasicParsing

    $want = (Get-Content "$tmp\checksums.txt" | Where-Object { $_ -match " $([regex]::Escape($asset))$" }) -split ' ' | Select-Object -First 1
    $got = (Get-FileHash "$tmp\$asset" -Algorithm SHA256).Hash.ToLower()
    if (-not $want -or $want -ne $got) { throw "checksum mismatch for $asset" }

    Expand-Archive "$tmp\$asset" -DestinationPath $tmp -Force
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item "$tmp\else.sh.exe" (Join-Path $dir 'else.sh.exe') -Force
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
$exe = Join-Path $dir 'else.sh.exe'
Write-Host "==> installed $exe"

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $dir) {
    [Environment]::SetEnvironmentVariable('Path', "$dir;$userPath", 'User')
    Write-Host "==> added $dir to your user PATH"
}
$env:Path = "$dir;$env:Path"

if ($env:ELSE_SH_NO_HOOK -ne '1') {
    # Hook the PowerShell that is running this script (pwsh or Windows PowerShell).
    & $exe install (Get-Process -Id $PID).Path
}
