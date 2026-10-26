import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MediaItem, SeriesDetails as SeriesDetailsPayload } from '../../api/types'
import { SeriesDetails } from './SeriesDetails'

const series: MediaItem = {
  id: 'series-1',
  libraryId: 'library-series',
  type: 'series',
  title: 'Näidissari',
  sortTitle: 'Näidissari',
  seasonCount: 2,
  episodeCount: 2,
  file: { relativePath: '', name: '', extension: '', sizeBytes: 0, modifiedAt: '2026-01-01T00:00:00Z' },
  technical: {},
  externalIds: { tmdb: '100' },
  metadata: { summary: 'Seriaali kirjeldus.', metadataSource: 'tmdb', genres: ['Sci-Fi', 'Draama'], imdbVotes: 12345 },
  addedAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

function episode(id: string, season: number, number: number, title: string, filename: string): MediaItem {
  return {
    id,
    libraryId: 'library-series',
    seriesId: series.id,
    type: 'episode',
    title,
    sortTitle: `${season}-${number}`,
    seriesTitle: series.title,
    season,
    episode: number,
    episodeTitle: title,
    file: { relativePath: `Näidissari\\Season ${season}\\${filename}`, name: filename, extension: '.mkv', sizeBytes: 1024, modifiedAt: '2026-01-01T00:00:00Z' },
    technical: { durationMs: 3_600_000 },
    externalIds: {},
    metadata: {},
    addedAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  }
}

const firstEpisode = episode('episode-1', 1, 1, 'Piloot', 'Show.S01E01.Pilot.mkv')
const secondEpisode = episode('episode-2', 2, 1, 'Finaal', 'Show.S02E01.Finale.mkv')
const response: SeriesDetailsPayload = {
  series,
  playTarget: firstEpisode,
  availability: {
    releasedSeasonCount: 2,
    releasedEpisodeCount: 2,
    availableSeasonCount: 2,
    availableEpisodeCount: 2,
    latestReleasedEpisode: { season: 2, episode: 1, title: 'Finaal', releaseDate: '2026-02-01' },
    newEpisodeAvailable: true,
  },
  seasons: [
    { number: 1, title: 'Hooaeg 1', episodeCount: 1, episodes: [firstEpisode] },
    { number: 2, title: 'Hooaeg 2', episodeCount: 1, episodes: [secondEpisode] },
  ],
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('SeriesDetails', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(response)))
  })

  it('groups episodes by season and opens an episode detail view', async () => {
    const user = userEvent.setup()
    const onOpenEpisode = vi.fn()
    render(<SeriesDetails item={series} onClose={vi.fn()} onOpenEpisode={onOpenEpisode} />)

    const firstSeasonTab = await screen.findByRole('tab', { name: 'Hooaeg 1' })
    expect(firstSeasonTab.querySelector('small')).toBeNull()
    expect(document.querySelector('.season-tabs__meta')?.textContent).toBe('1 osa')
    expect(screen.queryByRole('button', { name: /metaand/i })).toBeNull()
    expect(screen.getByText('Show.S01E01.Pilot.mkv')).toBeTruthy()
    expect(document.querySelector('.episode-carousel__track')).toBeTruthy()
    expect(document.querySelector('.episode-list')).toBeNull()

    await user.click(screen.getByRole('tab', { name: /Hooaeg 2/ }))
    expect(document.querySelector('.season-tabs__meta')?.textContent).toBe('1 osa')
    expect(screen.getByText('Show.S02E01.Finale.mkv')).toBeTruthy()

    await user.click(screen.getByRole('button', { name: /Finaal/ }))
    expect(onOpenEpisode).toHaveBeenCalledWith(secondEpisode)
  })
  it('keeps the 0.6.4 metadata layout and places only the latest release on the right', async () => {
    const { container } = render(<SeriesDetails item={series} onClose={vi.fn()} onOpenEpisode={vi.fn()} />)

    const callout = await screen.findByRole('status', { name: 'Uuem osa on ilmunud' })
    expect(callout.closest('.details__metadata-column')).toBeTruthy()
    expect(within(callout).getByText('Uuem osa on ilmunud')).toBeTruthy()
    expect(within(callout).getByText(/S02E01 · Finaal/)).toBeTruthy()
    expect(screen.getByText(/Sci-Fi\s+\|\s+Draama/)).toBeTruthy()
    expect(container.querySelector('.series-meta-panel')).toBeNull()
    expect(container.querySelector('.series-meta-stack')).toBeNull()
  })


  it('uses a short play label and shows the episode code below it', async () => {
    render(<SeriesDetails item={series} onClose={vi.fn()} onOpenEpisode={vi.fn()} onPlayEpisode={vi.fn()} />)

    const play = await screen.findByRole('button', { name: 'Esita' })
    expect(play.classList.contains('media-play-button')).toBe(true)
    expect(within(play.parentElement as HTMLElement).getByText('S01E01')).toBeTruthy()
  })

  it('renders one season as quiet metadata instead of a misleading tab', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => jsonResponse({
      ...response,
      seasons: [response.seasons[1]],
    })))

    render(<SeriesDetails item={series} onClose={vi.fn()} onOpenEpisode={vi.fn()} />)

    await screen.findByText('Show.S02E01.Finale.mkv')
    expect(screen.queryByRole('tab')).toBeNull()
    expect(document.querySelector('.season-tabs-single')?.textContent).toContain('Hooaeg 2')
    expect(document.querySelector('.season-tabs-single')?.textContent).toContain('1 osa')
  })

})
