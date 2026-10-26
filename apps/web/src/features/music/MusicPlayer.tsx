import { useEffect, useMemo, useRef, useState } from 'react'
import type { MediaItem } from '../../api/types'
import { api } from '../../api/client'
import { tr } from '../../i18n/i18n'
import { Icon } from '../../shared/components/Icon'
import { formatDuration } from '../../shared/utils/format'
import { MusicArtwork } from './MusicArtwork'
import { MusicTrackDetails } from './MusicTrackDetails'

interface MusicPlayerProps {
  queue: MediaItem[]
  index: number
  onIndexChange: (index: number) => void
}

export function MusicPlayer({ queue, index, onIndexChange }: MusicPlayerProps) {
  const track = index >= 0 ? queue[index] : undefined
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const progressSentAt = useRef(0)
  const [playing, setPlaying] = useState(false)
  const [position, setPosition] = useState(0)
  const [duration, setDuration] = useState(0)
  const [volume, setVolume] = useState(1)
  const [losslessFallback, setLosslessFallback] = useState(false)
  const [detailsOpen, setDetailsOpen] = useState(false)
  const [error, setError] = useState('')

  const source = useMemo(() => track
    ? (losslessFallback ? api.music.losslessUrl(track.id) : api.music.directUrl(track.id))
    : '', [losslessFallback, track])

  useEffect(() => {
    setPosition(0)
    setDuration((track?.technical.durationMs || 0) / 1000)
    setLosslessFallback(false)
    setError('')
    progressSentAt.current = 0
  }, [track?.id])

  useEffect(() => {
    if (!track || !audioRef.current) return
    audioRef.current.load()
    void audioRef.current.play().then(() => setPlaying(true)).catch(() => setPlaying(false))
  }, [track?.id, source])

  useEffect(() => {
    if (!track || !('mediaSession' in navigator)) return
    navigator.mediaSession.metadata = new MediaMetadata({
      title: track.title,
      artist: track.artist || '',
      album: track.album || '',
      artwork: track.metadata.posterUrl ? [{ src: track.metadata.posterUrl }] : undefined,
    })
    navigator.mediaSession.setActionHandler('play', () => void audioRef.current?.play())
    navigator.mediaSession.setActionHandler('pause', () => audioRef.current?.pause())
    navigator.mediaSession.setActionHandler('previoustrack', () => { if (index > 0) onIndexChange(index - 1) })
    navigator.mediaSession.setActionHandler('nexttrack', () => { if (index + 1 < queue.length) onIndexChange(index + 1) })
    return () => {
      navigator.mediaSession.metadata = null
      for (const action of ['play', 'pause', 'previoustrack', 'nexttrack'] as MediaSessionAction[]) {
        try { navigator.mediaSession.setActionHandler(action, null) } catch { /* unsupported action */ }
      }
    }
  }, [index, onIndexChange, queue.length, track])

  const sendProgress = (completed = false) => {
    if (!track || !audioRef.current) return
    const audio = audioRef.current
    void api.watch.update(track.id, {
      positionMs: Math.round(audio.currentTime * 1000),
      durationMs: Math.round((Number.isFinite(audio.duration) ? audio.duration : duration) * 1000),
      completed,
    }, completed)
  }

  const onTimeUpdate = () => {
    const audio = audioRef.current
    if (!audio) return
    setPosition(audio.currentTime)
    if (Date.now() - progressSentAt.current >= 10_000) {
      progressSentAt.current = Date.now()
      sendProgress(false)
    }
  }

  const onEnded = () => {
    setPlaying(false)
    sendProgress(true)
    if (index + 1 < queue.length) onIndexChange(index + 1)
  }

  const onAudioError = () => {
    if (track && !losslessFallback) {
      setLosslessFallback(true)
      setError('')
      return
    }
    setError(tr('Audiofaili ei saanud esitada.'))
    setPlaying(false)
  }

  return (
    <>
      <aside className={`music-player ${track ? 'music-player--active' : ''}`} aria-label={tr('Muusikapleier')}>
        <audio
          ref={audioRef}
          src={source || undefined}
          preload="auto"
          onPlay={() => setPlaying(true)}
          onPause={() => setPlaying(false)}
          onLoadedMetadata={(event) => setDuration(Number.isFinite(event.currentTarget.duration) ? event.currentTarget.duration : duration)}
          onTimeUpdate={onTimeUpdate}
          onEnded={onEnded}
          onError={onAudioError}
          onVolumeChange={(event) => setVolume(event.currentTarget.volume)}
        />
        <div className="music-player__track">
          <MusicArtwork src={track?.metadata.posterUrl} alt={track?.album || ''} className="music-player__art" />
          <div><strong>{track?.title || tr('Vali kuulamiseks lugu')}</strong><small>{track ? `${track.artist || tr('Tundmatu esitaja')}${track.album ? ` · ${track.album}` : ''}` : tr('Kadudeta heli originaalkvaliteediga')}</small></div>
        </div>
        <div className="music-player__transport">
          <button type="button" onClick={() => index > 0 && onIndexChange(index - 1)} disabled={index <= 0} aria-label={tr('Eelmine lugu')} data-focusable="true" data-pressable="true"><Icon name="rewind" size={19} /></button>
          <button type="button" className="music-player__play" disabled={!track} onClick={() => playing ? audioRef.current?.pause() : void audioRef.current?.play()} aria-label={playing ? tr('Paus') : tr('Esita')} data-focusable="true" data-pressable="true"><Icon name={playing ? 'pause' : 'play'} size={22} /></button>
          <button type="button" onClick={() => index + 1 < queue.length && onIndexChange(index + 1)} disabled={index < 0 || index + 1 >= queue.length} aria-label={tr('Järgmine lugu')} data-focusable="true" data-pressable="true"><Icon name="forward" size={19} /></button>
        </div>
        <div className="music-player__timeline">
          <span>{formatDuration(Math.round(position * 1000))}</span>
          <input type="range" min={0} max={Math.max(duration, 1)} step={0.1} value={Math.min(position, duration || 0)} disabled={!track} onChange={(event) => { const next = Number(event.target.value); if (audioRef.current) audioRef.current.currentTime = next; setPosition(next) }} aria-label={tr('Loo ajajoon')} data-focusable="true" />
          <span>{formatDuration(Math.round(duration * 1000))}</span>
        </div>
        <div className="music-player__tools">
          {losslessFallback ? <span className="music-player__lossless" title={tr('Brauserile dekodeeritakse kadudeta PCM-voog')}>LOSSLESS</span> : null}
          <Icon name={volume === 0 ? 'mute' : 'volume'} size={18} />
          <input type="range" min={0} max={1} step={0.01} value={volume} onChange={(event) => { const next = Number(event.target.value); setVolume(next); if (audioRef.current) audioRef.current.volume = next }} aria-label={tr('Helitugevus')} data-focusable="true" />
          <button type="button" disabled={!track} onClick={() => setDetailsOpen(true)} aria-label={tr('Ava audiofaili detailid')} data-focusable="true" data-pressable="true"><Icon name="info" size={19} /></button>
        </div>
        {error ? <span className="music-player__error">{error}</span> : null}
      </aside>
      {detailsOpen && track ? <MusicTrackDetails track={track} onClose={() => setDetailsOpen(false)} /> : null}
    </>
  )
}
