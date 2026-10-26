#requires -Version 5.1
[CmdletBinding()]
param()

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
Import-DotEnv -Path (Join-Path $root '.env')

if ([string]::IsNullOrWhiteSpace($env:MEDIA_SERVER_DATA_DIR)) {
    $env:MEDIA_SERVER_DATA_DIR = Join-Path $root 'data'
}

$candidates = @(
    (Join-Path $root 'kodumedia.exe'),
    (Join-Path $root 'dist\windows-x64\kodumedia.exe')
)
$executable = $candidates | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
if ([string]::IsNullOrWhiteSpace($executable)) {
    throw 'KoduMedia binaari ei leitud. Lähtekoodis käivita esmalt build.cmd.'
}

$process = Start-Process -FilePath $executable -WorkingDirectory $root -PassThru
Write-Host "KoduMedia käivitati system trayl. Protsessi ID: $($process.Id)" -ForegroundColor Cyan
