import { tr, trExternal } from '../i18n/i18n'
import { normalizeLibrary, normalizeLibraryList, normalizeMediaPage, normalizeSeriesDetails, type LibraryPayload, type MediaPagePayload, type SeriesDetailsPayload } from './normalizers'
import type {
  ClientCapabilities,
  CastCommand,
  CastDevice,
  CastSession,
  CastSource,
  DirectoryListing,
  FilePickerRequest,
  FilePickerResult,
  HealthStatus,
  Job,
  Library,
  LibraryKind,
  MediaItem,
  MediaQuery,
  MusicAlbumDetails,
  MusicArtistDetails,
  MusicLanding,
  MusicTrackPage,
  MusicTrackQuery,
  NotificationPage,
  PlaybackOptions,
  PlaybackPlan,
  PlaybackPreferences,
  PlaybackRuntimeStats,
  PlaybackSession,
  PostPlay,
  SubtitleSearchResult,
  SubtitleTrack,
  SystemConfiguration,
  SystemInfo,
  SystemSettingsUpdate,
  ThemePlaybackConfiguration,
  UpdateStatus,
  StorageCleanupResult,
  StorageReport,
  WatchState,
} from './types'

interface ErrorPayload {
  error?: {
    code?: string
    message?: string
  }
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

async function requestText(path: string, init?: RequestInit): Promise<string> {
  const response = await fetch(path, {
    ...init,
    cache: 'no-store',
    credentials: 'same-origin',
    headers: {
      Accept: 'text/vtt, text/plain;q=0.9, */*;q=0.1',
      ...init?.headers,
    },
  })
  if (!response.ok) {
    let message = tr('Päring ebaõnnestus ({{status}}).', { status: response.status })
    try {
      const payload = (await response.json()) as ErrorPayload
      message = trExternal(payload?.error?.message) || message
    } catch {
      // Subtitle conversion errors may be plain text or have no response body.
    }
    throw new ApiError(response.status, 'subtitle_load_failed', message)
  }
  return response.text()
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = typeof FormData !== 'undefined' && init?.body instanceof FormData
  const response = await fetch(path, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body && !isFormData ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  if (!response.ok) {
    let payload: ErrorPayload | undefined
    try {
      payload = (await response.json()) as ErrorPayload
    } catch {
      payload = undefined
    }
    throw new ApiError(
      response.status,
      payload?.error?.code ?? 'request_failed',
      trExternal(payload?.error?.message) || tr('Päring ebaõnnestus ({{status}}).', { status: response.status }),
    )
  }

  if (response.status === 204) {
    return undefined as T
  }
  return (await response.json()) as T
}

function musicTrackQueryString(query: MusicTrackQuery): string {
  const values = new URLSearchParams()
  values.set('libraryId', query.libraryId)
  if (query.search) values.set('search', query.search)
  if (query.artist) values.set('artist', query.artist)
  if (query.album) values.set('album', query.album)
  if (query.genre) values.set('genre', query.genre)
  if (query.style) values.set('style', query.style)
  if (query.year) values.set('year', String(query.year))
  if (query.codec) values.set('codec', query.codec)
  if (query.quality) values.set('quality', query.quality)
  if (query.sort) values.set('sort', query.sort)
  if (query.order) values.set('order', query.order)
  if (query.limit) values.set('limit', String(query.limit))
  if (query.offset) values.set('offset', String(query.offset))
  return `?${values.toString()}`
}

function queryString(query: MediaQuery): string {
  const values = new URLSearchParams()
  if (query.libraryId) values.set('libraryId', query.libraryId)
  if (query.seriesId) values.set('seriesId', query.seriesId)
  if (query.season) values.set('season', String(query.season))
  if (query.type) values.set('type', Array.isArray(query.type) ? query.type.join(',') : query.type)
  if (query.search) values.set('search', query.search)
  if (query.genre) values.set('genre', Array.isArray(query.genre) ? query.genre.join(',') : query.genre)
  if (query.yearFrom) values.set('yearFrom', String(query.yearFrom))
  if (query.yearTo) values.set('yearTo', String(query.yearTo))
  if (query.sort) values.set('sort', query.sort)
  if (query.order) values.set('order', query.order)
  if (query.limit) values.set('limit', String(query.limit))
  if (query.offset) values.set('offset', String(query.offset))
  const encoded = values.toString()
  return encoded ? `?${encoded}` : ''
}

export const api = {
  system: {
    health: () => request<HealthStatus>('/api/v1/health'),
    info: () => request<SystemInfo>('/api/v1/system/info'),
    configuration: () => request<SystemConfiguration>('/api/v1/system/settings'),
    test: (settings: SystemSettingsUpdate) =>
      request<SystemConfiguration>('/api/v1/system/settings/test', { method: 'POST', body: JSON.stringify(settings) }),
    update: (settings: SystemSettingsUpdate) =>
      request<SystemConfiguration>('/api/v1/system/settings', { method: 'PUT', body: JSON.stringify(settings) }),
    refreshIMDb: () =>
      request<Job>('/api/v1/system/metadata/imdb/refresh', { method: 'POST' }),
    updateStatus: () => request<UpdateStatus>('/api/v1/system/update'),
    storage: () => request<StorageReport>('/api/v1/system/storage'),
    manageStorage: (action: 'cleanup' | 'clear-optimized') =>
      request<StorageCleanupResult>('/api/v1/system/storage', {
        method: 'POST',
        body: JSON.stringify({ action }),
        headers: { 'X-KoduMedia-Storage': 'manage' },
      }),
    installRelease: (file: File) => {
      const form = new FormData()
      form.append('package', file, file.name)
      return request<UpdateStatus>('/api/v1/system/update', {
        method: 'POST',
        body: form,
        headers: { 'X-KoduMedia-Update': 'install' },
      })
    },
  },
  filesystem: {
    browse: (path = '', options: { includeFiles?: boolean; extensions?: string[]; signal?: AbortSignal } = {}) => {
      const values = new URLSearchParams()
      if (path) values.set('path', path)
      if (options.includeFiles) values.set('files', 'true')
      if (options.extensions?.length) values.set('extensions', options.extensions.join(','))
      const encoded = values.toString()
      return request<DirectoryListing>(`/api/v1/filesystem/browse${encoded ? `?${encoded}` : ''}`, { signal: options.signal })
    },
    pick: (input: FilePickerRequest) =>
      request<FilePickerResult | undefined>('/api/v1/filesystem/pick', { method: 'POST', body: JSON.stringify(input) }),
  },
  libraries: {
    list: async () => {
      const response = await request<{ items?: LibraryPayload[] | null }>('/api/v1/libraries')
      return normalizeLibraryList(response.items)
    },
    create: (name: string, kind: LibraryKind) =>
      request<LibraryPayload>('/api/v1/libraries', {
        method: 'POST',
        body: JSON.stringify({ name, kind }),
      }).then(normalizeLibrary),
    update: (id: string, input: { name?: string; kind?: LibraryKind }) =>
      request<LibraryPayload>(`/api/v1/libraries/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }).then(normalizeLibrary),
    remove: (id: string) =>
      request<void>(`/api/v1/libraries/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    addSource: (id: string, path: string) =>
      request<LibraryPayload>(`/api/v1/libraries/${encodeURIComponent(id)}/sources`, {
        method: 'POST',
        body: JSON.stringify({ path }),
      }).then(normalizeLibrary),
    removeSource: (libraryId: string, sourceId: string) =>
      request<LibraryPayload>(
        `/api/v1/libraries/${encodeURIComponent(libraryId)}/sources/${encodeURIComponent(sourceId)}`,
        { method: 'DELETE' },
      ).then(normalizeLibrary),
    scan: (id: string) =>
      request<Job>(`/api/v1/libraries/${encodeURIComponent(id)}/scan`, { method: 'POST' }),
  },
  jobs: {
    get: (id: string) => request<Job>(`/api/v1/jobs/${encodeURIComponent(id)}`),
  },
  media: {
    list: (query: MediaQuery = {}, signal?: AbortSignal) =>
      request<MediaPagePayload>(`/api/v1/media${queryString(query)}`, { signal }).then(normalizeMediaPage),
    get: (id: string) => request<MediaItem>(`/api/v1/media/${encodeURIComponent(id)}`),
    postPlay: (id: string) =>
      request<PostPlay>(`/api/v1/media/${encodeURIComponent(id)}/post-play`),
    refreshMetadata: (id: string) =>
      request<MediaItem>(`/api/v1/media/${encodeURIComponent(id)}/metadata/refresh`, {
        method: 'POST',
      }),
  },
  series: {
    get: (id: string) => request<SeriesDetailsPayload>(`/api/v1/series/${encodeURIComponent(id)}`).then(normalizeSeriesDetails),
  },
  themes: {
    settings: () => request<ThemePlaybackConfiguration>('/api/v1/themes/settings'),
    seriesUrl: (seriesId: string) => `/api/v1/series/${encodeURIComponent(seriesId)}/theme`,
  },
  music: {
    landing: (libraryId: string) =>
      request<MusicLanding>(`/api/v1/music?libraryId=${encodeURIComponent(libraryId)}`),
    artist: (libraryId: string, artist: string) =>
      request<MusicArtistDetails>(`/api/v1/music/artists/${encodeURIComponent(artist)}?libraryId=${encodeURIComponent(libraryId)}`),
    album: (libraryId: string, albumId: string) =>
      request<MusicAlbumDetails>(`/api/v1/music/albums/${encodeURIComponent(albumId)}?libraryId=${encodeURIComponent(libraryId)}`),
    tracks: (query: MusicTrackQuery, signal?: AbortSignal) =>
      request<MusicTrackPage>(`/api/v1/music/tracks${musicTrackQueryString(query)}`, { signal }),
    enrich: (libraryId = '') =>
      request<Job>('/api/v1/music/enrich', { method: 'POST', body: JSON.stringify({ libraryId }) }),
    directUrl: (mediaId: string) => `/api/v1/media/${encodeURIComponent(mediaId)}/stream`,
    losslessUrl: (mediaId: string) => `/api/v1/media/${encodeURIComponent(mediaId)}/audio/lossless`,
  },
  notifications: {
    list: (limit = 5, offset = 0) =>
      request<NotificationPage>(`/api/v1/notifications?limit=${encodeURIComponent(String(limit))}&offset=${encodeURIComponent(String(offset))}`),
    markAllRead: () =>
      request<void>('/api/v1/notifications/read', { method: 'POST' }),
  },

  watch: {
    continueWatching: (limit = 5) =>
      request<{ items?: MediaItem[] | null }>(`/api/v1/watch/continue?limit=${encodeURIComponent(String(limit))}`)
        .then((response) => Array.isArray(response.items) ? response.items : []),
    update: (mediaId: string, input: { positionMs: number; durationMs: number; completed?: boolean }, keepalive = false) =>
      request<WatchState>(`/api/v1/media/${encodeURIComponent(mediaId)}/watch`, {
        method: 'PUT',
        body: JSON.stringify(input),
        keepalive,
      }),
  },

  playback: {
    options: (mediaId: string) =>
      request<PlaybackOptions>(`/api/v1/media/${encodeURIComponent(mediaId)}/playback-options`),
    plan: (mediaId: string, capabilities: ClientCapabilities, preferences: PlaybackPreferences = {}) =>
      request<PlaybackPlan>('/api/v1/playback/plan', {
        method: 'POST',
        body: JSON.stringify({ mediaId, capabilities, preferences }),
      }),
    start: (mediaId: string, capabilities: ClientCapabilities, preferences: PlaybackPreferences = {}) =>
      request<PlaybackSession>('/api/v1/playback/sessions', {
        method: 'POST',
        body: JSON.stringify({ mediaId, capabilities, preferences }),
      }),
    stats: (sessionId: string, signal?: AbortSignal) =>
      request<PlaybackRuntimeStats>(`/api/v1/playback/sessions/${encodeURIComponent(sessionId)}/stats`, { signal }),
    stop: (sessionId: string) =>
      request<void>(`/api/v1/playback/sessions/${encodeURIComponent(sessionId)}`, { method: 'DELETE', keepalive: true }),
    searchSubtitles: (mediaId: string, languages: string[] = []) => {
      const values = new URLSearchParams()
      if (languages.length) values.set('languages', languages.join(','))
      return request<{ items: SubtitleSearchResult[] }>(
        `/api/v1/media/${encodeURIComponent(mediaId)}/subtitles/search${values.size ? `?${values.toString()}` : ''}`,
      ).then((response) => response.items ?? [])
    },
    downloadSubtitle: (mediaId: string, result: SubtitleSearchResult) =>
      request<SubtitleTrack>(`/api/v1/media/${encodeURIComponent(mediaId)}/subtitles/download`, {
        method: 'POST',
        body: JSON.stringify({ fileId: result.fileId, fileName: result.fileName, language: result.language }),
      }),
    loadSubtitle: (url: string, signal?: AbortSignal) => requestText(url, { signal }),
  },
  cast: {
    devices: (signal?: AbortSignal) =>
      request<{ items: CastDevice[] }>('/api/v1/cast/devices', { signal }).then((response) => response.items ?? []),
    active: (mediaId: string) =>
      request<CastSession | undefined>(`/api/v1/cast/sessions/active?mediaId=${encodeURIComponent(mediaId)}`),
    start: (input: { deviceId: string; mediaId: string; title: string; subtitle?: string; preferences: PlaybackPreferences }) =>
      request<CastSession>('/api/v1/cast/sessions', { method: 'POST', body: JSON.stringify(input) }),
    status: (sessionId: string, signal?: AbortSignal) =>
      request<CastSession>(`/api/v1/cast/sessions/${encodeURIComponent(sessionId)}`, { signal }),
    command: (sessionId: string, command: CastCommand) =>
      request<CastSession>(`/api/v1/cast/sessions/${encodeURIComponent(sessionId)}/commands`, {
        method: 'POST',
        body: JSON.stringify(command),
      }),
    stop: (sessionId: string) =>
      request<void>(`/api/v1/cast/sessions/${encodeURIComponent(sessionId)}`, { method: 'DELETE', keepalive: true }),
    source: (mediaId: string, preferences: PlaybackPreferences) =>
      request<CastSource>('/api/v1/cast/sources', {
        method: 'POST',
        body: JSON.stringify({ mediaId, preferences }),
      }),
    stopSource: (sourceId: string) =>
      request<void>(`/api/v1/cast/sources/${encodeURIComponent(sourceId)}`, { method: 'DELETE', keepalive: true }),
  },
}
