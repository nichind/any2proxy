$ErrorActionPreference = "Stop"

$repo = if ($env:A2P_REPO) { $env:A2P_REPO } else { "nichind/any2proxy" }
$version = if ($env:A2P_VERSION) { $env:A2P_VERSION } else { "latest" }
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

if ($version -eq "latest") {
    $base = "https://github.com/$repo/releases/latest/download"
} else {
    $base = "https://github.com/$repo/releases/download/$version"
}

$archive = "a2p_windows_${arch}.zip"
$tmp = Join-Path $env:TEMP ("a2p-install-" + [guid]::NewGuid())
New-Item -ItemType Directory -Force $tmp | Out-Null

Invoke-WebRequest "$base/$archive" -OutFile (Join-Path $tmp $archive)
Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $tmp "checksums.txt")

$line = Get-Content (Join-Path $tmp "checksums.txt") | Where-Object { $_ -match " $archive$" }
$expected = ($line -split " ")[0]
$actual = (Get-FileHash (Join-Path $tmp $archive) -Algorithm SHA256).Hash.ToLower()
if ($expected -ne $actual) { throw "checksum mismatch" }

Expand-Archive (Join-Path $tmp $archive) -DestinationPath $tmp
$bin = Join-Path $env:LOCALAPPDATA "any2proxy\bin"
New-Item -ItemType Directory -Force $bin | Out-Null
Copy-Item (Join-Path $tmp "a2p.exe") $bin -Force
Copy-Item (Join-Path $tmp "a2pd.exe") $bin -Force
Remove-Item -Recurse -Force $tmp
Write-Output "installed to $bin; add it to PATH"
