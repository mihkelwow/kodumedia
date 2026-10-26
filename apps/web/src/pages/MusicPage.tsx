import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Library, MediaItem, MusicAlbum, MusicArtist, MusicLanding, MusicTrackPage, MusicTrackQuery } from '../api/types'
import { api } from '../api/client'
import { MusicArtwork } from '../features/music/MusicArtwork'
import { tr, trCount } from '../i18n/i18n'
import { AmbientBackdrop } from '../shared/components/AmbientBackdrop'
import { Button } from '../shared/components/Button'
import { EmptyState } from '../shared/components/EmptyState'
import { Icon } from '../shared/components/Icon'
import { Spinner } from '../shared/components/Spinner'
import { formatDuration } from '../shared/utils/format'

interface MusicPageProps {
  library: Library
  artist?: string
  albumId?: string
  tracksView?: boolean
  activeTrackId?: string
  onNavigate: (path: string) => void
  onPlayTrack: (track: MediaItem, queue: MediaItem[]) => void
}

const emptyLanding: MusicLanding = { totalTracks: 0, recentlyPlayed: [], recentlyAdded: [], artists: [], albums: [] }
const emptyTrackPage: MusicTrackPage = { items: [], total: 0, limit: 200, offset: 0, facets: { artists: [], albums: [], genres: [], styles: [], years: [], codecs: [], lossless: 0, hiRes: 0, unclassified: 0 } }

function TrackRows({ tracks, activeTrackId, onPlayTrack }: { tracks: MediaItem[]; activeTrackId?: string; onPlayTrack: (track: MediaItem, queue: MediaItem[]) => void }) {
  return (
    <div className="music-track-list">
      {tracks.map((track, index) => (
        <button type="button" className={`music-track-row ${track.id === activeTrackId ? 'music-track-row--active' : ''}`} key={track.id} onClick={() => onPlayTrack(track, tracks)} data-focusable="true">
          <span className="music-track-row__number">{track.trackNumber || index + 1}</span>
          <MusicArtwork src={track.metadata.posterUrl || `/api/v1/media/${encodeURIComponent(track.id)}/artwork`} alt="" className="music-track-row__art" />
          <span className="music-track-row__copy">
            <strong>{track.title}</strong>
            <small>{track.artist || tr('Tundmatu esitaja')}{track.album ? ` · ${track.album}` : ''}</small>
            <span className="music-track-row__classification">
              {(track.musicClassification?.genres?.length || track.metadata.genres?.length) ? <span>{(track.musicClassification?.genres || track.metadata.genres || []).join(' · ')}</span> : null}
              {track.musicClassification?.styles?.length ? <span>{track.musicClassification.styles.join(' · ')}</span> : null}
              {!(track.musicClassification?.genres?.length || track.metadata.genres?.length || track.musicClassification?.styles?.length) ? <span className="music-track-row__unclassified">{tr('Liigitamata')}</span> : null}
            </span>
          </span>
          <span className="music-track-row__quality">
            {track.technical.audioCodec?.toUpperCase() || track.file.extension.replace('.', '').toUpperCase()}
            {track.technical.audioSampleRate ? <small>{Math.round(track.technical.audioSampleRate / 100) / 10} kHz{track.technical.audioBitDepth ? ` · ${track.technical.audioBitDepth}-bit` : ''}</small> : null}
          </span>
          <span className="music-track-row__duration">{track.technical.durationMs ? formatDuration(track.technical.durationMs) : '—'}</span>
          <Icon name={track.id === activeTrackId ? 'volume' : 'play'} size={18} />
        </button>
      ))}
    </div>
  )
}

function AlbumGrid({ albums, onOpen }: { albums: MusicAlbum[]; onOpen: (album: MusicAlbum) => void }) {
  return (
    <div className="music-album-grid">
      {albums.map((album) => (
        <button type="button" className="music-album-card" key={album.id} onClick={() => onOpen(album)} data-focusable="true">
          <MusicArtwork src={album.artworkUrl} alt={album.title} className="music-album-card__art" />
          <span><strong>{album.title}</strong><small>{album.artist}{album.year ? ` · ${album.year}` : ''}</small></span>
        </button>
      ))}
    </div>
  )
}

function ArtistGrid({ artists, onOpen }: { artists: MusicArtist[]; onOpen: (artist: MusicArtist) => void }) {
  return (
    <div className="music-artist-grid">
      {artists.map((artist) => (
        <button type="button" className="music-artist-card" key={artist.id} onClick={() => onOpen(artist)} data-focusable="true">
          <MusicArtwork src={artist.artworkUrl} alt={artist.name} className="music-artist-card__art" />
          <span><strong>{artist.name}</strong><small>{trCount('{{count}} album', '{{count}} albumit', artist.albumCount)} · {trCount('{{count}} lugu', '{{count}} lugu', artist.trackCount)}</small></span>
        </button>
      ))}
    </div>
  )
}

export function MusicPage({ library, artist, albumId, tracksView = false, activeTrackId, onNavigate, onPlayTrack }: MusicPageProps) {
  const [landing, setLanding] = useState<MusicLanding>(emptyLanding)
  const [artistAlbums, setArtistAlbums] = useState<MusicAlbum[]>([])
  const [artistInfo, setArtistInfo] = useState<MusicArtist | null>(null)
  const [album, setAlbum] = useState<MusicAlbum | null>(null)
  const [trackPage, setTrackPage] = useState<MusicTrackPage>(emptyTrackPage)
  const [trackSearch, setTrackSearch] = useState('')
  const [trackRefreshing, setTrackRefreshing] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const loadedTrackLibrary = useRef('')
  const trackQuery: MusicTrackQuery = {
    libraryId: library.id,
    search: trackSearch,
    sort: 'title',
    order: 'asc',
    limit: 200,
    offset: 0,
  }

  useEffect(() => {
    let cancelled = false
    setError('')
    setAlbum(null)
    setArtistInfo(null)
    setArtistAlbums([])
    const initialTrackLoad = tracksView && loadedTrackLibrary.current !== library.id
    if (initialTrackLoad) {
      setLoading(true)
      setTrackPage(emptyTrackPage)
    } else if (tracksView) {
      setTrackRefreshing(true)
    } else {
      loadedTrackLibrary.current = ''
      setLoading(true)
      setTrackPage(emptyTrackPage)
    }

    const task = tracksView
      ? api.music.tracks(trackQuery)
        .then((result) => { if (!cancelled) setTrackPage(result) })
      : albumId
        ? api.music.album(library.id, albumId).then((result) => { if (!cancelled) setAlbum(result.album) })
        : artist
          ? api.music.artist(library.id, artist).then((result) => { if (!cancelled) { setArtistInfo(result.artist); setArtistAlbums(result.albums) } })
          : api.music.landing(library.id).then((result) => { if (!cancelled) setLanding(result) })

    task.catch((caught) => { if (!cancelled) setError(caught instanceof Error ? caught.message : tr('Muusikakogu laadimine ebaõnnestus.')) })
      .finally(() => {
        if (!cancelled) {
          if (tracksView) loadedTrackLibrary.current = library.id
          setLoading(false)
          setTrackRefreshing(false)
        }
      })
    return () => { cancelled = true }
  }, [albumId, artist, library.id, trackSearch, tracksView])

  const loadMoreTracks = useCallback(async () => {
    if (loadingMore || trackPage.items.length >= trackPage.total) return
    setLoadingMore(true)
    setError('')
    try {
      const next = await api.music.tracks({ ...trackQuery, offset: trackPage.items.length })
      setTrackPage((current) => ({ ...next, items: [...current.items, ...next.items], offset: 0 }))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Muusikakogu laadimine ebaõnnestus.'))
    } finally {
      setLoadingMore(false)
    }
  }, [library.id, loadingMore, trackPage.items.length, trackPage.total, trackSearch])

  const ambient = useMemo(() => {
    if (album?.artworkUrl) return album.artworkUrl
    if (artistInfo?.artworkUrl) return artistInfo.artworkUrl
    if (tracksView) return trackPage.items[0]?.metadata.posterUrl
    return landing.recentlyPlayed[0]?.metadata.posterUrl || landing.albums[0]?.artworkUrl
  }, [album, artistInfo, landing, trackPage.items, tracksView])
  const usesArtworkBackdrop = Boolean(artist || albumId || tracksView)

  if (loading) return <div className="page music-page cinematic-page"><div className="page-loader"><Spinner label={tr('Laen muusikakogu')} /></div></div>

  return (
    <div className="page music-page cinematic-page">
      {usesArtworkBackdrop ? <AmbientBackdrop url={ambient} subtle /> : null}
      {usesArtworkBackdrop ? <div className="ambient-vignette" aria-hidden="true" /> : null}
      {error ? <div className="notice notice--error"><Icon name="alert" />{error}</div> : null}

      {tracksView ? (
        <>
          <header className="music-library-header">
            <div>
              <Button compact variant="ghost" icon="rewind" onClick={() => onNavigate('/music')}>{tr('Tagasi muusika avalehele')}</Button>
              <span className="eyebrow">{tr('Muusikakogu')}</span>
              <h1>{tr('Kõik lood')}</h1>
              <p>{tr('Otsi kogu muusikateegist.')}</p>
            </div>
            <label className={`music-track-search ${trackRefreshing ? 'music-track-search--loading' : ''}`}>
              <Icon name="search" size={18} />
              <input value={trackSearch} onChange={(event) => setTrackSearch(event.target.value)} placeholder={tr('Otsi lugusid, artiste, albumeid või stiile')} data-focusable="true" />
              {trackRefreshing ? <span className="music-track-search__status" aria-live="polite">{tr('Otsin')}</span> : null}
            </label>
          </header>
          <section className="music-section music-section--all-tracks">
            <div className="music-section__heading">
              <h2>{trCount('{{count}} lugu', '{{count}} lugu', trackPage.total)}</h2>
              {trackPage.items.length ? <span>{tr('{{shown}} / {{total}} kuvatud', { shown: trackPage.items.length, total: trackPage.total })}</span> : null}
            </div>
            {trackPage.items.length ? <TrackRows tracks={trackPage.items} activeTrackId={activeTrackId} onPlayTrack={onPlayTrack} /> : null}
            {!trackPage.items.length ? <EmptyState icon="search" title={tr('Lugusid ei leitud')} description={tr('Muuda otsingut või proovi mõnda teist artisti või albumit.')} /> : null}
            {trackPage.items.length < trackPage.total ? <div className="music-load-more"><Button variant="secondary" icon="plus" disabled={loadingMore} onClick={() => void loadMoreTracks()}>{loadingMore ? tr('Laen…') : tr('Laadi veel')}</Button></div> : null}
          </section>
        </>
      ) : album ? (
        <>
          <header className="music-detail-hero">
            <Button compact variant="ghost" icon="rewind" onClick={() => onNavigate(`/music/artist/${encodeURIComponent(album.artist)}`)}>{tr('Tagasi artisti juurde')}</Button>
            <div className="music-detail-hero__content">
              <MusicArtwork src={album.artworkUrl} alt={album.title} className="music-detail-hero__art" />
              <div><span className="eyebrow">{tr('Album')}</span><h1>{album.title}</h1><button type="button" className="music-link" onClick={() => onNavigate(`/music/artist/${encodeURIComponent(album.artist)}`)} data-focusable="true" data-pressable="true">{album.artist}</button><p>{album.year || ''}{album.year ? ' · ' : ''}{trCount('{{count}} lugu', '{{count}} lugu', album.trackCount)}</p>{album.tracks?.length ? <Button variant="primary" icon="play" onClick={() => onPlayTrack(album.tracks?.[0] as MediaItem, album.tracks as MediaItem[])}>{tr('Esita album')}</Button> : null}</div>
            </div>
          </header>
          <section className="music-section"><div className="music-section__heading"><h2>{tr('Lood')}</h2></div><TrackRows tracks={album.tracks || []} activeTrackId={activeTrackId} onPlayTrack={onPlayTrack} /></section>
        </>
      ) : artistInfo ? (
        <>
          <header className="music-detail-hero music-detail-hero--artist">
            <Button compact variant="ghost" icon="rewind" onClick={() => onNavigate('/music')}>{tr('Tagasi muusika avalehele')}</Button>
            <div className="music-detail-hero__content">
              <MusicArtwork src={artistInfo.artworkUrl} alt={artistInfo.name} className="music-detail-hero__art music-detail-hero__art--round" />
              <div><span className="eyebrow">{tr('Artist')}</span><h1>{artistInfo.name}</h1><p>{trCount('{{count}} album', '{{count}} albumit', artistInfo.albumCount)} · {trCount('{{count}} lugu', '{{count}} lugu', artistInfo.trackCount)}</p></div>
            </div>
          </header>
          <section className="music-section"><div className="music-section__heading"><h2>{tr('Albumid')}</h2></div><AlbumGrid albums={artistAlbums} onOpen={(item) => onNavigate(`/music/album/${encodeURIComponent(item.id)}`)} /></section>
        </>
      ) : (
        <>
          <header className="music-page__hero"><span className="eyebrow">{tr('Kadudeta kuulamine')}</span><h1>{tr('Muusika')}</h1><p>{tr('Originaalheli eelistav audiofiilne muusikakogu.')}</p></header>
          <section className="music-section music-all-tracks-entry">
            <span className="music-all-tracks-entry__icon" aria-hidden="true"><Icon name="music" size={38} /></span>
            <div className="music-all-tracks-entry__copy">
              <span className="eyebrow">{tr('Muusikakogu')}</span>
              <h2>{tr('Kõik lood')}</h2>
              <p>{trCount('{{count}} lugu serveris', '{{count}} lugu serveris', landing.totalTracks)}</p>
            </div>
            <Button className="music-all-tracks-entry__action" variant="primary" icon="forward" onClick={() => onNavigate('/music/tracks')}>{tr('Vaata kõiki')}</Button>
          </section>
          {landing.recentlyPlayed.length ? <section className="music-section"><div className="music-section__heading"><h2>{tr('Viimati kuulatud')}</h2></div><TrackRows tracks={landing.recentlyPlayed} activeTrackId={activeTrackId} onPlayTrack={onPlayTrack} /></section> : null}
          {landing.recentlyAdded.length ? <section className="music-section"><div className="music-section__heading"><h2>{tr('Viimati lisatud')}</h2></div><TrackRows tracks={landing.recentlyAdded.slice(0, 10)} activeTrackId={activeTrackId} onPlayTrack={onPlayTrack} /></section> : null}
          {landing.albums.length ? <section className="music-section"><div className="music-section__heading"><h2>{tr('Albumid')}</h2></div><AlbumGrid albums={landing.albums} onOpen={(item) => onNavigate(`/music/album/${encodeURIComponent(item.id)}`)} /></section> : null}
          {landing.artists.length ? <section className="music-section"><div className="music-section__heading"><h2>{tr('Artistid')}</h2></div><ArtistGrid artists={landing.artists} onOpen={(item) => onNavigate(`/music/artist/${encodeURIComponent(item.name)}`)} /></section> : null}
          {!landing.recentlyAdded.length && !landing.albums.length ? <EmptyState icon="music" title={tr('Muusikakogu on tühi')} description={tr('Lisa Muusika kogule artisti ja albumi kaustadega helifailid ning käivita skannimine.')} action={<Button variant="primary" icon="settings" onClick={() => onNavigate('/settings')}>{tr('Ava seaded')}</Button>} /> : null}
        </>
      )}
    </div>
  )
}
