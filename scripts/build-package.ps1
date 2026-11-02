# Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
[CmdletBinding()]
param(
    [string]$Version = '2.0.0'
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$DistRoot = Join-Path $Root 'dist'
$OutDir = Join-Path $DistRoot ("v" + $Version)
if (Test-Path -LiteralPath $OutDir) {
    throw "Output directory already exists: $OutDir. Move it aside or choose a new output directory; this script will not overwrite it."
}
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

Push-Location $Root
try {
    Write-Host 'Running Go and browser checks…' -ForegroundColor Cyan
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'go test failed.' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
    node --check web/app.js
    if ($LASTEXITCODE -ne 0) { throw 'web/app.js syntax check failed.' }
    node --check web/admin.js
    if ($LASTEXITCODE -ne 0) { throw 'web/admin.js syntax check failed.' }

    $SourceStage = Join-Path $OutDir 'source-stage'
    New-Item -ItemType Directory -Path $SourceStage -Force | Out-Null
    Get-ChildItem -LiteralPath $Root -Force | Where-Object { $_.Name -notin @('dist', '.git', '.idea', '.vscode') } | ForEach-Object {
        Copy-Item -LiteralPath $_.FullName -Destination $SourceStage -Recurse -Force
    }
    $SourceZip = Join-Path $OutDir ("Speed-Test-v" + $Version + '-Source.zip')
    Compress-Archive -Path (Join-Path $SourceStage '*') -DestinationPath $SourceZip -CompressionLevel Optimal
    Remove-Item -LiteralPath $SourceStage -Recurse -Force

    $PackageStage = Join-Path $OutDir 'windows-stage'
    New-Item -ItemType Directory -Path $PackageStage -Force | Out-Null
    $OldGOOS = $env:GOOS
    $OldGOARCH = $env:GOARCH
    $OldCGO = $env:CGO_ENABLED
    try {
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        $env:CGO_ENABLED = '0'
        go build -trimpath -ldflags ("-s -w -X main.appVersion=" + $Version) -o (Join-Path $PackageStage 'speed-test.exe') .
        if ($LASTEXITCODE -ne 0) { throw 'Windows x64 build failed.' }
    } finally {
        $env:GOOS = $OldGOOS
        $env:GOARCH = $OldGOARCH
        $env:CGO_ENABLED = $OldCGO
    }

    $RequiredFiles = @('Start-Speed-Test.bat','Start-Speed-Test.ps1','README.md','LICENSE','NOTICE','CHANGELOG.md','TEST-STATUS.md','VERSION')
    foreach ($Name in $RequiredFiles) {
        $Source = Join-Path $Root $Name
        if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) { throw "Required distribution file missing: $Name" }
        Copy-Item -LiteralPath $Source -Destination $PackageStage
    }
    Set-Content -LiteralPath (Join-Path $PackageStage 'VERSION') -Value $Version -Encoding ascii
    Copy-Item -LiteralPath $SourceZip -Destination $PackageStage

    $WindowsZip = Join-Path $OutDir 'Speed-Test-Windows-x64.zip'
    Compress-Archive -Path (Join-Path $PackageStage '*') -DestinationPath $WindowsZip -CompressionLevel Optimal
    $WindowsHash = (Get-FileHash -LiteralPath $WindowsZip -Algorithm SHA256).Hash.ToLowerInvariant()
    $SourceHash = (Get-FileHash -LiteralPath $SourceZip -Algorithm SHA256).Hash.ToLowerInvariant()
    @("$WindowsHash  Speed-Test-Windows-x64.zip", "$SourceHash  Speed-Test-v$Version-Source.zip") | Set-Content -LiteralPath (Join-Path $OutDir 'SHA256SUMS.txt') -Encoding ascii

    Write-Host ''
    Write-Host 'Package created:' -ForegroundColor Green
    Write-Host "  $WindowsZip"
    Write-Host "  $SourceZip"
    Write-Host "  $(Join-Path $OutDir 'SHA256SUMS.txt')"
} finally {
    Pop-Location
}
