import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MediaItem } from '../api/types'
import { ThemeProvider } from '../features/theme/ThemeProvider'
import { App } from './App'

const track: MediaItem = {
  id: 'track-1', libraryId: 'library-music', type: 'music', title: 'Persistent Song', sortTitle: 'Artist Album 0001 Persistent Song', artist: 'Artist', album: 'Album', trackNumber: 1,
  file: { relativePath: 'Artist/Album/01 - Persistent Song.wav', name: '01 - Persistent Song.wav', extension: '.wav', sizeBytes: 1024, modifiedAt: '2026-01-01T00:00:00Z' },
  technical: { audioCodec: 'pcm_s24le', audioChannels: 2, audioSampleRate: 96000, audioBitDepth: 24, durationMs: 180000 },
  externalIds: {}, metadata: { posterUrl: '/api/v1/media/track-1/artwork' }, addedAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z',
}

const libraries = { items: [
  { id: 'library-movies', name: 'Filmid', kind: 'movies', system: true, sources: [], createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
  { id: 'library-series', name: 'Seriaalid', kind: 'series', system: true, sources: [], createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
  { id: 'library-music', name: 'Muusika', kind: 'music', system: true, sources: [], createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
] }

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('global music player', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/music')
    window.localStorage.clear()
    vi.spyOn(window, 'scrollTo').mockImplementation(() => undefined)
    vi.spyOn(HTMLMediaElement.prototype, 'load').mockImplementation(() => undefined)
    vi.spyOn(HTMLMediaElement.prototype, 'play').mockImplementation(function () {
      this.dispatchEvent(new Event('play'))
      return Promise.resolve()
    })
    vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(function () {
      this.dispatchEvent(new Event('pause'))
    })
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.pathname + input.search : input.url
      if (url === '/api/v1/libraries') return jsonResponse(libraries)
      if (url.startsWith('/api/v1/music?')) return jsonResponse({ totalTracks: 1, recentlyPlayed: [track], recentlyAdded: [track], artists: [], albums: [] })
      if (url.startsWith('/api/v1/notifications')) return jsonResponse({ items: [], total: 0, unread: 0, limit: 5, offset: 0 })
      if (url.startsWith('/api/v1/watch/continue')) return jsonResponse({ items: [] })
      if (url.startsWith('/api/v1/media')) return jsonResponse({ items: [], total: 0, limit: 18, offset: 0 })
      throw new Error(`Unexpected request: ${url}`)
    }))
  })

  it('stays mounted across routes and does not restart a paused track', async () => {
    const user = userEvent.setup()
    const play = vi.mocked(HTMLMediaElement.prototype.play)
    render(<ThemeProvider><App /></ThemeProvider>)

    const trackButtons = await screen.findAllByRole('button', { name: /Persistent Song/ }, { timeout: 5000 })
    await user.click(trackButtons[0])
    await waitFor(() => expect(play).toHaveBeenCalledTimes(1))
    expect(screen.getByLabelText('Muusikapleier')).toBeTruthy()

    await user.click(await screen.findByRole('button', { name: 'Paus' }))
    await user.click(screen.getByRole('link', { name: 'Avaleht' }))

    expect(window.location.pathname).toBe('/')
    expect(screen.getByLabelText('Muusikapleier')).toBeTruthy()
    expect(play).toHaveBeenCalledTimes(1)
  })
})
