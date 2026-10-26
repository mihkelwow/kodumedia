import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { MediaItem } from '../../api/types'
import { MusicTrackDetails } from './MusicTrackDetails'

const track: MediaItem = {
  id: 'track-1', libraryId: 'music', type: 'music', title: 'Night Drive (Bootleg)', sortTitle: 'night drive',
  artist: 'Artist', album: 'Unreleased Edits',
  musicClassification: { genres: ['Electronic'], styles: ['Progressive House', 'Bootleg'], source: 'mixed', genreSource: 'embedded', styleSource: 'musicbrainz', confidence: 'matched', sourceUrl: 'https://musicbrainz.org/recording/recording-id' },
  file: { relativePath: 'Artist/Unreleased Edits/Night Drive.flac', name: 'Night Drive.flac', extension: '.flac', sizeBytes: 42_000_000, modifiedAt: '2026-01-01T00:00:00Z' },
  technical: { container: 'flac', audioCodec: 'flac', audioSampleRate: 96000, audioBitDepth: 24, audioChannels: 2, bitrate: 2300000, durationMs: 210000 },
  externalIds: { musicBrainzRecording: 'recording-id' }, metadata: { posterUrl: '/artwork.jpg' }, addedAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z',
}

describe('MusicTrackDetails', () => {
  it('groups technical and classification fields into a compact view', () => {
    render(<MusicTrackDetails track={track} onClose={vi.fn()} />)

    expect(screen.getByText('Helikvaliteet')).toBeTruthy()
    expect(screen.getByText(/FLAC · FLAC · 96.0 kHz · 24-bit · 2 kanalit/)).toBeTruthy()
    expect(screen.getByText('Žanr ja stiil')).toBeTruthy()
    expect(screen.getByText('Electronic · Progressive House · Bootleg')).toBeTruthy()
    expect(screen.queryByText('Codec')).toBeNull()
    expect(screen.queryByText('Diskreetimissagedus')).toBeNull()
    expect(screen.getByText('Allikad: žanr – helifaili tagid; stiil – MusicBrainz')).toBeTruthy()
    expect(screen.queryByText('Liigitus')).toBeNull()
    expect(screen.getByRole('link', { name: 'Ava allikaleht' }).getAttribute('href')).toBe('https://musicbrainz.org/recording/recording-id')
    expect(screen.getByRole('img').closest('.music-track-details__art-frame')).toBeTruthy()
  })
})
