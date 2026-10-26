#requires -Version 5.1
[CmdletBinding()]
param(
    [string] $Version = '',

    [switch] $SkipNpmInstall
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
}
Assert-SemanticVersion -Version $Version

$webDirectory = Join-Path $root 'apps\web'
$serverDirectory = Join-Path $root 'apps\server'
$outputDirectory = Join-Path $root 'dist\windows-x64'
$serverOutputPath = Join-Path $outputDirectory 'kodumedia.exe'
$updaterOutputPath = Join-Path $outputDirectory 'kodumedia-updater.exe'
$updaterManifestSource = Join-Path $serverDirectory 'cmd\updater\kodumedia-updater.manifest'
$updaterManifestOutputPath = Join-Path $outputDirectory 'kodumedia-updater.exe.manifest'
$buildInfoOutputPath = Join-Path $outputDirectory 'build-info.json'

$npm = Resolve-NativeCommand @('npm.cmd', 'npm')
$node = Resolve-NativeCommand @('node.exe', 'node')
$go = Resolve-NativeCommand @('go.exe', 'go')

Sync-EmbeddedVersion -ProjectRoot $root -Version $Version

Invoke-CheckedCommand -FilePath $node -Arguments @('scripts/verify-public-registry.mjs') -WorkingDirectory $webDirectory

if (-not $SkipNpmInstall) {
    Invoke-CheckedCommand -FilePath $npm -Arguments @('ci', '--registry=https://registry.npmjs.org/') -WorkingDirectory $webDirectory
}
Invoke-CheckedCommand -FilePath $npm -Arguments @('run', 'build') -WorkingDirectory $webDirectory
Sync-WebAssets -ProjectRoot $root
Invoke-CheckedCommand -FilePath $node -Arguments @('scripts/verify-ui-foundation.mjs', '--require-embedded') -WorkingDirectory $webDirectory
Invoke-CheckedCommand -FilePath $go -Arguments @('test', './internal/buildinfo') -WorkingDirectory $serverDirectory

New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null

$previousGoOS = $env:GOOS
$previousGoArch = $env:GOARCH
$previousCgo = $env:CGO_ENABLED
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    Invoke-CheckedCommand `
        -FilePath $go `
        -Arguments @(
            'build',
            '-buildvcs=false',
            '-trimpath',
            '-ldflags', '-s -w -H windowsgui',
            '-o', $serverOutputPath,
            './cmd/mediaserver'
        ) `
        -WorkingDirectory $serverDirectory
    Invoke-CheckedCommand `
        -FilePath $go `
        -Arguments @(
            'build',
            '-buildvcs=false',
            '-trimpath',
            '-ldflags', '-s -w -H windowsgui',
            '-o', $updaterOutputPath,
            './cmd/updater'
        ) `
        -WorkingDirectory $serverDirectory
}
finally {
    $env:GOOS = $previousGoOS
    $env:GOARCH = $previousGoArch
    $env:CGO_ENABLED = $previousCgo
}

Copy-Item -LiteralPath $updaterManifestSource -Destination $updaterManifestOutputPath -Force
Assert-BinaryContainsVersionMarker -Path $serverOutputPath -Version $Version

$buildInfo = [ordered]@{
    schemaVersion = 1
    version = $Version
    platform = 'windows-amd64'
    server = [ordered]@{
        path = 'kodumedia.exe'
        sha256 = (Get-LowerSha256 -Path $serverOutputPath)
        size = [int64]((Get-Item -LiteralPath $serverOutputPath).Length)
    }
    updater = [ordered]@{
        path = 'kodumedia-updater.exe'
        sha256 = (Get-LowerSha256 -Path $updaterOutputPath)
        size = [int64]((Get-Item -LiteralPath $updaterOutputPath).Length)
    }
}
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($buildInfoOutputPath, ($buildInfo | ConvertTo-Json -Depth 6), $utf8WithoutBom)

Write-Host "`nValmis Windows x64 binaarid:" -ForegroundColor Green
Write-Host $serverOutputPath
Write-Host $updaterOutputPath
Write-Host $updaterManifestOutputPath
Write-Host $buildInfoOutputPath
