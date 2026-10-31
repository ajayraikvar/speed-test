[CmdletBinding()]
param([string]$Version = '1.0.0')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root ('dist-' + $Version)
if (Test-Path $dist) { throw "Output folder already exists; nothing was overwritten: $dist" }
$package = Join-Path $dist 'package'
New-Item -ItemType Directory -Force -Path $package | Out-Null
Push-Location $root
try {
    go test ./...
    go vet ./...
    go test -race ./...
    go build -trimpath -ldflags "-s -w -X main.appVersion=$Version" -o (Join-Path $package 'speed-test.exe') .
    Copy-Item Start-Speed-Test.bat, Start-Speed-Test.ps1, README.md, LICENSE, NOTICE -Destination $package
    $zip = Join-Path $dist 'Speed-Test-Windows-x64.zip'
    Compress-Archive -Path (Join-Path $package '*') -DestinationPath $zip -Force
    $hash = (Get-FileHash -Algorithm SHA256 $zip).Hash.ToLowerInvariant()
    "$hash  Speed-Test-Windows-x64.zip" | Set-Content -Encoding ascii (Join-Path $dist 'SHA256SUMS.txt')
    Write-Host "Created $zip"
    Write-Host "SHA256 $hash"
} finally { Pop-Location }
