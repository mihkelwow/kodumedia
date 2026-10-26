#requires -Version 5.1
[CmdletBinding()]
param(
    [string] $Version = '',
    [string] $PayloadDirectory = '',
    [string] $OutputDirectory = ''
)

. (Join-Path (Split-Path -Parent $PSScriptRoot) 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
}
Assert-SemanticVersion -Version $Version
if ([string]::IsNullOrWhiteSpace($PayloadDirectory)) {
    $PayloadDirectory = Join-Path $root "dist\package-work\kodumedia-$Version-windows-x64"
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $root 'dist\packages'
}

$makensis = Resolve-NativeCommand @('makensis.exe', 'makensis')
$script = Join-Path $PSScriptRoot 'kodumedia.nsi'
$output = Join-Path $OutputDirectory "kodumedia-$Version-setup-x64.exe"
foreach ($required in @($script, $PayloadDirectory)) {
    if (-not (Test-Path -LiteralPath $required)) { throw "Puuduv installeri sisend: $required" }
}
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null

Invoke-CheckedCommand -FilePath $makensis -Arguments @(
    '/V3',
    "/DVERSION=$Version",
    "/DPAYLOAD_DIR=$PayloadDirectory",
    "/DSOURCE_DIR=$root",
    "/DOUTPUT_FILE=$output",
    $script
) -WorkingDirectory $root

Write-Host "`nValmis KoduMedia installer:" -ForegroundColor Green
Write-Host $output
Write-Host (Get-LowerSha256 -Path $output)
