#requires -Version 5.1
[CmdletBinding()]
param(
    [switch] $Build,
    [string] $Version = ''
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
$executable = Join-Path $root 'dist\windows-x64\kodumedia.exe'

if ($Build -or -not (Test-Path -LiteralPath $executable -PathType Leaf)) {
    & (Join-Path $root 'build.ps1') -Version $Version
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}

Import-DotEnv -Path (Join-Path $root '.env')

Write-Host 'KoduMedia käivitub aadressil http://localhost:8096' -ForegroundColor Cyan
Push-Location $root
try {
    & $executable
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
