# KoduMedia music metadata and artwork

KoduMedia fills missing music classification per field. User-curated data is
never overwritten by an automatic source.

Priority:

1. file-local KoduMedia sidecar (`<audio-file>.kodumedia.json`);
2. embedded audio tags (`GENRE`, `STYLE`, `SUBGENRE`, `GROUPING`, `MOOD`);
3. a Shazam audio-fingerprint match;
4. an exact AcoustID/MusicBrainz recording and release match;
5. a high-confidence public catalog match;
6. explicit style words in the recognized title, album, filename, or folder path;
7. local audio analysis;
8. an explicit low-confidence `Other` fallback.

Genre, style, and mood store their sources separately. The source is shown only
as small secondary text under **Genre and style** in the audio-file detail
dialog. Track rows do not show source badges.

## Matching and persistent cache

KoduMedia first creates a Shazam-compatible acoustic signature from a short
16 kHz mono PCM segment. Signature generation happens locally and only the
compact signature is sent to the unofficial Shazam recognition endpoint; the
original audio file is not uploaded. This path requires no API key or paid
account. Because the endpoint is reverse-engineered rather than a documented
public API, it can change or rate-limit requests, so it remains optional and is
protected by throttling, persistent caching, and independent fallbacks.

When Shazam has no match, KoduMedia searches MusicBrainz conservatively using
title, artist, album, and duration. A free AcoustID application key enables
Chromaprint audio fingerprint matching. On Windows, the official `fpcalc` 1.6.0
binary is downloaded only when fingerprinting is first needed and its pinned
SHA-256 checksum is verified before execution.

When MusicBrainz does not find a sufficiently strong result, KoduMedia can use a
broader catalog fallback. Candidate scoring includes title, artist, version
markers such as remix/edit/bootleg, and duration. A critical version mismatch is
rejected so an unofficial remix is not silently assigned the original track's
artwork.

Cached data is stored below `data/music-metadata`:

- successful catalog matches: 180 days;
- negative results: 7 days;
- fingerprints: until source size or modification time changes;
- local audio analysis: until source size or modification time changes.

KoduMedia does not rewrite audio files. External identifiers and classification
are stored in the local catalog.

## Local audio analysis

The explicit whole-library enrichment job can decode a short mono PCM sample
with the configured FFmpeg binary. It estimates tempo, onset strength,
brightness, zero-crossing rate, and dynamics. These features provide a broad
style estimate such as House, Trance, Drum & Bass, Ambient, Rock, Acoustic,
or Pop. Tempo alone never assigns Hip-Hop; a Hip-Hop-family estimate requires
independent supporting classification context.

This result is an estimate, not a verified release fact. It is marked with the
`audio-analysis` source and an estimated or low confidence value. It is ignored
when it conflicts with a stronger Shazam, MusicBrainz, or catalog genre. Clear style
terms in the track metadata or path are preferred because they can identify
more specific styles such as Deep House, Liquid Drum & Bass, or Psytrance.

## Artwork chain

Artwork priority depends on how trustworthy the local collection context is:

1. a track-specific image beside the audio file (`<audio-stem>.jpg`, `.jpeg`, or `.png`);
2. a safe local album image (`cover`, `folder`, `front`, or `album`) only when the directory is a single-track folder or its name matches a meaningful album title;
3. for a normal well-tagged album, embedded artwork extracted from the audio file;
4. exact Cover Art Archive release or release-group artwork, then a high-confidence Shazam/catalog image;
5. for a loose multi-track folder with generic tags such as Music / 2026, the Shazam/catalog image is tried before embedded artwork because batch downloads may contain the same playlist image in every file;
6. a deterministic KoduMedia-generated SVG track card.

All cached artwork is keyed to the physical audio file. Generic or duplicate
artist/album tags therefore cannot make unrelated tracks share an embedded,
catalog, generated, or negative-cache result. Exact external IDs are included
in the key so a corrected match invalidates the old image. Generic multi-track
directories such as Music, Downloads, Singles, Remixes, or a year folder do not
share a stray `cover.jpg` across every file. A meaningful album folder and a
single-track folder retain local/embedded priority.

The generated card contains the track title, artist, and an optional
remix/edit/bootleg marker. It is intentionally neutral and is not represented as
official album artwork. The **Reprocess the entire music library** job retries
the external chain and can replace a generated card later.

## User-verified sidecar

A sidecar is treated as user-verified metadata and overrides embedded and
automatic values. Example for `Artist - Bootleg Remix.flac`:

```json
{
  "title": "Bootleg Remix",
  "artist": "DJ Example",
  "album": "Unreleased Edits",
  "year": 2026,
  "trackNumber": 4,
  "genres": ["Electronic"],
  "styles": ["Progressive House", "Bootleg"],
  "moods": ["Energetic"]
}
```

Save it as `Artist - Bootleg Remix.flac.kodumedia.json`. The extension-free form
`Artist - Bootleg Remix.kodumedia.json` is also accepted. UTF-8 JSON is required.
Rescan the Music library after editing a sidecar.
