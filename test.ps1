#requires -Version 5.1
[CmdletBinding()]
param(
    [switch] $SkipNpmInstall
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
$webDirectory = Join-Path $root 'apps\web'
$serverDirectory = Join-Path $root 'apps\server'
$npm = Resolve-NativeCommand @('npm.cmd', 'npm')
$node = Resolve-NativeCommand @('node.exe', 'node')
$go = Resolve-NativeCommand @('go.exe', 'go')

Invoke-CheckedCommand -FilePath $node -Arguments @('scripts/verify-public-registry.mjs') -WorkingDirectory $webDirectory

if (-not $SkipNpmInstall) {
    Invoke-CheckedCommand -FilePath $npm -Arguments @('ci', '--registry=https://registry.npmjs.org/') -WorkingDirectory $webDirectory
}
Invoke-CheckedCommand -FilePath $npm -Arguments @('run', 'typecheck') -WorkingDirectory $webDirectory
Invoke-CheckedCommand -FilePath $npm -Arguments @('test') -WorkingDirectory $webDirectory
Invoke-CheckedCommand -FilePath $npm -Arguments @('run', 'build') -WorkingDirectory $webDirectory
Sync-WebAssets -ProjectRoot $root
Invoke-CheckedCommand -FilePath $go -Arguments @('test', './...') -WorkingDirectory $serverDirectory
Invoke-CheckedCommand -FilePath $go -Arguments @('vet', './...') -WorkingDirectory $serverDirectory

Write-Host "`nKõik kontrollid läbisid." -ForegroundColor Green
