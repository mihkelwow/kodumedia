import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Library, MediaItem, MusicLanding } from '../api/types'
import { MusicPage } from './MusicPage'

const library: Library = { id: 'music', name: 'Muusika', kind: 'music', system: true, sources: [], createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }
const track: MediaItem = {
  id: 'track-1', libraryId: library.id, type: 'music', title: 'Opening Track', sortTitle: 'Artist Album 0001 Opening Track', artist: 'Artist', album: 'Album', trackNumber: 1,
  file: { relativePath: 'Artist/Album/01 - Opening Track.wav', name: '01 - Opening Track.wav', extension: '.wav', sizeBytes: 1024, modifiedAt: '2026-01-01T00:00:00Z' },
  technical: { audioCodec: 'pcm_s24le', audioChannels: 2, audioSampleRate: 96000, audioBitDepth: 24, durationMs: 180000 },
  musicClassification: { genres: ['Electronic'], styles: ['Progressive House'], source: 'sidecar', confidence: 'verified' },
  externalIds: {}, metadata: { posterUrl: '/api/v1/media/track-1/artwork', genres: ['Electronic'] }, addedAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z',
}
const landing: MusicLanding = {
  totalTracks: 1, recentlyPlayed: [track], recentlyAdded: [track],
  artists: [{ id: 'artist', name: 'Artist', albumCount: 1, trackCount: 1, artworkUrl: track.metadata.posterUrl }],
  albums: [{ id: 'album', artist: 'Artist', title: 'Album', trackCount: 1, artworkUrl: track.metadata.posterUrl }],
}

function response(body: unknown): Response { return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }) }

describe('MusicPage', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn(async () => response(landing))))


  it('shows a complete track-library entry and opens the all-tracks route', async () => {
    const user = userEvent.setup()
    const onNavigate = vi.fn()
    render(<MusicPage library={library} onNavigate={onNavigate} onPlayTrack={vi.fn()} />)

    const heading = await screen.findByRole('heading', { name: 'Kõik lood' })
    expect(heading.closest('section')?.classList.contains('music-all-tracks-entry')).toBe(true)
    expect(screen.getByText('1 lugu serveris')).toBeTruthy()
    await user.click(screen.getByRole('button', { name: 'Vaata kõiki' }))
    expect(onNavigate).toHaveBeenCalledWith('/music/tracks')
  })

  it('lists every track in the dedicated all-tracks view', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => response({ items: [track], total: 1, limit: 200, offset: 0, facets: { artists: [{ value: 'Artist', count: 1 }], albums: [{ value: 'Album', count: 1 }], genres: [{ value: 'Electronic', count: 1 }], styles: [{ value: 'Progressive House', count: 1 }], years: [], codecs: [{ value: 'PCM_S24LE', count: 1 }], lossless: 1, hiRes: 1, unclassified: 0 } })))
    render(<MusicPage library={library} tracksView onNavigate={vi.fn()} onPlayTrack={vi.fn()} />)

    expect(await screen.findByRole('heading', { name: 'Kõik lood' })).toBeTruthy()
    expect(screen.getByRole('button', { name: /Opening Track/ })).toBeTruthy()
    expect(screen.getByPlaceholderText('Otsi lugusid, artiste, albumeid või stiile')).toBeTruthy()
    expect(screen.getByText('Electronic')).toBeTruthy()
    expect(screen.getByText('Progressive House')).toBeTruthy()
    expect(screen.queryByText('Kinnitatud')).toBeNull()
    expect(screen.queryByText('Faili tagidest')).toBeNull()
  })

  it('searches on every keypress without replacing the search field', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn(async () => response({ items: [track], total: 1, limit: 200, offset: 0, facets: { artists: [{ value: 'Artist', count: 1 }], albums: [{ value: 'Album', count: 1 }], genres: [{ value: 'Electronic', count: 1 }], styles: [{ value: 'Progressive House', count: 1 }], years: [], codecs: [{ value: 'PCM_S24LE', count: 1 }], lossless: 1, hiRes: 1, unclassified: 0 } }))
    vi.stubGlobal('fetch', fetchMock)
    render(<MusicPage library={library} tracksView onNavigate={vi.fn()} onPlayTrack={vi.fn()} />)

    await screen.findByRole('heading', { name: 'Kõik lood' })
    const search = screen.getByPlaceholderText('Otsi lugusid, artiste, albumeid või stiile') as HTMLInputElement
    await user.type(search, 'ab')

    await waitFor(() => {
      const urls = fetchMock.mock.calls.map(([url]) => String(url))
      expect(urls.some((url) => url.includes('search=a'))).toBe(true)
      expect(urls.some((url) => url.includes('search=ab'))).toBe(true)
    })
    expect(search.value).toBe('ab')
    expect(document.activeElement).toBe(search)
    expect(screen.queryByText('Sorteeri')).toBeNull()
    expect(screen.queryByLabelText('Žanr')).toBeNull()
  })

  it('starts a track directly from the music landing page', async () => {
    const user = userEvent.setup()
    const onPlayTrack = vi.fn()
    render(<MusicPage library={library} onNavigate={vi.fn()} onPlayTrack={onPlayTrack} />)
    const buttons = await screen.findAllByRole('button', { name: /Opening Track/ })
    await user.click(buttons[0])
    expect(onPlayTrack).toHaveBeenCalledWith(track, expect.any(Array))
    expect(screen.queryByRole('link', { name: /Opening Track/ })).toBeNull()
  })
})
