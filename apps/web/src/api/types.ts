export type LibraryKind = 'movies' | 'series' | 'music' | 'mixed'

export interface LibrarySource {
  id: string
  path: string
  createdAt: string
}

export interface Library {
  id: string
  name: string
  kind: LibraryKind
  system: boolean
  sources: LibrarySource[]
  createdAt: string
  updatedAt: string
}

export type MediaType = 'movie' | 'series' | 'episode' | 'music' | 'unknown'

export interface MediaTrack {
  index: number
  streamOrdinal?: number
  kind: string
  codec: string
  codecProfile?: string
  codecLevel?: number
  frameRate?: number
  bitrate?: number
  language?: string
  title?: string
  channels?: number
  channelLayout?: string
  sampleRate?: number
  width?: number
  height?: number
  bitDepth?: number
  pixelFormat?: string
  colorPrimaries?: string
  colorTransfer?: string
  colorSpace?: string
  hdrFormat?: string
  default?: boolean
  forced?: boolean
  hearingImpaired?: boolean
}


export interface EpisodeReleaseInfo {
  season: number
  episode: number
  title?: string
  releaseDate?: string
}

export interface WatchState {
  mediaId: string
  seriesId?: string
  mediaType: MediaType
  season?: number
  episode?: number
  positionMs: number
  durationMs: number
  progress: number
  completed: boolean
  lastWatchedAt: string
  updatedAt: string
}

export interface PersonCredit {
  name: string
  character?: string
  role?: string
  imageUrl?: string
}

export interface TrailerCandidate {
  youtubeKey: string
  name?: string
}

export interface MediaItem {
  id: string
  libraryId: string
  seriesId?: string
  type: MediaType
  title: string
  sortTitle: string
  year?: number
  releaseDate?: string
  season?: number
  episode?: number
  episodeTitle?: string
  seriesTitle?: string
  seasonCount?: number
  episodeCount?: number
  artist?: string
  album?: string
  trackNumber?: number
  discNumber?: number
  musicClassification?: {
    genres?: string[]
    styles?: string[]
    moods?: string[]
    source?: 'embedded' | 'sidecar' | 'musicbrainz' | 'mixed' | string
    genreSource?: 'embedded' | 'sidecar' | 'musicbrainz' | string
    styleSource?: 'embedded' | 'sidecar' | 'musicbrainz' | string
    moodSource?: 'embedded' | 'sidecar' | 'musicbrainz' | string
    confidence?: 'tagged' | 'verified' | 'matched' | 'fingerprint' | string
    sourceUrl?: string
  }
  file: {
    path?: string
    relativePath: string
    name: string
    extension: string
    sizeBytes: number
    modifiedAt: string
  }
  technical: {
    probeVersion?: number
    container?: string
    durationMs?: number
    bitrate?: number
    width?: number
    height?: number
    videoCodec?: string
    videoProfile?: string
    videoLevel?: number
    frameRate?: number
    audioCodec?: string
    audioChannels?: number
    audioSampleRate?: number
    audioBitDepth?: number
    bitDepth?: number
    pixelFormat?: string
    colorPrimaries?: string
    colorTransfer?: string
    colorSpace?: string
    hdrFormat?: string
    tracks?: MediaTrack[]
  }
  externalIds: {
    imdb?: string
    tmdb?: string
    tvdb?: string
    soundcloud?: string
    musicBrainzRecording?: string
    musicBrainzRelease?: string
    musicBrainzReleaseGroup?: string
    acoustId?: string
  }
  metadata: {
    summary?: string
    tagline?: string
    genres?: string[]
    cast?: PersonCredit[]
    directors?: PersonCredit[]
    posterUrl?: string
    backdropUrl?: string
    logoUrl?: string
    trailerYoutubeKey?: string
    trailerName?: string
    trailerCandidates?: TrailerCandidate[]
    heroAssetsVersion?: number
    imdbRating?: number
    imdbVotes?: number
    metadataSource?: string
    officialSeasonCount?: number
    officialEpisodeCount?: number
    latestEpisode?: EpisodeReleaseInfo
    updatedAt?: string
  }
  addedAt: string
  updatedAt: string
  watch?: WatchState
}



export interface MusicArtist {
  id: string
  name: string
  albumCount: number
  trackCount: number
  artworkUrl?: string
}

export interface MusicAlbum {
  id: string
  artist: string
  title: string
  year?: number
  trackCount: number
  artworkUrl?: string
  tracks?: MediaItem[]
}

export interface MusicLanding {
  totalTracks: number
  recentlyPlayed: MediaItem[]
  recentlyAdded: MediaItem[]
  artists: MusicArtist[]
  albums: MusicAlbum[]
}

export interface MusicArtistDetails {
  artist: MusicArtist
  albums: MusicAlbum[]
}

export interface MusicAlbumDetails {
  album: MusicAlbum
}

export interface MusicFacet {
  value: string
  count: number
}

export interface MusicTrackFacets {
  artists: MusicFacet[]
  albums: MusicFacet[]
  genres: MusicFacet[]
  styles: MusicFacet[]
  years: MusicFacet[]
  codecs: MusicFacet[]
  lossless: number
  hiRes: number
  unclassified: number
}

export interface MusicTrackQuery {
  libraryId: string
  search?: string
  artist?: string
  album?: string
  genre?: string
  style?: string
  year?: number
  codec?: string
  quality?: 'lossless' | 'hires' | 'unclassified' | ''
  sort?: 'title' | 'artist' | 'album' | 'year' | 'addedAt' | 'duration' | 'quality' | 'lastPlayed'
  order?: 'asc' | 'desc'
  limit?: number
  offset?: number
}

export interface MusicTrackPage {
  items: MediaItem[]
  total: number
  limit: number
  offset: number
  facets: MusicTrackFacets
}

export type NotificationKind = 'series_episode_released'

export interface NotificationItem {
  id: string
  kind: NotificationKind
  seriesId: string
  seriesTitle: string
  season: number
  episode: number
  episodeTitle?: string
  releaseDate?: string
  createdAt: string
  readAt?: string
  resolvedAt?: string
}

export interface NotificationPage {
  items: NotificationItem[]
  total: number
  unread: number
  limit: number
  offset: number
}

export interface PostPlayRecommendation {
  item: MediaItem
  playTarget?: MediaItem
  sharedGenres?: string[]
  reason?: string
}

export interface PostPlay {
  nextEpisode?: MediaItem
  recommendations: PostPlayRecommendation[]
}

export interface MediaPage {
  items: MediaItem[]
  total: number
  limit: number
  offset: number
}

export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed'

export interface Job {
  id: string
  type: string
  status: JobStatus
  progress: number
  message?: string
  error?: string
  createdAt: string
  startedAt?: string
  completedAt?: string
}


export interface HealthStatus {
  status: 'ok'
  version: string
}

export type UpdateState = 'idle' | 'preparing' | 'installing' | 'restarting' | 'succeeded' | 'failed'

export interface UpdateStatus {
  state: UpdateState
  currentVersion: string
  targetVersion?: string
  packageName?: string
  message?: string
  error?: string
  localOnly: boolean
  startedAt?: string
  updatedAt?: string
  completedAt?: string
  rollbackApplied?: boolean
}

export interface SystemInfo {
  version: string
  ffmpegAvailable: boolean
  ffprobeAvailable: boolean
  metadataEnabled: boolean
}

export interface StorageCategory {
  id: 'updates' | 'optimized-media' | 'transcodes' | 'music-artwork' | 'music-metadata' | 'themes' | 'subtitles-cache' | 'subtitles-downloaded' | 'logs'
  sizeBytes: number
  limitBytes?: number
  reclaimableBytes?: number
  clearable?: boolean
  retentionCount?: number
}

export interface StorageReport {
  dataDirectory: string
  dataSizeBytes: number
  diskFreeBytes: number
  diskTotalBytes: number
  requiredReserveBytes: number
  categories: StorageCategory[]
  generatedAt: string
}

export interface StorageCleanupResult {
  freedBytes: number
  report: StorageReport
}

export type ThemePlaybackMode = 'always' | 'first-open' | 'off'

export interface ThemePlaybackConfiguration {
  seriesPlayback: ThemePlaybackMode
  volume: number
  fadeInMs: number
  fadeOutMs: number
}

export interface ComponentStatus {
  available: boolean
  refreshing?: boolean
  version?: string
  message?: string
  lastCheckedAt?: string
  updatedAt?: string
  sizeBytes?: number
}

export interface SystemConfiguration {
  version: string
  settings: {
    ffmpegPath: string
    ffprobePath: string
    metadata: {
      tmdbTokenConfigured: boolean
      tmdbTokenHint?: string
      language: string
      imdbRatingsSource: string
    }
    music: {
      metadataEnabled: boolean
      shazamEnabled: boolean
      fingerprintEnabled: boolean
      catalogFallbackEnabled: boolean
      audioAnalysisEnabled: boolean
      generatedArtworkEnabled: boolean
      acoustIdApiKeyConfigured: boolean
      acoustIdApiKeyHint?: string
    }
    themeMusic: {
      seriesPlayback: ThemePlaybackMode
    }
    subtitles: {
      preferredLanguages: string[]
      openSubtitlesApiKeyConfigured: boolean
      openSubtitlesApiKeyHint?: string
      openSubtitlesUsernameConfigured: boolean
      openSubtitlesUsernameHint?: string
      openSubtitlesPasswordConfigured: boolean
    }
    quality: {
      profile: 'original' | 'maximum' | 'balanced' | 'data-saver'
      preferredCodec: 'auto' | 'h264' | 'hevc' | 'av1'
      hardwarePreference: 'auto' | 'software' | 'qsv' | 'nvenc' | 'amf'
      cachePolicy: 'off' | 'watched' | 'complete' | 'finish'
      cacheMaxGb: number
      cacheFreeReserveGb?: number
      cacheFreeReservePercent?: number
      gracePeriodSeconds: number
      vmafEnabled: boolean
      vmafTarget: number
      vmafSampleSeconds: number
    }
  }
  status: {
    ffmpeg: ComponentStatus
    ffprobe: ComponentStatus
    metadata: {
      tmdb: ComponentStatus
      imdb: ComponentStatus
    }
  }
}

export interface SystemSettingsUpdate {
  ffmpegPath: string
  ffprobePath: string
  metadata: {
    tmdbToken?: string
    clearTmdbToken?: boolean
    language: string
    imdbRatingsSource: string
  }
  music: {
    metadataEnabled: boolean
    shazamEnabled: boolean
    fingerprintEnabled: boolean
    catalogFallbackEnabled: boolean
    audioAnalysisEnabled: boolean
    generatedArtworkEnabled: boolean
    acoustIdApiKey?: string
    clearAcoustIdApiKey?: boolean
  }
  themeMusic: {
    seriesPlayback: ThemePlaybackMode
  }
  subtitles: {
    preferredLanguages: string[]
    openSubtitlesApiKey?: string
    clearOpenSubtitlesApiKey?: boolean
    openSubtitlesUsername?: string
    clearOpenSubtitlesUsername?: boolean
    openSubtitlesPassword?: string
    clearOpenSubtitlesPassword?: boolean
  }
  quality: {
    profile: 'original' | 'maximum' | 'balanced' | 'data-saver'
    preferredCodec: 'auto' | 'h264' | 'hevc' | 'av1'
    hardwarePreference: 'auto' | 'software' | 'qsv' | 'nvenc' | 'amf'
    cachePolicy: 'off' | 'watched' | 'complete' | 'finish'
    cacheMaxGb: number
    cacheFreeReserveGb: number
    cacheFreeReservePercent: number
    gracePeriodSeconds: number
    vmafEnabled: boolean
    vmafTarget: number
    vmafSampleSeconds: number
  }
}

export interface FileSystemEntry {
  name: string
  path: string
  directory: boolean
  sizeBytes?: number
}

export interface DirectoryListing {
  currentPath?: string
  parentPath?: string
  roots: FileSystemEntry[]
  entries: FileSystemEntry[]
  truncated?: boolean
}

export type FilePickerMode = 'directory' | 'file'

export interface FilePickerRequest {
  mode: FilePickerMode
  title: string
  initialPath?: string
  extensions?: string[]
}

export interface FilePickerResult {
  path: string
}

export interface SeasonGroup {
  number: number
  title: string
  episodeCount: number
  episodes: MediaItem[]
}

export interface SeriesAvailability {
  releasedSeasonCount: number
  releasedEpisodeCount: number
  availableSeasonCount: number
  availableEpisodeCount: number
  latestReleasedEpisode?: EpisodeReleaseInfo
  latestAvailableSeason?: number
  latestAvailableEpisode?: number
  latestAvailableDate?: string
  newEpisodeAvailable: boolean
}

export interface SeriesDetails {
  series: MediaItem
  seasons: SeasonGroup[]
  playTarget?: MediaItem
  availability?: SeriesAvailability
}

export interface VideoDecodeCapability {
  codec: string
  codecProfile?: string
  codecLevel?: number
  source?: boolean
  contentType?: string
  width?: number
  height?: number
  frameRate?: number
  bitrate?: number
  bitDepth?: number
  hdrFormat?: string
  supported: boolean
  smooth: boolean
  powerEfficient: boolean
}

export interface ClientCapabilities {
  clientName: string
  clientVersion: string
  supportedMimes: string[]
  videoCodecs: string[]
  audioCodecs: string[]
  subtitleFormats: string[]
  supportsMse: boolean
  supportsHls: boolean
  maximumWidth?: number
  maximumHeight?: number
  maximumBitrate?: number
  maximumAudioChannels?: number
  hdrFormats?: string[]
  supportsPip?: boolean
  decodeCapabilities?: VideoDecodeCapability[]
}

export type PlaybackMode = 'direct' | 'remux' | 'transcode'
export type HDRMode = 'auto' | 'hdr' | 'sdr'
export type TrackAction = 'copy' | 'transcode' | 'drop' | 'external' | 'burn'

export interface PlaybackPreferences {
  forceTranscode?: boolean
  forceSubtitleBurn?: boolean
  startPositionMs?: number
  audioTrackIndex?: number
  subtitleId?: string
  hdrMode?: HDRMode
  recoveryLevel?: number
}

export type SubtitleSource = 'embedded' | 'external' | 'downloaded'
export type SubtitleKind = 'text' | 'image'

export interface SubtitleTrack {
  id: string
  source: SubtitleSource
  kind: SubtitleKind
  format: string
  language?: string
  title?: string
  streamIndex?: number
  streamOrdinal?: number
  default?: boolean
  forced?: boolean
  hearingImpaired?: boolean
  requiresBurnIn?: boolean
  url?: string
}

export interface PlaybackOptions {
  mediaId: string
  durationMs?: number
  audioTracks: MediaTrack[]
  subtitles: SubtitleTrack[]
}

export interface SubtitleSearchResult {
  id: string
  fileId: number
  fileName: string
  language: string
  release?: string
  hearingImpaired?: boolean
  trusted?: boolean
  rating?: number
  downloads?: number
  matchType?: string
}

export interface PlaybackPlan {
  mode: PlaybackMode
  reason: string
  container: string
  videoCodec?: string
  audioCodec?: string
  videoAction?: TrackAction
  audioAction?: TrackAction
  subtitleAction?: TrackAction
  selectedAudioTrackIndex?: number
  selectedSubtitleId?: string
  toneMap?: boolean
  preserveHdr?: boolean
  sourceHdrFormat?: string
  targetWidth?: number
  targetHeight?: number
  startPositionMs?: number
  durationMs?: number
  directUrl?: string
  playlistUrl?: string
  transcodeProfile?: string
  qualityProfile?: string
  videoEncoder?: string
  videoDecoder?: string
  videoPreset?: string
  encoderFallbackReason?: string
  videoCrf?: number
  pixelFormat?: string
  cachePolicy?: string
  cacheMaxBytes?: number
  gracePeriodSeconds?: number
  vmafEnabled?: boolean
  vmafTarget?: number
  vmafSampleSeconds?: number
  encoderFallback?: boolean
  allowSoftwareAv1?: boolean
  recoveryLevel?: number
  startupBufferSeconds?: number
  rebufferBufferSeconds?: number
  desiredBufferSeconds?: number
  lowBufferSeconds?: number
  readRate?: number
  initialBurstSeconds?: number
  minimumEncoderSpeed?: number
  maxVideoBitrate?: number
  rateControlBuffer?: number
  cacheHit?: boolean
  initialPositionMs?: number
}

export interface PlaybackRuntimeStats {
  sessionId: string
  state: string
  producerAlive: boolean
  encoderSpeed?: number
  averageEncoderSpeed?: number
  outputPositionMs?: number
  generatedDurationMs?: number
  segmentCount?: number
  estimatedBitrate?: number
  minimumEncoderSpeed?: number
  fallbackRecommended?: boolean
  fallbackReason?: string
  updatedAt: string
}

export interface PlaybackSession {
  id?: string
  mediaId: string
  plan: PlaybackPlan
  playlistUrl?: string
  createdAt?: string
  lastAccessAt?: string
}

export type CastProtocol = 'dlna' | 'google-cast'
export type CastPlaybackState = 'connecting' | 'playing' | 'paused' | 'stopped' | 'buffering' | 'unknown'

export interface CastDevice {
  id: string
  name: string
  manufacturer?: string
  model?: string
  protocol: CastProtocol
  canSeek: boolean
  canSetVolume: boolean
}

export interface CastSource {
  id: string
  mediaId: string
  url: string
  contentType: string
  basePositionMs?: number
  startPositionMs?: number
  durationMs?: number
  plan: PlaybackPlan
  createdAt: string
}

export interface CastSession {
  id: string
  device: CastDevice
  mediaId: string
  sourceId: string
  state: CastPlaybackState
  positionMs?: number
  durationMs?: number
  volume?: number
  muted?: boolean
  startedAt: string
  lastUpdateAt: string
}

export type CastCommand =
  | { action: 'play' | 'pause' | 'stop' }
  | { action: 'seek'; positionMs: number }
  | { action: 'set-volume'; volume: number }
  | { action: 'set-muted'; muted: boolean }

export interface MediaQuery {
  libraryId?: string
  seriesId?: string
  season?: number
  type?: MediaType | MediaType[]
  search?: string
  genre?: string | string[]
  yearFrom?: number
  yearTo?: number
  sort?: 'addedAt' | 'modifiedAt' | 'releaseDate' | 'title' | 'rating' | 'year' | 'duration'
  order?: 'asc' | 'desc'
  limit?: number
  offset?: number
}
