import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from './App'
import { ThemeProvider } from '../features/theme/ThemeProvider'

const librariesPayload = {
  items: [
    {
      id: 'library-movies',
      name: 'Filmid',
      kind: 'movies',
      system: true,
      sources: null,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    },
    {
      id: 'library-series',
      name: 'Seriaalid',
      kind: 'series',
      system: true,
      sources: null,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    },
    {
      id: 'library-music',
      name: 'Muusika',
      kind: 'music',
      system: true,
      sources: null,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    },
  ],
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('App navigation regression', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/')
    window.localStorage.clear()
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.pathname + input.search : input.url
      if (url === '/api/v1/libraries') return jsonResponse(librariesPayload)
      if (url.startsWith('/api/v1/media')) return jsonResponse({ items: [], total: 0, limit: 200, offset: 0 })
      if (url === '/api/v1/system/settings') {
        return jsonResponse({
          version: '0.2.1-test',
          settings: {
            ffmpegPath: 'ffmpeg',
            ffprobePath: 'ffprobe',
            metadata: { tmdbTokenConfigured: false, language: 'et-EE', imdbRatingsSource: 'https://datasets.imdbws.com/title.ratings.tsv.gz' },
            music: { metadataEnabled: true, fingerprintEnabled: true, catalogFallbackEnabled: true, audioAnalysisEnabled: true, generatedArtworkEnabled: true, acoustIdApiKeyConfigured: false },
            subtitles: { preferredLanguages: ['et', 'en'], openSubtitlesApiKeyConfigured: false, openSubtitlesUsernameConfigured: false, openSubtitlesPasswordConfigured: false },
            quality: { profile: 'original', preferredCodec: 'auto', hardwarePreference: 'auto', cachePolicy: 'watched', cacheMaxGb: 50, gracePeriodSeconds: 15, vmafEnabled: true, vmafTarget: 95, vmafSampleSeconds: 12 },
          },
          status: { ffmpeg: { available: true }, ffprobe: { available: true }, metadata: { tmdb: { available: false }, imdb: { available: false } } },
        })
      }
      throw new Error(`Unexpected request: ${url}`)
    }))
    vi.spyOn(window, 'scrollTo').mockImplementation(() => undefined)
  })

  it('opens library and settings routes from a legacy null-sources response and returns on popstate', async () => {
    const user = userEvent.setup()
    render(
      <ThemeProvider>
        <App />
      </ThemeProvider>,
    )

    await screen.findByRole('heading', { name: 'Avaleht' })

    await user.click(screen.getByRole('link', { name: 'Filmid' }))
    expect(window.location.pathname).toBe('/library/library-movies')
    await screen.findByRole('heading', { name: 'Filmid' })
    expect(screen.getByRole('heading', { name: 'Sellel kogul pole veel meediakausta' })).toBeTruthy()

    act(() => {
      window.history.replaceState({}, '', '/')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })
    await waitFor(() => expect(screen.getByRole('heading', { name: 'Avaleht' })).toBeTruthy())

    await user.click(screen.getByRole('button', { name: 'Lisa meediakaust' }))
    expect(window.location.pathname).toBe('/settings')
    await screen.findByRole('heading', { name: 'Seaded' })
    expect(screen.getByRole('heading', { name: 'Meediakogud ja kaustad' })).toBeTruthy()
    expect(screen.getAllByText('Selle koguga pole veel ühtegi kausta seotud.')).toHaveLength(3)
    expect(screen.queryByText('Vaadet ei saanud avada')).toBeNull()
  })
})
