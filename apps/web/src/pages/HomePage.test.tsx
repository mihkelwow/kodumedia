import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { MediaItem } from '../api/types'
import { HomePage, resetHomeCacheForTests } from './HomePage'

afterEach(() => {
  vi.unstubAllGlobals()
  resetHomeCacheForTests()
})

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('HomePage watch refresh', () => {
  it('reloads continue watching when the player reports new progress', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/api/v1/watch/continue')) return jsonResponse({ items: [] })
      return jsonResponse({ items: [], total: 0, limit: 18, offset: 0 })
    })
    vi.stubGlobal('fetch', fetchMock)

    const view = render(<HomePage onOpen={vi.fn()} onNavigate={vi.fn()} refreshToken={0} />)
    await waitFor(() => {
      expect(fetchMock.mock.calls.filter(([url]) => String(url).includes('/api/v1/watch/continue'))).toHaveLength(1)
    })

    view.rerender(<HomePage onOpen={vi.fn()} onNavigate={vi.fn()} refreshToken={1} />)
    await waitFor(() => {
      expect(fetchMock.mock.calls.filter(([url]) => String(url).includes('/api/v1/watch/continue'))).toHaveLength(2)
    })
  })
})


function featuredItem(index: number): MediaItem {
  return {
    id: `featured-${index}`,
    libraryId: 'movies',
    type: index % 2 ? 'movie' : 'series',
    title: `Featured ${index}`,
    sortTitle: `Featured ${index}`,
    year: 2020 + index,
    releaseDate: `202${index}-01-01`,
    file: { relativePath: `featured-${index}.mkv`, name: `featured-${index}.mkv`, extension: '.mkv', sizeBytes: 1, modifiedAt: '2026-07-23T00:00:00Z' },
    technical: {},
    externalIds: {},
    metadata: {
      summary: `Summary ${index}`,
      backdropUrl: `/backdrop-${index}.jpg`,
      logoUrl: `/logo-${index}.png`,
      trailerYoutubeKey: `trailer-${index}`,
      imdbRating: 9 - index / 10,
    },
    addedAt: '2026-07-23T00:00:00Z',
    updatedAt: '2026-07-23T00:00:00Z',
  }
}

describe('HomePage featured carousel', () => {
  it('shows at most ten eligible items and navigates without leaving the page', async () => {
    const items = Array.from({ length: 12 }, (_, index) => featuredItem(index + 1))
    vi.stubGlobal('matchMedia', vi.fn((query: string) => ({
      matches: query.includes('prefers-reduced-motion'),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })))
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).includes('/api/v1/watch/continue')) return jsonResponse({ items: [] })
      return jsonResponse({ items, total: items.length, limit: 18, offset: 0 })
    }))

    render(<HomePage onOpen={vi.fn()} onNavigate={vi.fn()} />)

    const next = await screen.findByLabelText('Järgmine esiletõstetud sisu')
    expect(screen.getAllByRole('tab')).toHaveLength(10)
    const firstLabel = screen.getByRole('region').getAttribute('aria-label')
    fireEvent.click(next)
    await waitFor(() => expect(screen.getByRole('region').getAttribute('aria-label')).not.toBe(firstLabel))
    expect(screen.getByText('2 / 10')).toBeTruthy()
  })

  it('does not include items without a trailer in the hero carousel', async () => {
    const withTrailer = featuredItem(1)
    const withoutTrailer = {
      ...featuredItem(2),
      id: 'without-trailer',
      title: 'Without trailer',
      metadata: {
        ...featuredItem(2).metadata,
        trailerYoutubeKey: undefined,
        trailerCandidates: [],
        imdbRating: 10,
      },
    }
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })))
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).includes('/api/v1/watch/continue')) return jsonResponse({ items: [] })
      return jsonResponse({ items: [withoutTrailer, withTrailer], total: 2, limit: 18, offset: 0 })
    }))

    render(<HomePage onOpen={vi.fn()} onNavigate={vi.fn()} />)

    expect(await screen.findByRole('region', { name: /Esiletõstetud: Featured 1/ })).toBeTruthy()
    expect(screen.queryByRole('region', { name: /Without trailer/ })).toBeNull()
    expect(screen.queryByRole('tab')).toBeNull()
  })
})

describe('HomePage genre rows', () => {
  it('uses genre order as category weight, keeps animation exclusive, and avoids duplicates', async () => {
    const base = featuredItem(1)
    const continueItem: MediaItem = {
      ...featuredItem(2),
      id: 'continue-item',
      title: 'Continue item',
      watch: {
        mediaId: 'continue-item',
        mediaType: 'movie',
        positionMs: 1_000,
        durationMs: 10_000,
        progress: 0.1,
        completed: false,
        lastWatchedAt: '2026-07-24T00:00:00Z',
        updatedAt: '2026-07-24T00:00:00Z',
      },
    }
    const horror = { ...featuredItem(3), id: 'horror', title: 'Horror movie', metadata: { ...featuredItem(3).metadata, genres: ['Horror'] } }
    const action = { ...featuredItem(4), id: 'action', title: 'Action movie', metadata: { ...featuredItem(4).metadata, genres: ['Action'] } }
    const animatedAction = { ...featuredItem(5), id: 'animated-action', title: 'Animated action', metadata: { ...featuredItem(5).metadata, genres: ['Action', 'Animation'] } }
    const scienceFictionAction = { ...featuredItem(8), id: 'science-fiction-action', title: 'Science fiction first', metadata: { ...featuredItem(8).metadata, genres: ['Science Fiction', 'Action'] } }
    const comedyHorror = { ...featuredItem(6), id: 'comedy-horror', title: 'Comedy first', metadata: { ...featuredItem(6).metadata, genres: ['Comedy', 'Horror'] } }
    const horrorComedy = { ...featuredItem(7), id: 'horror-comedy', title: 'Horror first', metadata: { ...featuredItem(7).metadata, genres: ['Horror', 'Comedy'] } }

    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })))
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(String(input), 'http://localhost')
      if (url.pathname.includes('/api/v1/watch/continue')) return jsonResponse({ items: [continueItem] })
      const genre = url.searchParams.get('genre') ?? ''
      if (genre.includes('Horror')) return jsonResponse({ items: [horror, comedyHorror, horrorComedy], total: 3, limit: 100, offset: 0 })
      if (genre.includes('Comedy')) return jsonResponse({ items: [comedyHorror, horrorComedy], total: 2, limit: 100, offset: 0 })
      if (genre.includes('Action')) return jsonResponse({ items: [action, animatedAction, scienceFictionAction], total: 3, limit: 100, offset: 0 })
      if (genre.includes('Animation')) return jsonResponse({ items: [animatedAction], total: 1, limit: 100, offset: 0 })
      if (genre.includes('Science Fiction')) return jsonResponse({ items: [scienceFictionAction], total: 1, limit: 100, offset: 0 })
      if (genre) return jsonResponse({ items: [], total: 0, limit: 100, offset: 0 })
      if (url.searchParams.get('type') === 'movie' && url.searchParams.get('sort') === 'modifiedAt') {
        return jsonResponse({ items: [base], total: 1, limit: 18, offset: 0 })
      }
      return jsonResponse({ items: [], total: 0, limit: 18, offset: 0 })
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<HomePage onOpen={vi.fn()} onNavigate={vi.fn()} />)

    const featured = await screen.findByRole('region', { name: /Esiletõstetud:/ })
    const continueHeading = screen.getByRole('heading', { name: 'Jätka vaatamist' })
    expect(featured.parentElement?.firstElementChild).toBe(featured)
    expect(featured.compareDocumentPosition(continueHeading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()

    const horrorSection = screen.getByRole('heading', { name: 'Õudusfilmid' }).closest('section')
    const comedySection = screen.getByRole('heading', { name: 'Komöödiad' }).closest('section')
    const actionSection = screen.getByRole('heading', { name: 'Märul ja seiklus' }).closest('section')
    const animationSection = screen.getByRole('heading', { name: 'Animatsioon' }).closest('section')
    const scienceFictionSection = screen.getByRole('heading', { name: 'Ulmefilmid' }).closest('section')
    expect(horrorSection).toBeTruthy()
    expect(comedySection).toBeTruthy()
    expect(actionSection).toBeTruthy()
    expect(animationSection).toBeTruthy()
    expect(scienceFictionSection).toBeTruthy()

    expect(within(horrorSection!).getByRole('button', { name: 'Ava Horror movie' })).toBeTruthy()
    expect(within(horrorSection!).getByRole('button', { name: 'Ava Horror first' })).toBeTruthy()
    expect(within(horrorSection!).queryByRole('button', { name: 'Ava Comedy first' })).toBeNull()

    expect(within(comedySection!).getByRole('button', { name: 'Ava Comedy first' })).toBeTruthy()
    expect(within(comedySection!).queryByRole('button', { name: 'Ava Horror first' })).toBeNull()

    expect(within(actionSection!).getByRole('button', { name: 'Ava Action movie' })).toBeTruthy()
    expect(within(actionSection!).queryByRole('button', { name: 'Ava Animated action' })).toBeNull()
    expect(within(actionSection!).queryByRole('button', { name: 'Ava Science fiction first' })).toBeNull()
    expect(within(animationSection!).getByRole('button', { name: 'Ava Animated action' })).toBeTruthy()
    expect(within(scienceFictionSection!).getByRole('button', { name: 'Ava Science fiction first' })).toBeTruthy()

    expect(screen.queryByRole('heading', { name: 'Romantilised filmid' })).toBeNull()
    expect(screen.getAllByRole('button', { name: 'Ava Animated action' })).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: 'Ava Comedy first' })).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: 'Ava Horror first' })).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: 'Ava Science fiction first' })).toHaveLength(1)

    const requestedGenres = fetchMock.mock.calls.map(([input]) => new URL(String(input), 'http://localhost').searchParams.get('genre'))
    expect(requestedGenres).toContain('Action,Märul,Adventure,Seiklus')
    expect(requestedGenres).toContain('Science Fiction,Sci-Fi,Ulme')
  })
})


describe('HomePage perceived loading', () => {
  it('restores persisted hero content synchronously while data revalidates in the background', () => {
    const item = featuredItem(1)
    window.localStorage.setItem('kodumedia.home.v2', JSON.stringify({
      storedAt: Date.now(),
      data: {
        continueWatching: [], movies: [item], series: [], episodes: [], releases: [item], rated: [item],
        horror: [], comedy: [], actionAdventure: [], romance: [], animation: [], scienceFiction: [],
      },
    }))
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: true,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })))
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>(() => undefined)))

    render(<HomePage onOpen={vi.fn()} onNavigate={vi.fn()} />)

    expect(screen.getByRole('region', { name: /Esiletõstetud: Featured 1/ })).toBeTruthy()
    expect(screen.queryByText('Laen avalehte')).toBeNull()
  })
})
