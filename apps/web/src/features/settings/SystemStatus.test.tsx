import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Library, SystemConfiguration } from '../../api/types'
import { resetLocaleForTests } from '../../i18n/i18n'
import { ThemeProvider } from '../theme/ThemeProvider'
import { SystemStatus } from './SystemStatus'

const configuration: SystemConfiguration = {
  version: '0.2.2-test',
  settings: {
    ffmpegPath: 'ffmpeg',
    ffprobePath: 'ffprobe',
    metadata: {
      tmdbTokenConfigured: false,
      language: 'et-EE',
      imdbRatingsSource: 'https://datasets.imdbws.com/title.ratings.tsv.gz',
    },
    music: {
      metadataEnabled: true,
      shazamEnabled: true,
      fingerprintEnabled: true,
      catalogFallbackEnabled: true,
      audioAnalysisEnabled: true,
      generatedArtworkEnabled: true,
      acoustIdApiKeyConfigured: false,
    },
    subtitles: {
      preferredLanguages: ['et', 'en'],
      openSubtitlesApiKeyConfigured: false,
      openSubtitlesUsernameConfigured: false,
      openSubtitlesPasswordConfigured: false,
    },
    quality: {
      profile: 'original',
      preferredCodec: 'auto',
      hardwarePreference: 'auto',
      cachePolicy: 'watched',
      cacheMaxGb: 50,
      gracePeriodSeconds: 15,
      vmafEnabled: true,
      vmafTarget: 95,
      vmafSampleSeconds: 12,
    },
  },
  status: {
    ffmpeg: { available: true, version: 'ffmpeg version test', message: 'FFmpeg töötab.' },
    ffprobe: { available: true, version: 'ffprobe version test', message: 'FFprobe töötab.' },
    metadata: {
      tmdb: { available: false, message: 'TMDb token puudub.' },
      imdb: { available: false, refreshing: false, message: 'IMDb andmestikku pole veel laaditud.' },
    },
  },
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('SystemStatus', () => {
  beforeEach(() => {
    resetLocaleForTests('et-EE')
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.pathname : input.url
      if (url === '/api/v1/system/settings' && (!init?.method || init.method === 'GET')) return jsonResponse(configuration)
      if (url === '/api/v1/system/update' && (!init?.method || init.method === 'GET')) return jsonResponse({ state: 'idle', currentVersion: '0.2.2-test', localOnly: true })
      if (url === '/api/v1/system/storage' && (!init?.method || init.method === 'GET')) return jsonResponse({
        dataDirectory: 'C:\\KoduMedia\\data',
        dataSizeBytes: 0,
        diskFreeBytes: 100 * 1024 * 1024 * 1024,
        diskTotalBytes: 500 * 1024 * 1024 * 1024,
        requiredReserveBytes: 50 * 1024 * 1024 * 1024,
        categories: [],
        generatedAt: '2026-07-25T20:00:00Z',
      })
      if (url === '/api/v1/system/settings/test' && init?.method === 'POST') return jsonResponse(configuration)
      if (url === '/api/v1/system/settings' && init?.method === 'PUT') {
        return jsonResponse({
          ...configuration,
          settings: {
            ...configuration.settings,
            ffmpegPath: 'C:\\Tools\\ffmpeg.exe',
            metadata: { ...configuration.settings.metadata, tmdbTokenConfigured: true, tmdbTokenHint: '••••oken' },
          },
          status: {
            ...configuration.status,
            metadata: { ...configuration.status.metadata, tmdb: { available: true, message: 'TMDb töötab.' } },
          },
        })
      }
      throw new Error(`Unexpected request: ${url} ${init?.method || 'GET'}`)
    }))
  })

  it('tests and saves runtime settings without exposing a stored token', async () => {
    const user = userEvent.setup()
    render(<ThemeProvider><SystemStatus /></ThemeProvider>)

    const ffmpeg = await screen.findByLabelText('FFmpeg käivitusfail')
    expect(document.querySelectorAll('fieldset.settings-form-section')).toHaveLength(0)
    expect(document.querySelectorAll('section.settings-form-section')).toHaveLength(8)
    expect(document.querySelectorAll('.settings-form-section__heading')).toHaveLength(8)
    for (const label of [/Vaikimisi kvaliteediprofiil/, /Eelistatud väljundcodec/, /Encoder'i eelistus/, /Optimeeritud segmentide säilitamine/]) {
      expect(screen.getByLabelText(label, { selector: 'select' }).closest('.select-control')).toBeTruthy()
    }
    const imdbSource = screen.getByLabelText('IMDb reitinguandmestiku URL või fail') as HTMLInputElement
    expect(imdbSource.value).toBe('https://datasets.imdbws.com/title.ratings.tsv.gz')
    expect(screen.queryByRole('checkbox', { name: /automaat/i })).toBeNull()
    expect(screen.getByText('Muusika metaandmed')).toBeTruthy()
    expect(document.querySelectorAll('.instruction-card')).toHaveLength(4)
    expect(document.querySelectorAll('.settings-callout')).toHaveLength(0)
    expect(screen.getByRole('checkbox', { name: /Otsi puuduvaid muusika metaandmeid/ })).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: /esimese valikuna tasuta Shazami/ })).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: /Shazami järel AcoustID audio fingerprinti/ })).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: /laiemat kataloogivastet/ })).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: /helist lokaalne hinnang/ })).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: /neutraalset loo fallback-kaarti/ })).toBeTruthy()
    await user.clear(ffmpeg)
    await user.type(ffmpeg, 'C:\\Tools\\ffmpeg.exe')
    await user.type(screen.getByLabelText('TMDb API Read Access Token'), 'new-secret-token')

    await user.click(screen.getByRole('button', { name: 'Kontrolli väärtusi' }))
    await screen.findByText(/neid pole veel salvestatud/i)

    const saveBar = screen.getByRole('region', { name: 'Salvestamata muudatused' })
    await user.click(within(saveBar).getByRole('button', { name: 'Salvesta seaded' }))
    await screen.findByText(/puuduvad filmi- ja seriaalipildid/i)

    const fetchMock = vi.mocked(fetch)
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(5))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/system/storage', expect.anything())
    const putCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT')
    expect(putCall).toBeTruthy()
    const body = JSON.parse(String(putCall?.[1]?.body)) as { ffmpegPath: string; metadata: { tmdbToken?: string }; music: { metadataEnabled: boolean; shazamEnabled: boolean; fingerprintEnabled: boolean; catalogFallbackEnabled: boolean; audioAnalysisEnabled: boolean; generatedArtworkEnabled: boolean }; subtitles: { preferredLanguages: string[] }; quality: { profile: string; cacheMaxGb: number; cacheFreeReserveGb: number; cacheFreeReservePercent: number } }
    expect(body.ffmpegPath).toBe('C:\\Tools\\ffmpeg.exe')
    expect(body.metadata.tmdbToken).toBe('new-secret-token')
    expect(body.music.metadataEnabled).toBe(true)
    expect(body.music.shazamEnabled).toBe(true)
    expect(body.music.fingerprintEnabled).toBe(true)
    expect(body.music.catalogFallbackEnabled).toBe(true)
    expect(body.music.audioAnalysisEnabled).toBe(true)
    expect(body.music.generatedArtworkEnabled).toBe(true)
    expect(body.subtitles.preferredLanguages).toEqual(['et', 'en'])
    expect(body.quality.profile).toBe('original')
    expect(body.quality.cacheMaxGb).toBe(50)
    expect(body.quality.cacheFreeReserveGb).toBe(10)
    expect(body.quality.cacheFreeReservePercent).toBe(10)
    expect(screen.queryByDisplayValue('new-secret-token')).toBeNull()
    expect(screen.getByPlaceholderText(/Salvestatud ••••oken/)).toBeTruthy()
  }, 15_000)

  it('shows the shared save bar only for unsaved changes and can restore the saved values', async () => {
    const user = userEvent.setup()
    render(<ThemeProvider><SystemStatus /></ThemeProvider>)

    const quality = await waitFor(() => {
      const select = document.querySelector<HTMLSelectElement>('#system-quality-profile')
      expect(select).toBeTruthy()
      return select as HTMLSelectElement
    })
    expect(screen.queryByRole('region', { name: 'Salvestamata muudatused' })).toBeNull()

    await user.selectOptions(quality, 'balanced')
    const saveBar = screen.getByRole('region', { name: 'Salvestamata muudatused' })
    expect(within(saveBar).getByRole('button', { name: 'Loobu muudatustest' })).toBeTruthy()
    expect(within(saveBar).getByRole('button', { name: 'Kontrolli väärtusi' })).toBeTruthy()
    expect(within(saveBar).getByRole('button', { name: 'Salvesta seaded' })).toBeTruthy()

    await user.click(within(saveBar).getByRole('button', { name: 'Loobu muudatustest' }))
    expect(quality.value).toBe('original')
    expect(screen.queryByRole('region', { name: 'Salvestamata muudatused' })).toBeNull()
    expect(screen.getByText('Salvestamata muudatused tühistati.')).toBeTruthy()
  })

  it('opens the initial setup guide automatically when no media source is configured', async () => {
    render(<ThemeProvider><SystemStatus libraries={[]} /></ThemeProvider>)

    const toggle = await screen.findByRole('button', { name: /Esmane seadistus/ })
    await waitFor(() => expect(toggle.getAttribute('aria-expanded')).toBe('true'))
    expect(document.querySelector('#initial-setup-guide')).toBeTruthy()
  })

  it('opens the initial setup guide, links to setup sections, and runs save and scan actions', async () => {
    const user = userEvent.setup()
    const libraries: Library[] = [
      { id: 'movies', name: 'Filmid', kind: 'movies', system: true, sources: [{ id: 'source-1', path: 'D:\\Filmid', createdAt: '2026-07-24T00:00:00Z' }], createdAt: '2026-07-24T00:00:00Z', updatedAt: '2026-07-24T00:00:00Z' },
      { id: 'music', name: 'Muusika', kind: 'music', system: true, sources: [{ id: 'source-2', path: 'D:\\Muusika', createdAt: '2026-07-24T00:00:00Z' }], createdAt: '2026-07-24T00:00:00Z', updatedAt: '2026-07-24T00:00:00Z' },
    ]
    const onLibrariesChanged = vi.fn(async () => {})
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.pathname : input.url
      if (url === '/api/v1/system/settings' && (!init?.method || init.method === 'GET')) return jsonResponse(configuration)
      if (url === '/api/v1/system/update' && (!init?.method || init.method === 'GET')) return jsonResponse({ state: 'idle', currentVersion: '0.2.2-test', localOnly: true })
      if (url === '/api/v1/system/settings' && init?.method === 'PUT') return jsonResponse(configuration)
      if ((url === '/api/v1/libraries/movies/scan' || url === '/api/v1/libraries/music/scan') && init?.method === 'POST') {
        return jsonResponse({ id: `job-${url.includes('movies') ? 'movies' : 'music'}`, type: 'scan', status: 'queued', progress: 0, message: '' })
      }
      throw new Error(`Unexpected request: ${url} ${init?.method || 'GET'}`)
    })

    render(<ThemeProvider><SystemStatus libraries={libraries} onLibrariesChanged={onLibrariesChanged} /></ThemeProvider>)

    const toggle = await screen.findByRole('button', { name: /Esmane seadistus/ })
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    await user.click(toggle)

    expect(toggle.getAttribute('aria-expanded')).toBe('true')
    expect(screen.getByText(/Kõik vajalikud API-võtmed on tasuta/)).toBeTruthy()
    expect(screen.getByRole('link', { name: /Seadista TMDb/ }).getAttribute('href')).toBe('#settings-metadata-provider')
    expect(screen.getByRole('link', { name: /Seadista subtiitrid/ }).getAttribute('href')).toBe('#settings-subtitles-provider')
    expect(screen.getByRole('link', { name: /Lisa meediakaustad/ }).getAttribute('href')).toBe('#settings-libraries')
    expect(screen.getByRole('link', { name: /Muusika metaandmed/ }).getAttribute('href')).toBe('#settings-music-metadata')
    expect(screen.getByText(/system tray menüüst/)).toBeTruthy()

    const guide = document.querySelector('#initial-setup-guide') as HTMLElement
    await user.click(within(guide).getByRole('button', { name: 'Salvesta seaded' }))
    await screen.findByText(/Seaded salvestati/)

    await user.click(within(guide).getByRole('button', { name: 'Skanni kõik meediakogud' }))
    await screen.findByText('Kõigi meediakogude skannimine käivitati.')
    expect(onLibrariesChanged).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/libraries/movies/scan', expect.objectContaining({ method: 'POST' }))
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/libraries/music/scan', expect.objectContaining({ method: 'POST' }))
  })

  it('renders the complete initial setup guide in English', async () => {
    resetLocaleForTests('en-US')
    const user = userEvent.setup()
    render(<ThemeProvider><SystemStatus libraries={[{ id: 'movies', name: 'Movies', kind: 'movies', system: true, sources: [{ id: 'source-1', path: 'D:\\Movies', createdAt: '2026-07-24T00:00:00Z' }], createdAt: '2026-07-24T00:00:00Z', updatedAt: '2026-07-24T00:00:00Z' }]} /></ThemeProvider>)

    const toggle = await screen.findByRole('button', { name: /Initial setup/ })
    await user.click(toggle)

    expect(screen.getByText(/All required API keys are available for free/)).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Set up TMDb' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Set up subtitles' })).toBeTruthy()
    const guide = document.querySelector('#initial-setup-guide') as HTMLElement
    expect(within(guide).getByRole('button', { name: 'Save settings' })).toBeTruthy()
    expect(within(guide).getByRole('button', { name: 'Scan all media libraries' })).toBeTruthy()
    expect(screen.getByText(/system tray menu on the server PC/)).toBeTruthy()
  })
})
