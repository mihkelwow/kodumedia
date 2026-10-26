import { useCallback, useEffect, useState } from 'react'
import type { Library, MediaItem } from '../api/types'
import { api } from '../api/client'
import { MediaDetails } from '../features/media/MediaDetails'
import { SeriesDetails } from '../features/media/SeriesDetails'
import { AppHeader } from '../features/navigation/AppHeader'
import { PlayerModal } from '../features/player/PlayerModal'
import { SearchOverlay } from '../features/search/SearchOverlay'
import { HomePage } from '../pages/HomePage'
import { LibraryPage } from '../pages/LibraryPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { NotificationsPage } from '../pages/NotificationsPage'
import { MusicPage } from '../pages/MusicPage'
import { MusicPlayer } from '../features/music/MusicPlayer'
import { SeriesThemePlayer } from '../features/themes/SeriesThemePlayer'
import { SettingsPage } from '../pages/SettingsPage'
import { Button } from '../shared/components/Button'
import { Icon } from '../shared/components/Icon'
import { Spinner } from '../shared/components/Spinner'
import { StaticAppBackdrop } from '../shared/components/StaticAppBackdrop'
import { useDirectionalNavigation } from '../shared/hooks/useDirectionalNavigation'
import { useTranslation } from '../i18n/i18n'
import { useRouter } from './router'
import { usesStaticAppBackdrop } from './staticBackdropRoute'

export function App() {
  const { t } = useTranslation()
  const { route, pathname, navigate } = useRouter()
  const [libraries, setLibraries] = useState<Library[]>([])
  const [loadingLibraries, setLoadingLibraries] = useState(true)
  const [libraryError, setLibraryError] = useState('')
  const [selectedItem, setSelectedItem] = useState<MediaItem | null>(null)
  const [playingItem, setPlayingItem] = useState<MediaItem | null>(null)
  const [searchOpen, setSearchOpen] = useState(false)
  const [watchRevision, setWatchRevision] = useState(0)
  const [musicQueue, setMusicQueue] = useState<MediaItem[]>([])
  const [musicIndex, setMusicIndex] = useState(-1)
  useDirectionalNavigation(!playingItem && !searchOpen)

  const loadLibraries = useCallback(async () => {
    setLibraryError('')
    try {
      setLibraries(await api.libraries.list())
    } catch (caught) {
      setLibraryError(caught instanceof Error ? caught.message : t('Meediakogusid ei saanud laadida.'))
    } finally {
      setLoadingLibraries(false)
    }
  }, [])

  useEffect(() => { void loadLibraries() }, [loadLibraries])
  useEffect(() => { setSelectedItem(null) }, [pathname])
  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null
      const isTyping = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA' || target?.isContentEditable
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        setSearchOpen(true)
        return
      }
      if (!isTyping && event.key === '/') {
        event.preventDefault()
        setSearchOpen(true)
      }
    }
    window.addEventListener('keydown', handleShortcut)
    return () => window.removeEventListener('keydown', handleShortcut)
  }, [])

  const currentLibrary = route.name === 'library'
    ? libraries.find((library) => library.id === route.libraryId)
    : undefined
  const musicLibrary = libraries.find((library) => library.system && library.kind === 'music')
  const staticBackdrop = usesStaticAppBackdrop(route)
  const activeMusicTrack = musicIndex >= 0 ? musicQueue[musicIndex] : undefined
  const activeSeriesThemeId = playingItem
    ? undefined
    : selectedItem?.type === 'series'
      ? selectedItem.id
      : selectedItem?.type === 'episode'
        ? selectedItem.seriesId || undefined
        : undefined

  const openItem = (item: MediaItem) => {
    if (item.type === 'music') {
      setMusicQueue([item])
      setMusicIndex(0)
      navigate('/music')
      return
    }
    setSelectedItem(item)
  }
  const stopMusic = () => {
    setMusicQueue([])
    setMusicIndex(-1)
  }
  const playItem = (item: MediaItem) => {
    if (item.type === 'series' || item.type === 'music') return
    stopMusic()
    setSelectedItem(null)
    setPlayingItem(item)
  }
  const playMusic = (track: MediaItem, queue: MediaItem[]) => {
    setPlayingItem(null)
    const nextQueue = queue.length ? queue : [track]
    const nextIndex = Math.max(0, nextQueue.findIndex((item) => item.id === track.id))
    setMusicQueue(nextQueue)
    setMusicIndex(nextIndex)
  }

  let content
  if (loadingLibraries) {
    content = <div className="app-state"><Spinner label={t('Käivitan meediaserverit')} /><h2>{t('Ühendan serveriga')}</h2></div>
  } else if (libraryError) {
    content = <div className="app-state app-state--error"><Icon name="alert" size={30} /><h2>{t('Serveriga ei saanud ühendust')}</h2><p>{libraryError}</p><Button variant="primary" icon="refresh" onClick={() => void loadLibraries()}>{t('Proovi uuesti')}</Button></div>
  } else if (route.name === 'home') {
    content = <HomePage onOpen={openItem} onNavigate={navigate} refreshToken={watchRevision} />
  } else if (route.name === 'settings') {
    content = <SettingsPage libraries={libraries} onLibrariesChanged={loadLibraries} />
  } else if (route.name === 'notifications') {
    content = <NotificationsPage onOpenSeries={openItem} />
  } else if (route.name === 'music' && musicLibrary) {
    content = <MusicPage library={musicLibrary} artist={route.artist} albumId={route.albumId} tracksView={route.tracks} activeTrackId={activeMusicTrack?.id} onNavigate={navigate} onPlayTrack={playMusic} />
  } else if (route.name === 'library' && currentLibrary) {
    content = <LibraryPage key={currentLibrary.id} library={currentLibrary} onOpen={openItem} onNavigate={navigate} />
  } else {
    content = <NotFoundPage onNavigate={navigate} />
  }

  return (
    <div className={`app-shell ${activeMusicTrack ? 'app-shell--music-player' : ''}`}>
      {staticBackdrop ? <StaticAppBackdrop /> : null}
      <AppHeader libraries={libraries} pathname={pathname} onNavigate={navigate} onOpenSearch={() => setSearchOpen(true)} />
      <main className="main-shell">{content}</main>
      <SeriesThemePlayer seriesId={activeSeriesThemeId} />

      {searchOpen ? <SearchOverlay onClose={() => setSearchOpen(false)} onOpen={openItem} /> : null}
      {selectedItem?.type === 'series' ? (
        <SeriesDetails
          item={selectedItem}
          onClose={() => setSelectedItem(null)}
          onOpenEpisode={(episode) => setSelectedItem(episode)}
          onPlayEpisode={playItem}
        />
      ) : selectedItem ? (
        <MediaDetails
          item={selectedItem}
          onClose={() => setSelectedItem(null)}
          onPlay={playItem}
        />
      ) : null}
      {playingItem ? (
        <PlayerModal
          key={playingItem.id}
          item={playingItem}
          onClose={() => setPlayingItem(null)}
          onPlayItem={(item) => setPlayingItem(item)}
          onWatchUpdated={() => setWatchRevision((value) => value + 1)}
        />
      ) : null}
      {activeMusicTrack ? <MusicPlayer queue={musicQueue} index={musicIndex} onIndexChange={setMusicIndex} /> : null}
    </div>
  )
}
