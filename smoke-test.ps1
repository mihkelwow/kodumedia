#requires -Version 5.1
[CmdletBinding()]
param(
    [switch] $Build
)

. (Join-Path $PSScriptRoot 'scripts\windows\Common.ps1')

function Assert-Condition {
    param(
        [Parameter(Mandatory = $true)]
        [bool] $Condition,

        [Parameter(Mandatory = $true)]
        [string] $Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

function Get-FreeTcpPort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    try {
        $listener.Start()
        return ([System.Net.IPEndPoint] $listener.LocalEndpoint).Port
    }
    finally {
        $listener.Stop()
    }
}

function ConvertTo-RequestJson {
    param(
        [Parameter(Mandatory = $true)]
        [object] $Value
    )

    return $Value | ConvertTo-Json -Depth 12 -Compress
}

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory = $true)]
        [string] $Uri,

        [ValidateSet('GET', 'POST', 'PUT', 'PATCH', 'DELETE')]
        [string] $Method = 'GET',

        [AllowNull()]
        [object] $Body = $null
    )

    if ($null -eq $Body) {
        return Invoke-RestMethod -Uri $Uri -Method $Method -TimeoutSec 20
    }

    return Invoke-RestMethod `
        -Uri $Uri `
        -Method $Method `
        -ContentType 'application/json; charset=utf-8' `
        -Body (ConvertTo-RequestJson -Value $Body) `
        -TimeoutSec 20
}

function Wait-ForServer {
    param(
        [Parameter(Mandatory = $true)]
        [string] $BaseUrl,

        [Parameter(Mandatory = $true)]
        [System.Diagnostics.Process] $Process,

        [Parameter(Mandatory = $true)]
        [string] $StdoutPath,

        [Parameter(Mandatory = $true)]
        [string] $StderrPath
    )

    for ($attempt = 0; $attempt -lt 150; $attempt++) {
        $Process.Refresh()
        if ($Process.HasExited) {
            $stdoutText = if (Test-Path -LiteralPath $StdoutPath) { Get-Content -LiteralPath $StdoutPath -Raw } else { '' }
            $stderrText = if (Test-Path -LiteralPath $StderrPath) { Get-Content -LiteralPath $StderrPath -Raw } else { '' }
            throw "Server lõpetas käivitumise ajal. STDOUT: $stdoutText STDERR: $stderrText"
        }

        try {
            $health = Invoke-JsonRequest -Uri "$BaseUrl/api/v1/health"
            if ($health.status -eq 'ok') {
                return $health
            }
        }
        catch {
            Start-Sleep -Milliseconds 100
        }
    }

    throw "Server ei muutunud valmisolekusse. Logid: $StdoutPath ja $StderrPath"
}

function Wait-ForJob {
    param(
        [Parameter(Mandatory = $true)]
        [string] $BaseUrl,

        [Parameter(Mandatory = $true)]
        [string] $JobId
    )

    $encodedJobId = [System.Uri]::EscapeDataString($JobId)
    for ($attempt = 0; $attempt -lt 300; $attempt++) {
        $job = Invoke-JsonRequest -Uri "$BaseUrl/api/v1/jobs/$encodedJobId"
        if ($job.status -eq 'succeeded') {
            return $job
        }
        if ($job.status -eq 'failed') {
            throw "Taustatöö ebaõnnestus: $($job.error)"
        }
        Start-Sleep -Milliseconds 100
    }

    throw "Taustatöö $JobId ei lõpetanud 30 sekundi jooksul."
}

$root = Get-ProjectRoot
$buildScript = Join-Path $root 'build.ps1'
$rootExecutable = Join-Path $root 'kodumedia.exe'
$sourceExecutable = Join-Path $root 'dist\windows-x64\kodumedia.exe'
$rootUpdater = Join-Path $root 'kodumedia-updater.exe'
$sourceUpdater = Join-Path $root 'dist\windows-x64\kodumedia-updater.exe'

if ($Build) {
    if (-not (Test-Path -LiteralPath $buildScript -PathType Leaf)) {
        throw 'Parameetrit -Build saab kasutada ainult lähtekoodipakis, kus build.ps1 on olemas.'
    }
    & $buildScript
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}

$executable = @($rootExecutable, $sourceExecutable) |
    Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } |
    Select-Object -First 1

if ([string]::IsNullOrWhiteSpace($executable)) {
    if (Test-Path -LiteralPath $buildScript -PathType Leaf) {
        & $buildScript
        if ($LASTEXITCODE -ne 0) {
            exit $LASTEXITCODE
        }
        $executable = $sourceExecutable
    }
    else {
        throw 'KoduMedia binaari ei leitud.'
    }
}

$updaterExecutable = @($rootUpdater, $sourceUpdater) |
    Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } |
    Select-Object -First 1
Assert-Condition -Condition (-not [string]::IsNullOrWhiteSpace($updaterExecutable)) -Message 'kodumedia-updater.exe binaari ei leitud.'
$bundledFFmpeg = Join-Path $root 'tools\ffmpeg\bin\ffmpeg.exe'
$bundledFFprobe = Join-Path $root 'tools\ffmpeg\bin\ffprobe.exe'
Assert-Condition -Condition (Test-Path -LiteralPath $bundledFFmpeg -PathType Leaf) -Message 'Bundled ffmpeg.exe puudub.'
Assert-Condition -Condition (Test-Path -LiteralPath $bundledFFprobe -PathType Leaf) -Message 'Bundled ffprobe.exe puudub.'

$port = Get-FreeTcpPort
$work = Join-Path ([System.IO.Path]::GetTempPath()) ("kodumedia-smoke-" + [Guid]::NewGuid().ToString('N'))
$dataDirectory = Join-Path $work 'data'
$mediaDirectory = Join-Path $work 'media'
$seriesDirectory = Join-Path $mediaDirectory 'Severance'
$seasonOneDirectory = Join-Path $seriesDirectory 'Season 01'
$seasonTwoDirectory = Join-Path $seriesDirectory 'Season 02'
$lotteDirectory = Join-Path $mediaDirectory '[ UsaBit.com ] - Lotte.And.The.Moonstone.Secret.2011.DVDRip.XviD-Ltu'

New-Item -ItemType Directory -Path $dataDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $seasonOneDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $seasonTwoDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $lotteDirectory -Force | Out-Null

$utf8WithoutBom = New-Object System.Text.UTF8Encoding($false)
$dunePath = Join-Path $mediaDirectory 'Dune.2021.1080p.mkv'
& $bundledFFmpeg @(
    '-hide_banner', '-loglevel', 'error', '-y',
    '-f', 'lavfi', '-i', 'testsrc2=size=640x360:rate=24',
    '-f', 'lavfi', '-i', 'sine=frequency=880:sample_rate=48000',
    '-t', '6',
    '-c:v', 'libx264', '-preset', 'ultrafast', '-pix_fmt', 'yuv420p',
    '-c:a', 'aac', '-b:a', '96k',
    $dunePath
)
if ($LASTEXITCODE -ne 0) {
    throw 'Playback smoke-fixture loomine FFmpegiga ebaõnnestus.'
}
[System.IO.File]::WriteAllText((Join-Path $mediaDirectory 'Dune.2021.1080p.et.srt'), "1`n00:00:00,000 --> 00:00:01,000`nTere maailm!`n", $utf8WithoutBom)
[System.IO.File]::WriteAllText((Join-Path $mediaDirectory 'Legacy.Album.ape'), '', $utf8WithoutBom)
[System.IO.File]::WriteAllText((Join-Path $mediaDirectory '[Anime Time] Suzume No Tojimari [1080p][HEVC 10bit x265][AAC][Multi Sub].mkv'), '', $utf8WithoutBom)
[System.IO.File]::WriteAllText((Join-Path $mediaDirectory 'bonkai77] Wolf Children.2012.1080p.BluRay.x265.mkv'), '', $utf8WithoutBom)
[System.IO.File]::WriteAllText((Join-Path $lotteDirectory 'ltu-lotte.avi'), '', $utf8WithoutBom)
[System.IO.File]::WriteAllText((Join-Path $seasonOneDirectory '01 - Good News.mkv'), '', $utf8WithoutBom)
[System.IO.File]::WriteAllText((Join-Path $seasonTwoDirectory 'Severance.S02E03.Who.Is.Alive.mkv'), '', $utf8WithoutBom)
$ratingsPath = Join-Path $work 'title.ratings.tsv'
[System.IO.File]::WriteAllText($ratingsPath, "tconst`taverageRating`tnumVotes`ntt0000001`t8.2`t123`n", $utf8WithoutBom)

$legacyCatalog = @'
{
  "version": 1,
  "libraries": {
    "library-movies": {
      "id": "library-movies",
      "name": "Filmid",
      "kind": "movies",
      "system": true,
      "sources": null,
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-01T00:00:00Z"
    }
  },
  "media": {},
  "mediaPaths": {}
}
'@
[System.IO.File]::WriteAllText((Join-Path $dataDirectory 'catalog.json'), $legacyCatalog, $utf8WithoutBom)

$stdout = Join-Path $work 'stdout.log'
$stderr = Join-Path $work 'stderr.log'
$environmentNames = @(
    'MEDIA_SERVER_ADDRESS',
    'MEDIA_SERVER_DATA_DIR',
    'MEDIA_SERVER_FFMPEG',
    'MEDIA_SERVER_FFPROBE',
    'TMDB_TOKEN',
    'IMDB_RATINGS_SOURCE',
    'IMDB_RATINGS_PATH'
)
$previousEnvironment = @{}
foreach ($name in $environmentNames) {
    $previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

$server = $null
try {
    $env:MEDIA_SERVER_ADDRESS = "127.0.0.1:$port"
    $env:MEDIA_SERVER_DATA_DIR = $dataDirectory
    $env:MEDIA_SERVER_FFMPEG = '__kodumedia_missing_ffmpeg__'
    $env:MEDIA_SERVER_FFPROBE = '__kodumedia_missing_ffprobe__'
    $env:TMDB_TOKEN = ''
    $env:IMDB_RATINGS_SOURCE = $ratingsPath
    $env:IMDB_RATINGS_PATH = ''

    $server = Start-Process `
        -FilePath $executable `
        -WorkingDirectory $root `
        -RedirectStandardOutput $stdout `
        -RedirectStandardError $stderr `
        -PassThru `
        -NoNewWindow

    $baseUrl = "http://127.0.0.1:$port"
    $health = Wait-ForServer -BaseUrl $baseUrl -Process $server -StdoutPath $stdout -StderrPath $stderr

    $expectedVersion = ''
    $versionFile = Join-Path $root 'VERSION'
    if (Test-Path -LiteralPath $versionFile -PathType Leaf) {
        $expectedVersion = (Get-Content -LiteralPath $versionFile -Raw).Trim()
    }
    if (-not [string]::IsNullOrWhiteSpace($expectedVersion)) {
        Assert-Condition -Condition ($health.version -eq $expectedVersion) -Message "Ootasin serveri versiooni $expectedVersion, sain $($health.version)."
    }

    $updateStatus = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/update"
    Assert-Condition -Condition ($updateStatus.currentVersion -eq $health.version) -Message 'Uuenduse olek ei tagastanud praegust versiooni.'
    Assert-Condition -Condition ($updateStatus.localOnly) -Message 'Uuenduse endpoint ei ole märgitud localhost-piiranguga.'

    $librariesPayload = Invoke-JsonRequest -Uri "$baseUrl/api/v1/libraries"
    $libraries = @($librariesPayload.items)
    foreach ($systemLibraryId in @('library-movies', 'library-series', 'library-music')) {
        Assert-Condition `
            -Condition (@($libraries | Where-Object { $_.id -eq $systemLibraryId }).Count -eq 1) `
            -Message "Põhikogu $systemLibraryId puudub."
    }
    $moviesLibrary = @($libraries | Where-Object { $_.id -eq 'library-movies' })[0]
    Assert-Condition -Condition ($null -ne $moviesLibrary.sources) -Message 'Legacy sources:null jäi API vastuses parandamata.'
    Assert-Condition -Condition (@($moviesLibrary.sources).Count -eq 0) -Message 'Tühjal Filmide kogul on ootamatud allikad.'

    $configuration = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/settings"
    Assert-Condition -Condition ($null -ne $configuration.settings) -Message 'Runtime-seadete vastus puudub.'
    Assert-Condition -Condition ($null -ne $configuration.status.ffmpeg) -Message 'FFmpeg olek puudub.'
    Assert-Condition -Condition ($null -ne $configuration.status.ffprobe) -Message 'FFprobe olek puudub.'
    Assert-Condition -Condition ($null -ne $configuration.status.metadata.tmdb) -Message 'TMDb olek puudub.'
    Assert-Condition -Condition ($null -ne $configuration.status.metadata.imdb) -Message 'IMDb olek puudub.'
    Assert-Condition -Condition ($null -ne $configuration.settings.subtitles) -Message 'Subtiitrite seaded puuduvad.'
    Assert-Condition -Condition (@($configuration.settings.subtitles.preferredLanguages).Count -ge 1) -Message 'Eelistatud subtiitrikeeled puuduvad.'
    Assert-Condition -Condition (-not ($configuration.settings.subtitles.PSObject.Properties.Name -contains 'openSubtitlesApiKey')) -Message 'API lekitas OpenSubtitles API võtme.'
    Assert-Condition -Condition (-not ($configuration.settings.subtitles.PSObject.Properties.Name -contains 'openSubtitlesPassword')) -Message 'API lekitas OpenSubtitles parooli.'
    Assert-Condition `
        -Condition (-not ($configuration.settings.metadata.PSObject.Properties.Name -contains 'tmdbToken')) `
        -Message 'API lekitas salvestatud TMDb tokeni välja.'
    Assert-Condition -Condition (-not $configuration.settings.metadata.tmdbTokenConfigured) -Message 'TMDb token märgiti seadistatuks ilma tokenita.'
    Assert-Condition -Condition ($configuration.settings.metadata.imdbRatingsSource -eq $ratingsPath) -Message 'IMDb source bootstrap-väärtus ei jõudnud avalikku settings vastusesse.'
    Assert-Condition -Condition (-not ($configuration.settings.metadata.PSObject.Properties.Name -contains 'imdbRatingsPath')) -Message 'API tagastas legacy imdbRatingsPath välja.'
    Assert-Condition -Condition (-not ($configuration.settings.metadata.PSObject.Properties.Name -contains 'autoRefreshOnScan')) -Message 'API tagastas eemaldatud autoRefreshOnScan välja.'

    $settingsPayload = @{
        ffmpegPath = '__kodumedia_missing_ffmpeg__'
        ffprobePath = '__kodumedia_missing_ffprobe__'
        metadata = @{
            language = 'et-EE'
            imdbRatingsSource = $ratingsPath
        }
        subtitles = @{
            preferredLanguages = @('et', 'en')
        }
    }
    $testedConfiguration = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/settings/test" -Method POST -Body $settingsPayload
    Assert-Condition -Condition (-not $testedConfiguration.status.ffmpeg.available) -Message 'Puuduv FFmpeg käsk märgiti kättesaadavaks.'
    Assert-Condition -Condition (-not $testedConfiguration.status.ffprobe.available) -Message 'Puuduv FFprobe käsk märgiti kättesaadavaks.'

    $savedConfiguration = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/settings" -Method PUT -Body $settingsPayload
    Assert-Condition -Condition ($savedConfiguration.settings.ffmpegPath -eq '__kodumedia_missing_ffmpeg__') -Message 'FFmpeg runtime-seade ei salvestunud.'
    Assert-Condition -Condition ($savedConfiguration.settings.ffprobePath -eq '__kodumedia_missing_ffprobe__') -Message 'FFprobe runtime-seade ei salvestunud.'
    Assert-Condition -Condition (Test-Path -LiteralPath (Join-Path $dataDirectory 'settings.json') -PathType Leaf) -Message 'data\settings.json faili ei loodud.'
    Assert-Condition -Condition ($savedConfiguration.settings.metadata.imdbRatingsSource -eq $ratingsPath) -Message 'IMDb runtime source ei salvestunud.'

    $ratingsJob = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/metadata/imdb/refresh" -Method POST
    $completedRatingsJob = Wait-ForJob -BaseUrl $baseUrl -JobId ([string] $ratingsJob.id)
    Assert-Condition -Condition ($completedRatingsJob.progress -eq 100) -Message 'IMDb refresh töö ei lõpetanud 100% progressiga.'
    $refreshedConfiguration = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/settings"
    Assert-Condition -Condition ($refreshedConfiguration.status.metadata.imdb.available) -Message 'Kohalik IMDb andmestik ei muutunud refresh järel kättesaadavaks.'

    $validSettingsPayload = @{
        ffmpegPath = $bundledFFmpeg
        ffprobePath = $bundledFFprobe
        metadata = @{
            language = 'et-EE'
            imdbRatingsSource = $ratingsPath
        }
        subtitles = @{
            preferredLanguages = @('et', 'en')
        }
    }
    $validConfiguration = Invoke-JsonRequest -Uri "$baseUrl/api/v1/system/settings" -Method PUT -Body $validSettingsPayload
    Assert-Condition -Condition ($validConfiguration.status.ffmpeg.available) -Message 'Bundled FFmpeg ei muutunud runtime-seadetes kättesaadavaks.'
    Assert-Condition -Condition ($validConfiguration.status.ffprobe.available) -Message 'Bundled FFprobe ei muutunud runtime-seadetes kättesaadavaks.'

    $rootListing = Invoke-JsonRequest -Uri "$baseUrl/api/v1/filesystem/browse"
    Assert-Condition -Condition ($null -ne $rootListing.roots) -Message 'Failisirvija roots väli puudub.'
    Assert-Condition -Condition ($null -ne $rootListing.entries) -Message 'Failisirvija entries väli puudub.'
    Assert-Condition -Condition (@($rootListing.roots).Count -gt 0) -Message 'Windowsi failisirvija ei leidnud ühtegi ketast.'

    $encodedMediaDirectory = [System.Uri]::EscapeDataString($mediaDirectory)
    $mediaListing = Invoke-JsonRequest -Uri "$baseUrl/api/v1/filesystem/browse?path=$encodedMediaDirectory&files=true&extension=.mkv"
    $mediaEntries = @($mediaListing.entries)
    Assert-Condition -Condition (@($mediaEntries | Where-Object { $_.name -eq 'Dune.2021.1080p.mkv' -and -not $_.directory }).Count -eq 1) -Message 'Failisirvija ei kuvanud MKV faili.'
    Assert-Condition -Condition (@($mediaEntries | Where-Object { $_.name -eq 'Severance' -and $_.directory }).Count -eq 1) -Message 'Failisirvija ei kuvanud seriaali kausta.'

    foreach ($path in @('/', '/library/library-movies', '/settings')) {
        $response = Invoke-WebRequest -Uri ($baseUrl + $path) -Method Get -UseBasicParsing -TimeoutSec 20
        Assert-Condition `
            -Condition ($response.StatusCode -eq 200 -and $response.Content -match '<div id="root"></div>') `
            -Message "SPA fallback ebaõnnestus rajal $path."
    }

    $catalogText = Get-Content -LiteralPath (Join-Path $dataDirectory 'catalog.json') -Raw
    Assert-Condition -Condition ($catalogText -notmatch '"sources"\s*:\s*null') -Message 'Kataloogifaili legacy null väärtust ei parandatud.'

    $mixedLibrary = Invoke-JsonRequest `
        -Uri "$baseUrl/api/v1/libraries" `
        -Method POST `
        -Body @{ name = 'Smoke mixed'; kind = 'mixed' }
    $encodedLibraryId = [System.Uri]::EscapeDataString([string] $mixedLibrary.id)
    $updatedLibrary = Invoke-JsonRequest `
        -Uri "$baseUrl/api/v1/libraries/$encodedLibraryId/sources" `
        -Method POST `
        -Body @{ path = $mediaDirectory }
    Assert-Condition -Condition (@($updatedLibrary.sources).Count -eq 1) -Message 'Meediakausta lisamine API kaudu ebaõnnestus.'

    $scanJob = Invoke-JsonRequest -Uri "$baseUrl/api/v1/libraries/$encodedLibraryId/scan" -Method POST
    $completedJob = Wait-ForJob -BaseUrl $baseUrl -JobId ([string] $scanJob.id)
    Assert-Condition -Condition ($completedJob.progress -eq 100) -Message 'Skannimistöö ei lõpetanud 100% progressiga.'

    $moviesPage = Invoke-JsonRequest -Uri "$baseUrl/api/v1/media?libraryId=$encodedLibraryId&type=movie&limit=50"
    $seriesPage = Invoke-JsonRequest -Uri "$baseUrl/api/v1/media?libraryId=$encodedLibraryId&type=series&limit=50"
    $episodesPage = Invoke-JsonRequest -Uri "$baseUrl/api/v1/media?libraryId=$encodedLibraryId&type=episode&limit=50"
    $musicPage = Invoke-JsonRequest -Uri "$baseUrl/api/v1/media?libraryId=$encodedLibraryId&type=music&limit=50"
    $movies = @($moviesPage.items)
    $seriesItems = @($seriesPage.items)
    $episodes = @($episodesPage.items)
    $music = @($musicPage.items)

    Assert-Condition -Condition ($movies.Count -eq 4) -Message "Ootasin nelja filmi, sain $($movies.Count)."
    Assert-Condition -Condition (@($movies | Where-Object { $_.title -eq 'Dune' -and $_.year -eq 2021 }).Count -eq 1) -Message 'Filmi pealkirja/aasta automaattuvastus ebaõnnestus.'
    Assert-Condition -Condition (@($movies | Where-Object { $_.title -eq 'Suzume No Tojimari' }).Count -eq 1) -Message 'Fansub release-groupi eemaldamine ebaõnnestus.'
    Assert-Condition -Condition (@($movies | Where-Object { $_.title -eq 'Wolf Children' -and $_.year -eq 2012 }).Count -eq 1) -Message 'Loose release-groupi eemaldamine ebaõnnestus.'
    Assert-Condition -Condition (@($movies | Where-Object { $_.title -eq 'Lotte And The Moonstone Secret' -and $_.year -eq 2011 }).Count -eq 1) -Message 'Informatiivse parent-kausta kasutamine ebaõnnestus.'
    Assert-Condition -Condition ($seriesItems.Count -eq 1) -Message "Ootasin üht seriaali parent-kirjet, sain $($seriesItems.Count)."
    Assert-Condition -Condition ($seriesItems[0].title -eq 'Severance') -Message 'Seriaali nime tuvastus ebaõnnestus.'
    Assert-Condition -Condition ($seriesItems[0].seasonCount -eq 2) -Message 'Seriaali hooaegade arv on vale.'
    Assert-Condition -Condition ($seriesItems[0].episodeCount -eq 2) -Message 'Seriaali episoodide arv on vale.'
    Assert-Condition -Condition ($episodes.Count -eq 2) -Message "Ootasin kaht episoodi, sain $($episodes.Count)."
    Assert-Condition -Condition ($music.Count -eq 1) -Message "Legacy audio extension .ape ei klassifitseerunud muusikaks."
    Assert-Condition -Condition (@($episodes | Where-Object { $_.seriesId -ne $seriesItems[0].id }).Count -eq 0) -Message 'Kõiki episoode ei seostatud sama seriaali parent-kirjega.'
    Assert-Condition -Condition (@($episodes | Where-Object { $_.season -eq 1 -and $_.episode -eq 1 -and $_.title -eq 'Good News' }).Count -eq 1) -Message 'Season 01 kaustamustri tuvastus ebaõnnestus.'
    Assert-Condition -Condition (@($episodes | Where-Object { $_.season -eq 2 -and $_.episode -eq 3 -and $_.title -eq 'Who Is Alive' }).Count -eq 1) -Message 'S02E03 failinimemustri tuvastus ebaõnnestus.'

    $dune = @($movies | Where-Object { $_.title -eq 'Dune' })[0]
    $encodedDuneId = [System.Uri]::EscapeDataString([string] $dune.id)
    $playbackOptions = Invoke-JsonRequest -Uri "$baseUrl/api/v1/media/$encodedDuneId/playback-options"
    Assert-Condition -Condition ($null -ne $playbackOptions.audioTracks) -Message 'Playback options audioTracks puudub.'
    Assert-Condition -Condition (@($playbackOptions.subtitles).Count -eq 1) -Message 'Adjacent SRT subtiitrit ei tuvastatud.'
    Assert-Condition -Condition ($playbackOptions.subtitles[0].language -eq 'et') -Message 'Subtiitri keele tuvastus ebaõnnestus.'
    $subtitleUrl = [string] $playbackOptions.subtitles[0].url
    Assert-Condition -Condition (-not [string]::IsNullOrWhiteSpace($subtitleUrl)) -Message 'Tekstisubtiitri WebVTT URL puudub.'
    $subtitleResponse = Invoke-WebRequest -Uri "$baseUrl$subtitleUrl" -UseBasicParsing -TimeoutSec 20
    Assert-Condition -Condition ($subtitleResponse.StatusCode -eq 200) -Message 'WebVTT subtiiter ei avanenud.'
    Assert-Condition -Condition ([string] $subtitleResponse.Content -match '^\s*WEBVTT') -Message 'Subtitle endpoint ei tagastanud korrektset WebVTT faili.'
    Assert-Condition -Condition ([string] $subtitleResponse.Headers['Cache-Control'] -match 'no-store') -Message 'Subtitle endpoint lubas brauseril vana rada cache-ida.'

    $ffmpegIdsBeforePlayback = @(Get-Process -Name 'ffmpeg' -ErrorAction SilentlyContinue | ForEach-Object { $_.Id })
    $playbackRequest = @{
        mediaId = [string] $dune.id
        capabilities = @{
            clientName = 'windows-smoke'
            clientVersion = '1'
            supportedMimes = @('video/mp4')
            videoCodecs = @('h264')
            audioCodecs = @('aac')
            subtitleFormats = @('webvtt')
            hdrFormats = @()
            supportsMse = $true
            supportsHls = $true
            maximumWidth = 1920
            maximumHeight = 1080
            maximumAudioChannels = 2
        }
        preferences = @{
            forceTranscode = $true
        }
    }
    $playbackSession = Invoke-JsonRequest -Uri "$baseUrl/api/v1/playback/sessions" -Method POST -Body $playbackRequest
    Assert-Condition -Condition ($playbackSession.plan.mode -eq 'transcode') -Message 'Playback smoke ei loonud universaalset transcode-sessionit.'
    $encodedSessionId = [System.Uri]::EscapeDataString([string] $playbackSession.id)
    $sessionBaseUrl = "$baseUrl/api/v1/playback/sessions/$encodedSessionId"
    $playlistResponse = Invoke-WebRequest -Uri "$sessionBaseUrl/index.m3u8" -UseBasicParsing -TimeoutSec 20
    Assert-Condition -Condition ($playlistResponse.StatusCode -eq 200) -Message 'HLS playlist ei avanenud.'
    $playlistText = [string] $playlistResponse.Content
    $initMatch = [regex]::Match($playlistText, '#EXT-X-MAP:URI="([^"]+)"')
    Assert-Condition -Condition ($initMatch.Success) -Message 'HLS playlistis puudub fMP4 init fragment.'
    $segmentName = @($playlistText -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) -and -not $_.StartsWith('#') }) | Select-Object -First 1
    Assert-Condition -Condition (-not [string]::IsNullOrWhiteSpace($segmentName)) -Message 'HLS playlistis puudub esimene media fragment.'
    $initResponse = Invoke-WebRequest -Uri "$sessionBaseUrl/$($initMatch.Groups[1].Value)" -UseBasicParsing -TimeoutSec 20
    Assert-Condition -Condition ($initResponse.StatusCode -eq 200 -and $initResponse.RawContentLength -gt 0) -Message 'init.mp4 puudub või on tühi.'
    $segmentResponse = Invoke-WebRequest -Uri "$sessionBaseUrl/$segmentName" -UseBasicParsing -TimeoutSec 20
    Assert-Condition -Condition ($segmentResponse.StatusCode -eq 200 -and $segmentResponse.RawContentLength -gt 0) -Message 'Esimene HLS media fragment puudub või on tühi.'
    Invoke-WebRequest -Uri $sessionBaseUrl -Method DELETE -UseBasicParsing -TimeoutSec 20 | Out-Null
    Start-Sleep -Milliseconds 500
    Assert-Condition -Condition (-not (Test-Path -LiteralPath (Join-Path $dataDirectory "transcodes\$($playbackSession.id)"))) -Message 'Playback sessioni ajutine kaust jäi pärast DELETE päringut alles.'
    $newFFmpegProcesses = @(Get-Process -Name 'ffmpeg' -ErrorAction SilentlyContinue | Where-Object { $ffmpegIdsBeforePlayback -notcontains $_.Id })
    Assert-Condition -Condition ($newFFmpegProcesses.Count -eq 0) -Message 'Playback session jättis ffmpeg.exe protsessi tööle.'

    $encodedSeriesId = [System.Uri]::EscapeDataString([string] $seriesItems[0].id)
    $seriesDetails = Invoke-JsonRequest -Uri "$baseUrl/api/v1/series/$encodedSeriesId"
    $seasonGroups = @($seriesDetails.seasons)
    $episodeTotal = 0
    foreach ($season in $seasonGroups) {
        $episodeTotal += @($season.episodes).Count
    }
    Assert-Condition -Condition ($seasonGroups.Count -eq 2) -Message 'Seriaali detail API ei tagastanud kaht hooajagruppi.'
    Assert-Condition -Condition ($episodeTotal -eq 2) -Message 'Seriaali detail API kaotas episoode.'

    Write-Host 'Smoke-test läbis:' -ForegroundColor Green
    Write-Host '  - server health, versioon ja lokaalne update-status'
    Write-Host '  - kodumedia-updater.exe olemasolu'
    Write-Host '  - legacy kataloogi migratsioon ja SPA rajad'
    Write-Host '  - brauseris muudetavate seadete GET/test/PUT leping'
    Write-Host '  - kiire Windowsi fallback-failisirvija'
    Write-Host '  - IMDb source schema ja käsitsi refresh-job'
    Write-Host '  - filmi/seriaali automaattuvastus ja release-name puhastamine'
    Write-Host '  - legacy audio extension ning playback-options/subtitle discovery'
    Write-Host '  - päris FFmpeg fMP4 HLS: playlist, init.mp4, esimene segment ja protsessi lõpetamine'
    Write-Host '  - seriaal -> hooaeg -> episood rühmitamine'
}
finally {
    if ($null -ne $server) {
        $server.Refresh()
        if (-not $server.HasExited) {
            Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue
            $server.WaitForExit()
        }
    }

    foreach ($name in $environmentNames) {
        [Environment]::SetEnvironmentVariable($name, $previousEnvironment[$name], 'Process')
    }

    if (Test-Path -LiteralPath $work) {
        Remove-Item -LiteralPath $work -Recurse -Force
    }
}
