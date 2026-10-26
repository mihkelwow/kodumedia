import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './client'

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('system API client', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('uses the lightweight health endpoint for restart polling', async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => jsonResponse({ status: 'ok', version: '0.2.2' }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.system.health()).resolves.toEqual({ status: 'ok', version: '0.2.2' })
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/health', expect.objectContaining({ headers: expect.objectContaining({ Accept: 'application/json' }) }))
  })

  it('uploads a ZIP as multipart data without forcing a JSON content type', async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => jsonResponse({ state: 'installing', currentVersion: '0.2.1', targetVersion: '0.2.2', localOnly: true }))
    vi.stubGlobal('fetch', fetchMock)
    const file = new File(['package'], 'kodumedia-0.2.2-windows-x64.zip', { type: 'application/zip' })

    await api.system.installRelease(file)

    const [, init] = fetchMock.mock.calls[0]!
    expect(init?.method).toBe('POST')
    expect(init?.body).toBeInstanceOf(FormData)
    expect(init?.headers).toEqual(expect.objectContaining({ Accept: 'application/json', 'X-KoduMedia-Update': 'install' }))
    expect((init?.headers as Record<string, string>)['Content-Type']).toBeUndefined()
  })
  it('serializes multiple media genres as one OR filter', async () => {
    const fetchMock = vi.fn(async () => jsonResponse({ items: [], total: 0, limit: 18, offset: 0 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.media.list({ type: 'movie', genre: ['Action', 'Adventure'], limit: 18 })

    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/media?type=movie&genre=Action%2CAdventure&limit=18')
  })

  it('loads subtitle text without using the browser cache', async () => {
    const fetchMock = vi.fn(async () => new Response('WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nTere', {
      status: 200,
      headers: { 'Content-Type': 'text/vtt; charset=utf-8' },
    }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.playback.loadSubtitle('/api/v1/media/movie/subtitles/embedded-2')).resolves.toContain('WEBVTT')
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/media/movie/subtitles/embedded-2',
      expect.objectContaining({
        cache: 'no-store',
        credentials: 'same-origin',
        headers: expect.objectContaining({ Accept: expect.stringContaining('text/vtt') }),
      }),
    )
  })

  it('starts a local-network cast session with the selected playback preferences', async () => {
    const fetchMock = vi.fn(async () => jsonResponse({
      id: 'cast-1',
      device: { id: 'tv-1', name: 'Elutoa teler', protocol: 'dlna', canSeek: true, canSetVolume: true },
      mediaId: 'movie-1',
      sourceId: 'source-1',
      state: 'playing',
      startedAt: '2026-10-26T09:00:00Z',
      lastUpdateAt: '2026-10-26T09:00:00Z',
    }))
    vi.stubGlobal('fetch', fetchMock)

    await api.cast.start({
      deviceId: 'tv-1',
      mediaId: 'movie-1',
      title: 'Film',
      preferences: { startPositionMs: 42_000, subtitleId: 'sub-et' },
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/cast/sessions',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          deviceId: 'tv-1',
          mediaId: 'movie-1',
          title: 'Film',
          preferences: { startPositionMs: 42_000, subtitleId: 'sub-et' },
        }),
      }),
    )
  })

})
