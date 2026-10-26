#requires -Version 5.1
[CmdletBinding()]
param(
    [string] $Version = '',
    [switch] $SkipBuild,
    [switch] $SkipInstaller,
    [string] $OutputDirectory = ''
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

$root = Get-ProjectRoot
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
}
Assert-SemanticVersion -Version $Version
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $root 'dist\packages'
}

$repositoryVersion = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
$embeddedVersionPath = Join-Path $root 'apps\server\internal\buildinfo\version.txt'
$embeddedMarkerPath = Join-Path $root 'apps\server\internal\buildinfo\marker.txt'
$embeddedVersion = (Get-Content -LiteralPath $embeddedVersionPath -Raw).Trim()
$embeddedMarker = (Get-Content -LiteralPath $embeddedMarkerPath -Raw).Trim()
if ($repositoryVersion -ne $Version -or $embeddedVersion -ne $Version) {
    throw "Versioonid ei ühti: package=$Version root=$repositoryVersion embedded=$embeddedVersion"
}
if ($embeddedMarker -ne "KODUMEDIA_VERSION=$Version") {
    throw "Vigane embedded build marker: $embeddedMarker"
}

if (-not $SkipBuild) {
    & (Join-Path $root 'build.ps1') -Version $Version
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

$binaryDirectory = Join-Path $root 'dist\windows-x64'
$server = Join-Path $binaryDirectory 'kodumedia.exe'
$updater = Join-Path $binaryDirectory 'kodumedia-updater.exe'
$updaterManifest = Join-Path $binaryDirectory 'kodumedia-updater.exe.manifest'
$buildInfoPath = Join-Path $binaryDirectory 'build-info.json'
foreach ($required in @($server, $updater, $updaterManifest, $buildInfoPath)) {
    if (-not (Test-Path -LiteralPath $required -PathType Leaf)) {
        throw "Puuduv buildi fail: $required"
    }
}

$buildInfo = Get-Content -LiteralPath $buildInfoPath -Raw | ConvertFrom-Json
if ($buildInfo.version -ne $Version) {
    throw "Build-info versioon $($buildInfo.version) ei vasta paketiversioonile $Version"
}
$serverHash = Get-LowerSha256 -Path $server
$updaterHash = Get-LowerSha256 -Path $updater
if ($serverHash -ne [string] $buildInfo.server.sha256) {
    throw "kodumedia.exe räsi ei vasta build-info failile"
}
if ($updaterHash -ne [string] $buildInfo.updater.sha256) {
    throw "kodumedia-updater.exe räsi ei vasta build-info failile"
}
$serverSize = [int64]((Get-Item -LiteralPath $server).Length)
$updaterSize = [int64]((Get-Item -LiteralPath $updater).Length)
if ($serverSize -ne [int64] $buildInfo.server.size) {
    throw "kodumedia.exe suurus ei vasta build-info failile"
}
if ($updaterSize -ne [int64] $buildInfo.updater.size) {
    throw "kodumedia-updater.exe suurus ei vasta build-info failile"
}
Assert-BinaryContainsVersionMarker -Path $server -Version $Version

$packageName = "kodumedia-$Version-windows-x64"
$work = Join-Path $root 'dist\package-work'
$stage = Join-Path $work $packageName
if (Test-Path -LiteralPath $work) {
    Remove-Item -LiteralPath $work -Recurse -Force
}
New-Item -ItemType Directory -Path $stage -Force | Out-Null

function Copy-ReleaseFile {
    param([string] $RelativePath, [string] $SourcePath = '')
    if ([string]::IsNullOrWhiteSpace($SourcePath)) {
        $SourcePath = Join-Path $root $RelativePath
    }
    if (-not (Test-Path -LiteralPath $SourcePath -PathType Leaf)) {
        throw "Väljalaskest puudub fail: $SourcePath"
    }
    $destination = Join-Path $stage $RelativePath
    New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
    Copy-Item -LiteralPath $SourcePath -Destination $destination -Force
}

Copy-ReleaseFile -RelativePath 'kodumedia.exe' -SourcePath $server
Copy-ReleaseFile -RelativePath 'kodumedia-updater.exe' -SourcePath $updater
Copy-ReleaseFile -RelativePath 'kodumedia-updater.exe.manifest' -SourcePath $updaterManifest
Copy-ReleaseFile -RelativePath 'build-info.json' -SourcePath $buildInfoPath
foreach ($file in @(
    '.env.example',
    'VERSION',
    'README-FIRST.txt',
    'README.md',
    'CHANGELOG.md',
    'THIRD-PARTY-NOTICES.txt',
    'start-kodumedia.cmd',
    'start-kodumedia.ps1',
    'smoke-test.ps1',
    'scripts\windows\Common.ps1',
    'api\openapi.yaml',
    'docs\ARCHITECTURE.md',
    'docs\SECURITY.md',
    'docs\ROADMAP.md',
    'docs\music\METADATA.md',
    'docs\Kodu-visuaalne-identiteet-CVI-v1.0.pdf',
    'tools\ffmpeg\bin\ffmpeg.exe',
    'tools\ffmpeg\bin\ffprobe.exe',
    'tools\ffmpeg\LICENSE-GPL-3.0.txt',
    'tools\ffmpeg\README.txt',
    'tools\ffmpeg\KODUMEDIA.txt',
    'tools\licenses\shazamio-core-MIT.txt'
)) {
    Copy-ReleaseFile -RelativePath $file
}

$files = @()
Get-ChildItem -LiteralPath $stage -Recurse -File |
    Sort-Object FullName |
    ForEach-Object {
        $relative = $_.FullName.Substring($stage.Length).TrimStart('\', '/').Replace('\', '/')
        $hash = Get-LowerSha256 -Path $_.FullName
        $files += [ordered]@{
            path = $relative
            sha256 = $hash
            size = [int64] $_.Length
        }
    }

$manifest = [ordered]@{
    schemaVersion = 1
    product = 'KoduMedia'
    version = $Version
    platform = 'windows-amd64'
    entryPoint = 'kodumedia.exe'
    files = $files
}
$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
$manifestPath = Join-Path $stage 'release-manifest.json'
[System.IO.File]::WriteAllText($manifestPath, ($manifest | ConvertTo-Json -Depth 8), $utf8WithoutBom)

New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
$zipPath = Join-Path $OutputDirectory "$packageName.zip"
if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

# ZipFile.CreateFromDirectory on Windows PowerShell/.NET Framework stores entry
# names with Windows backslashes. The update validator deliberately accepts
# only ZIP-standard forward slashes so an archive path has one unambiguous
# meaning on every platform. Create each entry explicitly with a normalized
# name instead of inheriting the host OS directory separator.
$archive = [System.IO.Compression.ZipFile]::Open(
    $zipPath,
    [System.IO.Compression.ZipArchiveMode]::Create
)
try {
    Get-ChildItem -LiteralPath $work -Recurse -File |
        Sort-Object FullName |
        ForEach-Object {
            $entryName = $_.FullName.Substring($work.Length).TrimStart('\', '/').Replace('\', '/')
            if ([string]::IsNullOrWhiteSpace($entryName) -or $entryName.Contains('\')) {
                throw "Vigane ZIP-kirje tee: $entryName"
            }
            [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile(
                $archive,
                $_.FullName,
                $entryName,
                [System.IO.Compression.CompressionLevel]::Optimal
            ) | Out-Null
        }
}
finally {
    $archive.Dispose()
}

# Treat updater compatibility as a packaging postcondition. A green release
# build must never publish an archive that the currently installed version
# rejects before reading its manifest.
$verificationArchive = [System.IO.Compression.ZipFile]::OpenRead($zipPath)
try {
    $entryNames = @($verificationArchive.Entries | ForEach-Object { $_.FullName })
    $invalidEntry = $entryNames | Where-Object {
        [string]::IsNullOrWhiteSpace($_) -or $_.Contains('\') -or $_.StartsWith('/') -or $_.Contains(':')
    } | Select-Object -First 1
    if ($null -ne $invalidEntry) {
        throw "Uuenduspakett sisaldab uuendajaga ühildumatut ZIP-kirjet: $invalidEntry"
    }
    $expectedManifestEntry = "$packageName/release-manifest.json"
    if ($entryNames -notcontains $expectedManifestEntry) {
        throw "Uuenduspaketist puudub oodatud manifest: $expectedManifestEntry"
    }
    if ($entryNames.Count -ne ($files.Count + 1)) {
        throw "Uuenduspaketi failide arv ei vasta manifestile."
    }
}
finally {
    $verificationArchive.Dispose()
}

Write-Host "`nValmis KoduMedia uuenduspakett:" -ForegroundColor Green
Write-Host $zipPath
Write-Host (Get-LowerSha256 -Path $zipPath)

if (-not $SkipInstaller) {
    & (Join-Path $root 'installer\build.ps1') `
        -Version $Version `
        -PayloadDirectory $stage `
        -OutputDirectory $OutputDirectory
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
