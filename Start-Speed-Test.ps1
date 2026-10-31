# Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
[CmdletBinding()]
param(
    [ValidateRange(1, 65535)][int]$Port = 80
)

$ErrorActionPreference = 'Stop'
# Use PowerShell's intrinsic script directory. Do not pass a quoted %~dp0
# argument from CMD: its trailing backslash can corrupt the closing quote.
$bundleRoot = [System.IO.Path]::GetFullPath($PSScriptRoot)
$repo = 'ajayraikvar/speed-test'
$headers = @{ 'Accept' = 'application/vnd.github+json'; 'User-Agent' = 'AjayRaikvar-Speed-Test-Updater' }
$assetName = 'Speed-Test-Windows-x64.zip'
$dataRoot = Join-Path $env:LOCALAPPDATA 'AjayRaikvar\SpeedTest'
$versionsRoot = Join-Path $dataRoot 'versions'
$dataDir = $dataRoot
$tempRoot = Join-Path $dataRoot 'downloads'
New-Item -ItemType Directory -Force -Path $versionsRoot, $tempRoot, $dataDir | Out-Null

function Get-VersionFromTag([string]$Tag) {
    return ($Tag -replace '^v', '')
}
function Test-VersionGreater([string]$Candidate, [string]$Current) {
    try { return ([version](Get-VersionFromTag $Candidate)) -gt ([version](Get-VersionFromTag $Current)) }
    catch { return $false }
}
function Get-InstalledVersions {
    if (-not (Test-Path -LiteralPath $versionsRoot -PathType Container)) { return @() }
    return @(Get-ChildItem -LiteralPath $versionsRoot -Directory | Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName 'speed-test.exe') -PathType Leaf } | Sort-Object { try { [version](Get-VersionFromTag $_.Name) } catch { [version]'0.0.0' } } -Descending)
}
function Get-LatestRelease {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        return Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers $headers -TimeoutSec 8
    } catch {
        # GitHub returns 404 when the repository has no published non-draft release.
        # That is expected during pre-release testing; use the bundled/cached binary.
        $statusCode = 0
        try { if ($_.Exception.Response) { $statusCode = [int]$_.Exception.Response.StatusCode } } catch { }
        if ($statusCode -eq 404) { return $null }
        throw
    }
}

function Install-LatestRelease {
    $release = Get-LatestRelease
    if ($null -eq $release) { return $null }
    $tag = [string]$release.tag_name
    if ([string]::IsNullOrWhiteSpace($tag)) { throw 'GitHub did not return a release tag.' }
    if ($tag -notmatch '^v?\d+\.\d+\.\d+$') { throw 'The release tag has an invalid version format.' }
    $installed = Get-InstalledVersions
    $highest = if ($installed.Count -gt 0) { $installed[0].Name } else { '' }
    $needDownload = ($highest -eq '' -or (Test-VersionGreater $tag $highest))
    if (-not $needDownload) { return (Join-Path (Join-Path $versionsRoot $highest) 'speed-test.exe') }

    $zipAsset = @($release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1)
    $sumAsset = @($release.assets | Where-Object { $_.name -eq 'SHA256SUMS.txt' } | Select-Object -First 1)
    if ($zipAsset.Count -eq 0 -or $sumAsset.Count -eq 0) { throw 'Latest release is missing the ZIP or checksum asset.' }

    $zipPath = Join-Path $tempRoot ($tag + '-' + $assetName)
    $sumPath = Join-Path $tempRoot ($tag + '-SHA256SUMS.txt')
    Invoke-WebRequest -Uri $zipAsset[0].browser_download_url -OutFile $zipPath -Headers $headers -UseBasicParsing -TimeoutSec 120
    Invoke-WebRequest -Uri $sumAsset[0].browser_download_url -OutFile $sumPath -Headers $headers -UseBasicParsing -TimeoutSec 30
    $sumText = Get-Content -Raw -Path $sumPath
    $pattern = '(?im)^([a-f0-9]{64})\s+\*?' + [regex]::Escape($assetName) + '\s*$'
    $match = [regex]::Match($sumText, $pattern)
    if (-not $match.Success) { throw 'Checksum entry for the Windows ZIP was not found.' }
    $actualHash = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLowerInvariant()
    if ($actualHash -ne $match.Groups[1].Value.ToLowerInvariant()) { throw 'SHA-256 verification failed. The downloaded package was rejected.' }

    $version = Get-VersionFromTag $tag
    $target = Join-Path $versionsRoot $version
    if (Test-Path -LiteralPath $target -PathType Container) { return (Join-Path $target 'speed-test.exe') }
    $staging = Join-Path $versionsRoot ('.staging-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Force -Path $staging | Out-Null
    try {
        Expand-Archive -Path $zipPath -DestinationPath $staging -Force
        if (-not (Test-Path -LiteralPath (Join-Path $staging 'speed-test.exe') -PathType Leaf)) { throw 'The release archive does not contain speed-test.exe at its root.' }
        Move-Item -Path $staging -Destination $target
    } catch {
        if (Test-Path -LiteralPath $staging -PathType Container) { Remove-Item -LiteralPath $staging -Recurse -Force }
        throw
    }
    return (Join-Path $target 'speed-test.exe')
}

Write-Host 'Checking the latest Speed Test release...' -ForegroundColor Cyan
$exe = $null
try {
    $exe = Install-LatestRelease
    if ($exe) {
        Write-Host 'Latest release check completed.' -ForegroundColor Green
    } else {
        Write-Host 'No official GitHub release has been published yet. Using the bundled/offline copy.' -ForegroundColor Yellow
    }
} catch {
    Write-Warning ("Update check/download unavailable: " + $_.Exception.Message)
    Write-Host 'Trying the offline/local copy…' -ForegroundColor Yellow
}

if (-not $exe -or -not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    $bundleExe = Join-Path $bundleRoot 'speed-test.exe'
    if (Test-Path -LiteralPath $bundleExe -PathType Leaf) { $exe = $bundleExe }
}
if (-not $exe -or -not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    $installed = Get-InstalledVersions
    if ($installed.Count -gt 0) { $exe = Join-Path $installed[0].FullName 'speed-test.exe' }
}
if (-not $exe -or -not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    Write-Error 'No local server executable is available. For first-time offline use, extract the complete Speed-Test-Windows-x64.zip package, which includes speed-test.exe.'
    exit 10
}

Write-Host ''
Write-Host "Starting Speed Test with port $Port" -ForegroundColor Cyan
Write-Host "Persistent host data: $dataDir"
& $exe --port $Port --data-dir $dataDir
exit $LASTEXITCODE
