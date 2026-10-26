import type { Library, LibrarySource, MediaItem, MediaPage, SeasonGroup, SeriesDetails } from './types'

export type LibraryPayload = Omit<Library, 'sources'> & {
  sources?: LibrarySource[] | null
}

export type MediaPagePayload = Omit<MediaPage, 'items'> & {
  items?: MediaItem[] | null
}

export function normalizeLibrary(payload: LibraryPayload): Library {
  return {
    ...payload,
    sources: Array.isArray(payload.sources) ? payload.sources : [],
  }
}

export function normalizeLibraryList(payloads: LibraryPayload[] | null | undefined): Library[] {
  if (!Array.isArray(payloads)) return []
  return payloads.map(normalizeLibrary)
}

export function normalizeMediaPage(payload: MediaPagePayload): MediaPage {
  return {
    ...payload,
    items: Array.isArray(payload.items) ? payload.items : [],
  }
}

export type SeasonGroupPayload = Omit<SeasonGroup, 'episodes'> & {
  episodes?: MediaItem[] | null
}

export type SeriesDetailsPayload = Omit<SeriesDetails, 'seasons' | 'availability'> & {
  availability?: SeriesDetails['availability']
  seasons?: SeasonGroupPayload[] | null
}

export function normalizeSeriesDetails(payload: SeriesDetailsPayload): SeriesDetails {
  return {
    ...payload,
    availability: payload.availability ?? {
      releasedSeasonCount: payload.series?.metadata?.officialSeasonCount ?? payload.series?.seasonCount ?? 0,
      releasedEpisodeCount: payload.series?.metadata?.officialEpisodeCount ?? payload.series?.episodeCount ?? 0,
      availableSeasonCount: payload.series?.seasonCount ?? 0,
      availableEpisodeCount: payload.series?.episodeCount ?? 0,
      newEpisodeAvailable: false,
    },
    seasons: Array.isArray(payload.seasons)
      ? payload.seasons.map((season) => ({
          ...season,
          episodes: Array.isArray(season.episodes) ? season.episodes : [],
        }))
      : [],
  }
}
