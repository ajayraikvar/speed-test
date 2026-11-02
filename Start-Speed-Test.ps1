# Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
[CmdletBinding()]
param(
    [ValidateRange(1,65535)][int]$Port = 80,
    [ValidateRange(1,128)][int]$MaxClients = 16
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$bundleRoot = [System.IO.Path]::GetFullPath($PSScriptRoot)
$repo = 'ajayraikvar/speed-test'
$headers = @{ Accept = 'application/vnd.github+json'; 'User-Agent' = 'AjayRaikvar-Speed-Test-Updater' }
$assetName = 'Speed-Test-Windows-x64.zip'
$dataRoot = Join-Path $env:LOCALAPPDATA 'AjayRaikvar\SpeedTest'
$versionsRoot = Join-Path $dataRoot 'versions'
$tempRoot = Join-Path $dataRoot 'downloads'
New-Item -ItemType Directory -Force -Path $versionsRoot, $tempRoot, $dataRoot | Out-Null

function Get-Version([string]$Path) {
    try {
        $v = (Get-Content -LiteralPath $Path -Raw).Trim()
        if ($v -match '^\d+\.\d+\.\d+$') { return [version]$v }
    } catch { }
    return [version]'0.0.0'
}
function Get-VersionText([string]$Path) {
    try { return (Get-Content -LiteralPath $Path -Raw).Trim() } catch { return '0.0.0' }
}
function Get-LocalCandidates {
    $items = @()
    $bundled = Join-Path $bundleRoot 'speed-test.exe'
    if (Test-Path -LiteralPath $bundled -PathType Leaf) {
        $items += [pscustomobject]@{ Version = (Get-Version (Join-Path $bundleRoot 'VERSION')); Path = $bundled; Source = 'bundled package' }
    }
    if (Test-Path -LiteralPath $versionsRoot -PathType Container) {
        foreach ($dir in (Get-ChildItem -LiteralPath $versionsRoot -Directory -ErrorAction SilentlyContinue)) {
            $exe = Join-Path $dir.FullName 'speed-test.exe'
            if (Test-Path -LiteralPath $exe -PathType Leaf) {
                $verText = $dir.Name -replace '^v',''
                $ver = [version]'0.0.0'
                try { $ver = [version]$verText } catch { }
                $items += [pscustomobject]@{ Version = $ver; Path = $exe; Source = 'local version cache' }
            }
        }
    }
    return @($items | Sort-Object Version -Descending)
}
function Get-LatestRelease {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        return Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers $headers -TimeoutSec 8
    } catch {
        $statusCode = 0
        try { if ($_.Exception.Response) { $statusCode = [int]$_.Exception.Response.StatusCode } } catch { }
        if ($statusCode -eq 404) { return $null }
        throw
    }
}
function Install-Release([object]$Release) {
    $tag = [string]$Release.tag_name
    if ($tag -notmatch '^v?\d+\.\d+\.\d+$') { throw 'The latest release tag has an invalid semantic version.' }
    $versionText = $tag -replace '^v',''
    $version = [version]$versionText
    $target = Join-Path $versionsRoot $versionText
    $targetExe = Join-Path $target 'speed-test.exe'
    $targetSource = Join-Path $target ("Speed-Test-v" + $versionText + '-Source.zip')
    $targetVersionFile = Join-Path $target 'VERSION'
    if ((Test-Path -LiteralPath $targetExe -PathType Leaf) -and (Test-Path -LiteralPath $targetSource -PathType Leaf) -and (Test-Path -LiteralPath $targetVersionFile -PathType Leaf)) {
        return [pscustomobject]@{ Path=$targetExe; Version=$version; Source='local version cache' }
    }
    if (Test-Path -LiteralPath $target) { throw "A version directory already exists but is incomplete: $target. It was left untouched to avoid overwriting local files." }

    $zipAsset = @($Release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1)
    $sumAsset = @($Release.assets | Where-Object { $_.name -eq 'SHA256SUMS.txt' } | Select-Object -First 1)
    if ($zipAsset.Count -eq 0 -or $sumAsset.Count -eq 0) { throw 'Latest release is missing the Windows ZIP or SHA256SUMS.txt asset.' }
    $zipPath = Join-Path $tempRoot ($versionText + '-' + $assetName)
    $sumPath = Join-Path $tempRoot ($versionText + '-SHA256SUMS.txt')
    Invoke-WebRequest -Uri $zipAsset[0].browser_download_url -OutFile $zipPath -Headers $headers -UseBasicParsing -TimeoutSec 180
    Invoke-WebRequest -Uri $sumAsset[0].browser_download_url -OutFile $sumPath -Headers $headers -UseBasicParsing -TimeoutSec 30
    $sumText = Get-Content -LiteralPath $sumPath -Raw
    $match = [regex]::Match($sumText, '(?im)^([a-f0-9]{64})\s+\*?' + [regex]::Escape($assetName) + '\s*$')
    if (-not $match.Success) { throw 'The Windows package checksum was not found in SHA256SUMS.txt.' }
    $actual = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $match.Groups[1].Value.ToLowerInvariant()) { throw 'SHA-256 verification failed. The downloaded update was rejected.' }

    $staging = Join-Path $versionsRoot ('.staging-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Force -Path $staging | Out-Null
    try {
        Expand-Archive -LiteralPath $zipPath -DestinationPath $staging -Force
        $stagedExe = Join-Path $staging 'speed-test.exe'
        $stagedSource = Join-Path $staging ("Speed-Test-v" + $versionText + '-Source.zip')
        $stagedLicense = Join-Path $staging 'LICENSE'
        if (-not (Test-Path -LiteralPath $stagedExe -PathType Leaf)) { throw 'The release ZIP does not contain speed-test.exe at its root.' }
        if (-not (Test-Path -LiteralPath $stagedSource -PathType Leaf)) { throw 'The release ZIP does not contain the matching corresponding source archive.' }
        if (-not (Test-Path -LiteralPath $stagedLicense -PathType Leaf)) { throw 'The release ZIP does not contain the full LICENSE file.' }
        [System.IO.File]::WriteAllText((Join-Path $staging 'VERSION'), $versionText, [System.Text.UTF8Encoding]::new($false))
        Move-Item -LiteralPath $staging -Destination $target
    } catch {
        if (Test-Path -LiteralPath $staging -PathType Container) { Remove-Item -LiteralPath $staging -Recurse -Force }
        throw
    }
    return [pscustomobject]@{ Path=$targetExe; Version=$version; Source='downloaded official release' }
}

$ErrorActionPreference = 'Stop'
Write-Host "Speed Test v2.1 launcher · Port $Port · Max clients $MaxClients" -ForegroundColor Cyan
$local = @(Get-LocalCandidates)
$selected = if ($local.Count -gt 0) { $local[0] } else { $null }
try {
    Write-Host 'Checking the latest official GitHub release…' -ForegroundColor Cyan
    $release = Get-LatestRelease
    if ($null -ne $release) {
        $remoteText = ([string]$release.tag_name) -replace '^v',''
        $remoteVersion = [version]$remoteText
        $localVersion = if ($selected) { $selected.Version } else { [version]'0.0.0' }
        if ($remoteVersion -gt $localVersion) {
            Write-Host "New version $remoteText found. Downloading and verifying SHA-256…" -ForegroundColor Yellow
            $selected = Install-Release $release
            Write-Host "Installed update $remoteText in a version-specific folder." -ForegroundColor Green
        } else {
            Write-Host "Local version $localVersion is current or newer than the official release." -ForegroundColor Green
        }
    } else {
        Write-Host 'No published stable release was found. Using local/offline files.' -ForegroundColor Yellow
    }
} catch {
    Write-Warning ("Update check unavailable: " + $_.Exception.Message)
    Write-Host 'Falling back to the newest verified local copy.' -ForegroundColor Yellow
}
if (-not $selected) {
    $local = @(Get-LocalCandidates)
    if ($local.Count -gt 0) { $selected = $local[0] }
}
if (-not $selected -or -not (Test-Path -LiteralPath $selected.Path -PathType Leaf)) {
    Write-Error 'No local server executable is available. For first-time offline use, extract the complete Windows offline ZIP.'
    exit 10
}
Write-Host "Starting Speed Test $($selected.Version) from $($selected.Source)…" -ForegroundColor Green
Write-Host "Persistent host data: $dataRoot"
& $selected.Path --port $Port --max-clients $MaxClients --data-dir $dataRoot
exit $LASTEXITCODE
