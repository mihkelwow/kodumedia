import { describe, expect, it } from 'vitest'
import { normalizeLibrary, normalizeLibraryList, normalizeMediaPage, normalizeSeriesDetails } from './normalizers'

const baseLibrary = {
  id: 'library-movies',
  name: 'Filmid',
  kind: 'movies' as const,
  system: true,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

describe('API collection normalization', () => {
  it('converts legacy null library sources into an empty array', () => {
    const library = normalizeLibrary({ ...baseLibrary, sources: null })
    expect(library.sources).toEqual([])
  })

  it('returns an empty library list for a missing collection', () => {
    expect(normalizeLibraryList(undefined)).toEqual([])
    expect(normalizeLibraryList(null)).toEqual([])
  })

  it('converts a missing media page collection into an empty array', () => {
    const page = normalizeMediaPage({ total: 0, limit: 200, offset: 0, items: null })
    expect(page.items).toEqual([])
  })

  it('normalizes missing series and episode collections', () => {
    const series = {
      id: 'series-1',
      libraryId: 'library-series',
      type: 'series' as const,
      title: 'Näidissari',
      sortTitle: 'näidissari',
      file: { relativePath: '', name: '', extension: '', sizeBytes: 0, modifiedAt: '2026-01-01T00:00:00Z' },
      technical: {},
      externalIds: {},
      metadata: {},
      addedAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    }

    expect(normalizeSeriesDetails({ series, seasons: null }).seasons).toEqual([])
    expect(normalizeSeriesDetails({
      series,
      seasons: [{ number: 1, title: 'Hooaeg 1', episodeCount: 0, episodes: null }],
    }).seasons[0]?.episodes).toEqual([])
  })
})
