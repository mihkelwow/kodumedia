#requires -Version 5.1
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $InstallDirectory,

    [Parameter(Mandatory = $true)]
    [string] $DataDirectory
)

$ErrorActionPreference = 'Stop'
$runKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$previousCommand = ''
$runValues = Get-ItemProperty -LiteralPath $runKey -ErrorAction SilentlyContinue
if ($null -ne $runValues -and $null -ne $runValues.KoduMedia) {
    $previousCommand = [string] $runValues.KoduMedia
}

function Get-ExecutableFromCommand {
    param([string] $Command)
    $trimmed = $Command.Trim()
    if ($trimmed -match '^"([^"]+)"') { return $Matches[1] }
    if ($trimmed -match '^([^\s]+)') { return $Matches[1] }
    return ''
}

function Get-ConfiguredDataDirectory {
    param([string] $EnvironmentFile, [string] $Fallback)
    if (-not (Test-Path -LiteralPath $EnvironmentFile -PathType Leaf)) { return $Fallback }
    foreach ($line in [System.IO.File]::ReadAllLines($EnvironmentFile)) {
        if ($line -match '^\s*(?:export\s+)?MEDIA_SERVER_DATA_DIR\s*=\s*(.*?)\s*$') {
            $value = $Matches[1].Trim()
            if ($value.Length -ge 2 -and (($value[0] -eq '"' -and $value[$value.Length - 1] -eq '"') -or ($value[0] -eq "'" -and $value[$value.Length - 1] -eq "'"))) {
                $value = $value.Substring(1, $value.Length - 2)
            }
            if (-not [string]::IsNullOrWhiteSpace($value)) { return $value }
        }
    }
    return $Fallback
}

$previousExecutable = Get-ExecutableFromCommand -Command $previousCommand
$previousRoot = ''
$previousEnvironment = ''
$previousData = ''
if (-not [string]::IsNullOrWhiteSpace($previousExecutable)) {
    $previousRoot = Split-Path -Parent $previousExecutable
    $previousEnvironment = Join-Path $previousRoot '.env'
    $previousData = Get-ConfiguredDataDirectory -EnvironmentFile $previousEnvironment -Fallback (Join-Path $previousRoot 'data')
}

$installedEnvironment = Join-Path $InstallDirectory '.env'
if ([string]::IsNullOrWhiteSpace($previousEnvironment) -and (Test-Path -LiteralPath $installedEnvironment -PathType Leaf)) {
    $previousEnvironment = $installedEnvironment
    $previousData = Get-ConfiguredDataDirectory -EnvironmentFile $installedEnvironment -Fallback $DataDirectory
}

New-Item -ItemType Directory -Path $DataDirectory -Force | Out-Null
if (-not [string]::IsNullOrWhiteSpace($previousData) -and
    (Test-Path -LiteralPath $previousData -PathType Container) -and
    -not [string]::Equals([System.IO.Path]::GetFullPath($previousData), [System.IO.Path]::GetFullPath($DataDirectory), [System.StringComparison]::OrdinalIgnoreCase)) {
    & robocopy.exe $previousData $DataDirectory /E /COPY:DAT /DCOPY:DAT /R:2 /W:1 /XD transcodes updates /XF kodumedia.log kodumedia.log.1 | Out-Null
    if ($LASTEXITCODE -ge 8) {
        throw "KoduMedia andmete kopeerimine ebaõnnestus (robocopy $LASTEXITCODE)."
    }
}

$environmentLines = New-Object System.Collections.Generic.List[string]
if (-not [string]::IsNullOrWhiteSpace($previousEnvironment) -and (Test-Path -LiteralPath $previousEnvironment -PathType Leaf)) {
    foreach ($line in [System.IO.File]::ReadAllLines($previousEnvironment)) {
        if ($line -notmatch '^\s*(?:export\s+)?MEDIA_SERVER_DATA_DIR\s*=') {
            $environmentLines.Add($line)
        }
    }
}
$environmentLines.Add('MEDIA_SERVER_DATA_DIR="' + $DataDirectory + '"')
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllLines((Join-Path $InstallDirectory '.env'), $environmentLines, $utf8WithoutBom)
