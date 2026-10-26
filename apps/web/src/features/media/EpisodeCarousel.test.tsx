import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MediaItem } from '../../api/types'
import { EpisodeCarousel } from './EpisodeCarousel'

const carouselMock = vi.hoisted(() => {
  const api = {
    canScrollNext: vi.fn(() => true),
    canScrollPrev: vi.fn(() => false),
    off: vi.fn(),
    on: vi.fn(),
    reInit: vi.fn(),
    scrollNext: vi.fn(),
    scrollPrev: vi.fn(),
    scrollProgress: vi.fn(() => 0),
    scrollSnapList: vi.fn(() => [0, 1]),
    scrollTo: vi.fn(),
  }
  return { api, hook: vi.fn() }
})

vi.mock('embla-carousel-react', () => ({ default: carouselMock.hook }))

function episode(id: string, number: number, title: string): MediaItem {
  return {
    id,
    libraryId: 'library-series',
    seriesId: 'series-1',
    type: 'episode',
    title,
    sortTitle: `1-${number}`,
    seriesTitle: 'Näidissari',
    season: 1,
    episode: number,
    episodeTitle: title,
    file: { relativePath: `Näidissari\\Season 1\\Show.S01E0${number}.mkv`, name: `Show.S01E0${number}.mkv`, extension: '.mkv', sizeBytes: 1024, modifiedAt: '2026-01-01T00:00:00Z' },
    technical: { durationMs: 3_600_000 },
    externalIds: {},
    metadata: { summary: `${title} kirjeldus.` },
    addedAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  }
}

describe('EpisodeCarousel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    carouselMock.hook.mockReturnValue([vi.fn(), carouselMock.api])
  })

  it('renders episodes horizontally and exposes Embla navigation to pointer and keyboard users', async () => {
    const user = userEvent.setup()
    const first = episode('episode-1', 1, 'Piloot')
    const second = episode('episode-2', 2, 'Teine osa')
    const onOpenEpisode = vi.fn()

    const { container } = render(
      <EpisodeCarousel episodes={[first, second]} label="Hooaeg 1" onOpenEpisode={onOpenEpisode} />,
    )

    expect(screen.getByRole('region', { name: 'Hooaeg 1' })).toBeTruthy()
    expect(container.querySelectorAll('.episode-carousel__slide')).toHaveLength(2)
    expect(container.querySelector('.episode-carousel__track')).toBeTruthy()
    expect(carouselMock.hook).toHaveBeenCalledWith(expect.objectContaining({
      dragFree: false,
      breakpoints: {
        '(hover: hover) and (pointer: fine)': {
          dragFree: true,
          skipSnaps: true,
        },
      },
    }))

    await user.click(screen.getByRole('button', { name: 'Järgmine osa' }))
    expect(carouselMock.api.scrollNext).toHaveBeenCalled()

    const firstCard = screen.getByRole('button', { name: 'S01E01 Piloot' })
    const secondCard = screen.getByRole('button', { name: 'S01E02 Teine osa' })
    firstCard.focus()
    fireEvent.keyDown(firstCard, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(secondCard)
    expect(carouselMock.api.scrollTo).toHaveBeenCalledWith(1)

    fireEvent.keyDown(secondCard, { key: 'Home' })
    expect(document.activeElement).toBe(firstCard)
    expect(carouselMock.api.scrollTo).toHaveBeenCalledWith(0)

    await user.click(firstCard)
    expect(onOpenEpisode).toHaveBeenCalledWith(first)
  })
})
