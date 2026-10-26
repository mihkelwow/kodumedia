Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$script:KoduMediaProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path

function Get-ProjectRoot {
    return $script:KoduMediaProjectRoot
}

function Resolve-NativeCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string[]] $Candidates
    )

    foreach ($candidate in $Candidates) {
        $command = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($null -ne $command) {
            return $command.Source
        }
    }

    throw "Vajalik käsk puudub: $($Candidates -join ', ')"
}

function Invoke-CheckedCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string] $FilePath,

        [string[]] $Arguments = @(),

        [Parameter(Mandatory = $true)]
        [string] $WorkingDirectory
    )

    Push-Location $WorkingDirectory
    try {
        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "Käsk ebaõnnestus väljumiskoodiga $LASTEXITCODE: $FilePath $($Arguments -join ' ')"
        }
    }
    finally {
        Pop-Location
    }
}

function Import-DotEnv {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Path
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return
    }

    foreach ($rawLine in Get-Content -LiteralPath $Path) {
        $line = $rawLine.Trim()
        if ($line.Length -eq 0 -or $line.StartsWith('#')) {
            continue
        }

        $separator = $line.IndexOf('=')
        if ($separator -le 0) {
            throw "Vigane .env rida: $rawLine"
        }

        $name = $line.Substring(0, $separator).Trim()
        $value = $line.Substring($separator + 1).Trim()
        if ($name -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') {
            throw "Vigane keskkonnamuutuja nimi .env failis: $name"
        }

        if ($value.Length -ge 2) {
            $first = $value[0]
            $last = $value[$value.Length - 1]
            if (($first -eq '"' -and $last -eq '"') -or ($first -eq "'" -and $last -eq "'")) {
                $value = $value.Substring(1, $value.Length - 2)
            }
        }

        [Environment]::SetEnvironmentVariable($name, $value, 'Process')
    }
}

function Sync-WebAssets {
    param(
        [Parameter(Mandatory = $true)]
        [string] $ProjectRoot
    )

    $source = Join-Path $ProjectRoot 'apps\web\dist'
    $destination = Join-Path $ProjectRoot 'apps\server\internal\webui\dist'
    if (-not (Test-Path -LiteralPath $source -PathType Container)) {
        throw "Veebikliendi build puudub: $source"
    }

    if (Test-Path -LiteralPath $destination) {
        Remove-Item -LiteralPath $destination -Recurse -Force
    }
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
    Copy-Item -Path (Join-Path $source '*') -Destination $destination -Recurse -Force
}

function Assert-SemanticVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Version
    )

    if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$') {
        throw "Vigane SemVer versioon: $Version"
    }
}

function Sync-EmbeddedVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string] $ProjectRoot,

        [Parameter(Mandatory = $true)]
        [string] $Version
    )

    Assert-SemanticVersion -Version $Version
    $directory = Join-Path $ProjectRoot 'apps\server\internal\buildinfo'
    $versionPath = Join-Path $directory 'version.txt'
    $markerPath = Join-Path $directory 'marker.txt'
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
    $utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($versionPath, "$Version`n", $utf8WithoutBom)
    [System.IO.File]::WriteAllText($markerPath, "KODUMEDIA_VERSION=$Version`n", $utf8WithoutBom)
}

function Get-LowerSha256 {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Path
    )

    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Assert-BinaryContainsVersionMarker {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Path,

        [Parameter(Mandatory = $true)]
        [string] $Version
    )

    $marker = "KODUMEDIA_VERSION=$Version"
    $bytes = [System.IO.File]::ReadAllBytes($Path)
    $ascii = [System.Text.Encoding]::ASCII.GetString($bytes)
    if (-not $ascii.Contains($marker)) {
        throw "Binaar $Path ei sisalda oodatud build markerit $marker"
    }
}
