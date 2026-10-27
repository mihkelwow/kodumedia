# KoduMedia

[Eesti keeles](README.et.md) · [Getting started](docs/GETTING-STARTED.md) · [Changelog](CHANGELOG.md) · [Security](SECURITY.md) · [Contributing](CONTRIBUTING.md)

![KoduMedia logo](apps/web/public/kodu-media-logo.png)

[![CI](https://github.com/mihkelwow/kodumedia/actions/workflows/ci.yml/badge.svg)](https://github.com/mihkelwow/kodumedia/actions/workflows/ci.yml)
[![License: GPL-3.0-only](https://img.shields.io/badge/license-GPL--3.0--only-6f42c1.svg)](LICENSE)
[![Platform: Windows x64](https://img.shields.io/badge/platform-Windows%20x64-0078d4.svg)](#requirements)

KoduMedia is a self-hosted media server for Windows. It combines a Go backend
with a responsive React web interface for browsing and playing movies, TV
series and music on a trusted home network.

The current release is **0.8.58**.

[![KoduMedia home screen showing the featured title and Continue Watching](docs/screenshots/home-overview.png)](docs/screenshots/home-overview.png)

<p align="center"><em>Your movies, series and music in one cinematic home interface.</em></p>

> [!WARNING]
> KoduMedia does not yet provide user accounts, TLS or secure internet-facing
> remote access. Keep port `8096` restricted to trusted private networks.

## Highlights

- Movies, TV series and music in one responsive browser interface.
- Direct Play, remuxing and HLS transcoding selected from actual media and
  client capabilities.
- HDR detection, hardware-encoder validation and controlled SDR fallback.
- Local and embedded subtitles plus optional OpenSubtitles.com integration.
- TMDb metadata and artwork, IMDb ratings and music metadata enrichment.
- AirPlay, browser Remote Playback, Google Cast and server-side DLNA/UPnP.
- Watch progress, continue watching and episode-aware post-play suggestions.
- Estonian and English user-interface translations.
- Portable Windows releases with FFmpeg and FFprobe included.

## Screenshots

<table>
  <tr>
    <td width="50%">
      <a href="docs/screenshots/series-details.png"><img src="docs/screenshots/series-details.png" alt="KoduMedia series details page with seasons and episodes"></a><br>
      <strong>Series and episodes</strong><br>
      Browse title information, seasons and available episodes from one screen.
    </td>
    <td width="50%">
      <a href="docs/screenshots/media-details.png"><img src="docs/screenshots/media-details.png" alt="KoduMedia media details page with cast and technical information"></a><br>
      <strong>Rich media details</strong><br>
      Review cast, codec, resolution, audio and file information before playback.
    </td>
  </tr>
  <tr>
    <td width="50%">
      <a href="docs/screenshots/player-playback.png"><img src="docs/screenshots/player-playback.png" alt="KoduMedia browser player during movie playback"></a><br>
      <strong>Cinematic browser playback</strong><br>
      Play movies and episodes with seeking, subtitles, volume, casting and fullscreen controls.
    </td>
    <td width="50%">
      <a href="docs/screenshots/player-diagnostics.png"><img src="docs/screenshots/player-diagnostics.png" alt="KoduMedia player with playback diagnostics expanded"></a><br>
      <strong>Playback diagnostics</strong><br>
      Inspect the active profile, codecs, buffer, encoder speed, bitrate and cache state.
    </td>
  </tr>
</table>

## Requirements

### Running a release

- Windows 10 or Windows 11, x64.
- A modern browser on the same trusted local network.
- Optional free API credentials for TMDb, AcoustID and OpenSubtitles.com.

The table below is a practical baseline for one active video stream. Direct
Play is lightweight; live transcoding, HDR tone mapping, subtitle burn-in and
multiple simultaneous streams require more processing capacity.

| Component | Practical minimum | Recommended |
| --- | --- | --- |
| CPU | 64-bit dual-core / four-thread processor for library use, music and Direct Play | Modern six-core or better; eight cores or better for regular 4K/HDR transcoding |
| Memory | 4 GB RAM for basic Direct Play use | 16 GB RAM; 32 GB for frequent 4K/HDR work or several clients |
| GPU | Not required for Direct Play or remuxing | A working H.264/HEVC hardware encoder exposed through NVIDIA NVENC, Intel Quick Sync or AMD AMF |
| Storage | 2 GB free for the portable application and metadata, excluding media and transcode cache | SSD storage and at least 60 GB free when using the default 50 GB optimized-media cache |
| Network | Stable 100 Mbit/s local network for normal 1080p use | Wired gigabit Ethernet for high-bitrate 4K, large source files and television casting |

These figures are not a guarantee for every file. Codec, bit depth, frame rate,
subtitle processing, television firmware, browser support and network peaks all
affect the actual requirement. Each additional simultaneous transcode needs its
own encoder and network headroom.

KoduMedia never ships with a user's API credentials. Credentials entered in
Settings are stored under the local `data` directory, which is excluded from
Git.

### Building from source

- Go 1.23 or newer.
- Node.js 22 LTS or another version accepted by `apps/web/package.json`.
- npm 10.8.2 or newer.
- PowerShell 5.1 or newer.
- Windows 10/11 x64 for the official executable.
- NSIS 3.08 or newer only when building the optional installer.

## Playback and encoding: good to know

KoduMedia chooses the least destructive playback path supported by both the
source file and the receiving client:

1. **Direct Play** sends the original file without changing its container,
   video or audio. For video this is available for compatible MP4 and WebM
   files; compatible MP4, WebM, MP3, FLAC, Ogg and WAV audio can also play
   directly. The client must support the container, codecs, channel count,
   resolution, bitrate and HDR format, and the selected subtitles must not
   require burn-in.
2. **Remux** keeps compatible video and audio bitstreams unchanged but places
   them into an HLS container that the browser or television can consume. A
   compatible MKV video will commonly use this path. Selecting a different
   embedded audio track may also require remuxing even when that track itself
   can be copied unchanged.
3. **Transcode** converts only the incompatible track where possible. Video is
   encoded when its codec or dimensions exceed client limits, HDR must be tone
   mapped to SDR, image subtitles must be burned into the picture, or a
   recovery profile is required. Audio can still be copied while video is
   transcoded if the client supports it.

### Audio passthrough

Yes. KoduMedia uses FFmpeg stream copy for the selected audio track when the
client reports support for its codec and channel count. Audio passthrough can
therefore remain active during Direct Play, remuxing and a video-only
transcode. This preserves the original codec, channel layout and bitrate.

If the receiving device does not support the selected audio format or channel
count, KoduMedia converts it to AAC, 192 kbit/s, stereo, 48 kHz. A deliberately
forced universal transcode also uses this compatibility path. Actual
passthrough still depends on the browser, television, receiver and connection
advertising the capability correctly.

### Hardware encoder selection

In automatic mode KoduMedia validates a real encoder at runtime and tries the
available families in this order for the selected output codec:

1. NVIDIA NVENC;
2. Intel Quick Sync Video (QSV);
3. AMD Advanced Media Framework (AMF);
4. software encoding on the CPU.

This preference applies to every supported NVIDIA GPU, not specifically to an
RTX 3090, but merely having an NVIDIA card is not sufficient: the card, driver
and bundled FFmpeg build must expose a working encoder for the requested codec.
If a hardware encoder fails its probe or cannot start for the current file,
KoduMedia tries the next compatible path.

The current NVIDIA path can pair NVENC with CUDA hardware decoding for
supported H.264, HEVC and AV1 sources. QSV and AMF encoding are supported, but
their input decoding may still run on the CPU in 0.8.58. This can make the AMD
or Intel route more CPU-intensive without intentionally lowering its selected
quality. Subtitle burn-in can also require a software-decoding/filter path.

The **Original** quality profile means “preserve the source whenever the client
can play it”; it is not a promise of bit-for-bit output for an incompatible
file. Open the player's diagnostics panel to see the actual video and audio
actions, encoder, decoder, output codec, buffer and bitrate selected for the
current session.

## Install and run

1. Open [Releases](https://github.com/mihkelwow/kodumedia/releases).
2. Download `kodumedia-<version>-windows-x64.zip` from the newest release.
3. Extract the complete ZIP into a writable folder.
4. Run `start-kodumedia.cmd`.
5. Open `http://localhost:8096` on the server computer.
6. From another device on the same network, open
   `http://SERVER-LAN-IP:8096`.

When Windows Defender Firewall asks for access, allow **Private networks** only.

Complete first-time setup in **Settings**:

1. Add movie, series and music folders.
2. Add a free TMDb API Read Access Token for metadata.
3. Optionally configure AcoustID and OpenSubtitles.com.
4. Save settings and scan the media libraries.

For a screen-by-screen explanation, including local-network access, casting,
updating and common problems, read the
[first-time user guide](docs/GETTING-STARTED.md).

## Casting and external playback

The player exposes the methods available for the current browser and network:

- **AirPlay** uses native HLS on supported Apple devices.
- **Google Cast** requires a secure browser context such as HTTPS.
- **DLNA/UPnP** discovery and control are handled by the KoduMedia server.

KoduMedia uses MPEG-TS HLS and a deliberate startup buffer for Samsung
and other DLNA televisions. A DLNA session may therefore take 12–20 seconds to
start before steady playback begins.

Device and codec support still varies by television model, firmware, browser,
network quality and source media.

## Configuration and secrets

Copy `.env.example` to `.env` only if environment-based configuration is
needed. Leave `.env.example` values empty.

```powershell
Copy-Item .env.example .env
```

Never commit any of the following:

- `.env` or another local environment file;
- the `data` directory or exported settings;
- API keys, access tokens, account passwords or cookies;
- personal media, subtitles or metadata caches;
- built executables, FFmpeg binaries or release ZIP files.

The repository's `.gitignore` excludes these paths. GitHub secret scanning is
an additional safety net, not a replacement for checking changes before every
commit.

## Build and test

From the repository root on Windows:

```powershell
.\test.ps1
.\build.ps1
```

`test.ps1` installs the locked web dependencies, validates the public npm
registry and UI translations, runs frontend tests, builds the web application,
syncs it into the Go embed directory, and runs `go test` plus `go vet`.

`build.ps1` creates Windows binaries under `dist\windows-x64`.

To avoid installing npm dependencies again:

```powershell
.\test.ps1 -SkipNpmInstall
.\build.ps1 -SkipNpmInstall
```

## Repository structure

```text
apps/server/                    Go server and Windows executables
apps/server/internal/domain     Core domain model
apps/server/internal/ports      Adapter contracts
apps/server/internal/service    Application services and playback planning
apps/server/internal/adapters   Media, metadata, subtitles, casting and storage
apps/server/internal/transport  HTTP API
apps/web/                       React and TypeScript web client
api/openapi.yaml                HTTP API description
docs/                           Architecture, security and roadmap documents
scripts/windows/                Shared PowerShell build helpers
tools/ffmpeg/                   FFmpeg licensing and build provenance
```

See [Architecture](docs/ARCHITECTURE.md) and the
[OpenAPI description](api/openapi.yaml) for implementation details.

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) before
opening a pull request. Every user-facing UI string must have both Estonian and
English translations, and all applicable checks must pass.

Please report security issues according to [SECURITY.md](SECURITY.md), not in a
public issue containing exploit details or credentials.

## License

KoduMedia is licensed under the
[GNU General Public License v3.0 only](LICENSE).

Third-party components retain their own licenses. See
[THIRD-PARTY-NOTICES.txt](THIRD-PARTY-NOTICES.txt). The official Windows release
includes a GPL build of FFmpeg; its license and source-build provenance are
included in `tools/ffmpeg` and in every release package.

### Development disclosure

KoduMedia is an experimental learning project created through prompt-driven,
AI-assisted development. The project owner defined the requirements, made the
product decisions, tested the results and directed each iteration, but did not
manually write any of the source code. AI assistance does not change the
project's GPL-3.0-only license or the need to review and test contributions.
