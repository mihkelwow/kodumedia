#requires -Version 5.1
[CmdletBinding()]
param(
    [switch] $SkipNpmInstall
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
$webDirectory = Join-Path $root 'apps\web'
$serverDirectory = Join-Path $root 'apps\server'
$developmentDirectory = Join-Path $root '.dev'
$developmentExecutable = Join-Path $developmentDirectory 'kodumedia-dev.exe'

$npm = Resolve-NativeCommand @('npm.cmd', 'npm')
$go = Resolve-NativeCommand @('go.exe', 'go')
Import-DotEnv -Path (Join-Path $root '.env')

if ([string]::IsNullOrWhiteSpace($env:MEDIA_SERVER_DATA_DIR)) {
    $env:MEDIA_SERVER_DATA_DIR = Join-Path $root 'data'
}

if (-not $SkipNpmInstall) {
    Invoke-CheckedCommand -FilePath $npm -Arguments @('ci') -WorkingDirectory $webDirectory
}

New-Item -ItemType Directory -Path $developmentDirectory -Force | Out-Null
Invoke-CheckedCommand `
    -FilePath $go `
    -Arguments @('build', '-buildvcs=false', '-trimpath', '-o', $developmentExecutable, './cmd/mediaserver') `
    -WorkingDirectory $serverDirectory

$server = Start-Process -FilePath $developmentExecutable -WorkingDirectory $root -PassThru -NoNewWindow
try {
    Write-Host 'API: http://localhost:8096' -ForegroundColor Cyan
    Write-Host 'Veebiarendus: http://localhost:5173' -ForegroundColor Cyan
    Invoke-CheckedCommand -FilePath $npm -Arguments @('run', 'dev') -WorkingDirectory $webDirectory
}
finally {
    if ($null -ne $server -and -not $server.HasExited) {
        Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue
        $server.WaitForExit()
    }
}
