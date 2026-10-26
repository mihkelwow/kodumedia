import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { MediaItem } from '../../api/types'
import { MediaDetails } from './MediaDetails'

const movie: MediaItem = {
  id: 'movie-1',
  libraryId: 'library-movies',
  type: 'movie',
  title: 'Dune: Part Two',
  sortTitle: 'Dune Part Two',
  year: 2024,
  file: {
    path: 'C:\\Filmid\\Dune.Part.Two.2024.2160p.BluRay.x265.mkv',
    relativePath: 'Dune.Part.Two.2024.2160p.BluRay.x265.mkv',
    name: 'Dune.Part.Two.2024.2160p.BluRay.x265.mkv',
    extension: '.mkv',
    sizeBytes: 1024,
    modifiedAt: '2026-01-01T00:00:00Z',
  },
  technical: { container: 'mkv', videoCodec: 'hevc', audioCodec: 'aac' },
  externalIds: { tmdb: '693134' },
  metadata: { summary: 'Ametlik kirjeldus.', metadataSource: 'tmdb' },
  addedAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

describe('MediaDetails', () => {
  it('shows the original filename in the large pre-play view', async () => {
    const user = userEvent.setup()
    const onPlay = vi.fn()
    render(<MediaDetails item={movie} onClose={vi.fn()} onPlay={onPlay} />)

    expect(screen.getByRole('heading', { name: 'Dune: Part Two' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: /metaand/i })).toBeNull()
    expect(screen.getByText('Algne failinimi')).toBeTruthy()
    const original = screen.getByLabelText('Algne meediafail')
    expect(within(original).getByText('Dune.Part.Two.2024.2160p.BluRay.x265.mkv', { selector: 'strong' })).toBeTruthy()
    expect(within(original).getByText('C:\\Filmid\\Dune.Part.Two.2024.2160p.BluRay.x265.mkv')).toBeTruthy()

    const play = screen.getByRole('button', { name: 'Esita' })
    expect(play.classList.contains('media-play-button')).toBe(true)
    await user.click(play)
    expect(onPlay).toHaveBeenCalledWith(movie)
  })
})
