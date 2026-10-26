#requires -Version 5.1
[CmdletBinding()]
param(
    [switch] $Dependencies
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')
$root = Get-ProjectRoot

$paths = @(
    (Join-Path $root 'dist'),
    (Join-Path $root '.dev'),
    (Join-Path $root 'apps\web\dist')
)
if ($Dependencies) {
    $paths += (Join-Path $root 'apps\web\node_modules')
}

foreach ($path in $paths) {
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Recurse -Force
    }
}

$embedded = Join-Path $root 'apps\server\internal\webui\dist'
if (Test-Path -LiteralPath $embedded) {
    Remove-Item -LiteralPath $embedded -Recurse -Force
}
New-Item -ItemType Directory -Path $embedded -Force | Out-Null
Set-Content -LiteralPath (Join-Path $embedded 'index.html') -Encoding UTF8 -Value '<!doctype html><html lang="et"><body><p>Käivita build.ps1.</p></body></html>'
