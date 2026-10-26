# KoduMedia first-time user guide

This guide is for people installing the ready-made Windows release. No command
line or programming experience is required.

## Before you start

You need a Windows 10/11 x64 computer that will remain the KoduMedia server,
media folders available to that computer, a modern browser and a trusted home
network. Optional free TMDb, AcoustID and OpenSubtitles.com credentials enable
additional metadata and subtitle features.

## Install and start

1. Open the repository's **Releases** page.
2. Download `kodumedia-<version>-windows-x64.zip` from the latest release.
   Do not download GitHub's automatically generated Source code ZIP for normal
   use.
3. Right-click the downloaded ZIP and select **Extract All**.
4. Extract it to a writable folder. Do not run KoduMedia from inside the ZIP.
5. Run `start-kodumedia.cmd`.
6. If Windows Defender Firewall asks, allow **Private networks** only.
7. Open <http://localhost:8096> on the server computer.

## Complete first-time setup

Open **Settings → KoduMedia readiness → First-time setup**.

1. Add a free TMDb API Read Access Token under Metadata. Use the long Bearer
   token, not the short v3 API key.
2. Optionally configure AcoustID and OpenSubtitles.com.
3. Add movie, TV-series and music folders under Media libraries and folders.
4. Save settings.
5. Select **Scan media libraries**.

The system-tray KoduMedia menu can also open the interface, scan every library
and enable startup with Windows.

## Open KoduMedia on another device

Keep both devices on the same trusted network. Find the server computer's IPv4
address in Windows **Settings → Network & Internet → Wi-Fi/Ethernet → Hardware
properties**. Then open this address on the other device:

```text
http://SERVER-IP:8096
```

For example: `http://192.168.1.50:8096`.

## Cast to a television

Open a movie or episode, start playback and select the device button in the
player.

- AirPlay uses native HLS on supported Apple devices.
- Samsung and other DLNA televisions may take 12–20 seconds to start while the
  server creates a ready buffer.
- Google Cast browser sending requires a secure HTTPS context.

The server, phone or browser, and television must be on the same local network.

## Update KoduMedia

Download the complete new Windows ZIP from Releases. On the server computer,
open **Settings → System status → KoduMedia update** and select the ZIP without
extracting it. User settings and media data under `data` are preserved.

## Common problems

- **Another device cannot connect:** verify the current server IPv4 address,
  same-network connection and Private-network firewall permission.
- **Library is empty:** verify folder permissions and scan the libraries.
- **Posters are missing:** check the TMDb API Read Access Token and internet
  access.
- **TV connects without video:** allow the device on the TV and wait at least
  20 seconds for DLNA startup.
- **Playback buffers:** prefer Ethernet or strong 5 GHz Wi-Fi and try a lower
  quality profile when real-time transcoding is required.

## Security

Do not expose port `8096` to the public internet. KoduMedia does not yet provide
user accounts or TLS. Runtime credentials and settings stay in the local
`data` directory; never publish that directory or a real `.env` file.
