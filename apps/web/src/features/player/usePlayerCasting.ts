import { useCallback, useEffect, useMemo, useState, useSyncExternalStore, type RefObject } from 'react'
import { api } from '../../api/client'
import type { CastCommand, CastDevice, CastSession, MediaItem, PlaybackPreferences } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { googleCast, type GoogleCastSnapshot } from './googleCast'

interface WebKitPlaybackTargetAvailabilityEvent extends Event {
  availability?: 'available' | 'not-available'
}

interface RemotePlaybackLike extends EventTarget {
  state: 'connecting' | 'connected' | 'disconnected'
  prompt?: () => Promise<void>
  watchAvailability?: (callback: (available: boolean) => void) => Promise<number>
  cancelWatchAvailability?: (id: number) => Promise<void>
}

interface RemoteVideoElement extends HTMLVideoElement {
  webkitShowPlaybackTargetPicker?: () => void
  webkitCurrentPlaybackTargetIsWireless?: boolean
}

interface PlayerCastingInput {
  item: MediaItem
  videoRef: RefObject<HTMLVideoElement | null>
  preferences: PlaybackPreferences
  positionMs: number
  durationMs: number
  menuOpen: boolean
  progressRef: { current: number | null }
}

export interface ActiveRemotePlayback {
  kind: 'dlna' | 'google-cast'
  deviceName: string
  playing: boolean
  positionMs: number
  durationMs: number
  volume: number
  muted: boolean
  canSetVolume: boolean
}

const serverGoogleSnapshot: GoogleCastSnapshot = {
  supported: false,
  loading: false,
  available: false,
  connected: false,
  playing: false,
  positionMs: 0,
  durationMs: 0,
  volume: 1,
  muted: false,
  basePositionMs: 0,
}

function sessionIsPlaying(session: CastSession): boolean {
  return session.state === 'playing' || session.state === 'buffering' || session.state === 'connecting'
}

export function usePlayerCasting({
  item,
  videoRef,
  preferences,
  positionMs,
  durationMs,
  menuOpen,
  progressRef,
}: PlayerCastingInput) {
  const [devices, setDevices] = useState<CastDevice[]>([])
  const [discovering, setDiscovering] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [error, setError] = useState('')
  const [dlnaSession, setDlnaSession] = useState<CastSession | null>(null)
  const [systemSupported, setSystemSupported] = useState(false)
  const [systemAvailable, setSystemAvailable] = useState(false)
  const [systemConnected, setSystemConnected] = useState(false)
  const google = useSyncExternalStore(googleCast.subscribe, googleCast.getSnapshot, () => serverGoogleSnapshot)

  const refreshDevices = useCallback(async () => {
    setDiscovering(true)
    setError('')
    try {
      setDevices(await api.cast.devices())
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Sama võrgu seadmeid ei saanud otsida.'))
    } finally {
      setDiscovering(false)
    }
  }, [])

  useEffect(() => {
    const video = videoRef.current as RemoteVideoElement | null
    if (!video) return
    video.setAttribute('x-webkit-airplay', 'allow')
    const remote = video.remote as RemotePlaybackLike | undefined
    const webKitSupported = typeof video.webkitShowPlaybackTargetPicker === 'function'
    const remoteSupported = typeof remote?.prompt === 'function'
    setSystemSupported(webKitSupported || remoteSupported)
    setSystemAvailable(webKitSupported || remoteSupported)

    const onWebKitAvailability = (event: Event) => {
      const availability = (event as WebKitPlaybackTargetAvailabilityEvent).availability
      setSystemAvailable(availability !== 'not-available')
    }
    const onWebKitWirelessChange = () => {
      setSystemConnected(Boolean(video.webkitCurrentPlaybackTargetIsWireless))
    }
    const onRemoteChange = () => {
      setSystemConnected(remote?.state === 'connected')
    }
    video.addEventListener('webkitplaybacktargetavailabilitychanged', onWebKitAvailability)
    video.addEventListener('webkitcurrentplaybacktargetiswirelesschanged', onWebKitWirelessChange)
    remote?.addEventListener('connect', onRemoteChange)
    remote?.addEventListener('disconnect', onRemoteChange)
    remote?.addEventListener('connecting', onRemoteChange)
    let availabilityID: number | undefined
    if (!webKitSupported && remote?.watchAvailability) {
      void remote.watchAvailability((available) => setSystemAvailable(available)).then((id) => {
        availabilityID = id
      }).catch(() => undefined)
    }
    onWebKitWirelessChange()
    onRemoteChange()
    return () => {
      video.removeEventListener('webkitplaybacktargetavailabilitychanged', onWebKitAvailability)
      video.removeEventListener('webkitcurrentplaybacktargetiswirelesschanged', onWebKitWirelessChange)
      remote?.removeEventListener('connect', onRemoteChange)
      remote?.removeEventListener('disconnect', onRemoteChange)
      remote?.removeEventListener('connecting', onRemoteChange)
      if (availabilityID !== undefined) void remote?.cancelWatchAvailability?.(availabilityID).catch(() => undefined)
    }
  }, [item.id, videoRef])

  useEffect(() => {
    let cancelled = false
    setDlnaSession(null)
    void api.cast.active(item.id).then((session) => {
      if (!cancelled) setDlnaSession(session ?? null)
    }).catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [item.id])

  useEffect(() => {
    if (!menuOpen) return
    void refreshDevices()
    void googleCast.initialize()
  }, [menuOpen, refreshDevices])

  useEffect(() => {
    if (!dlnaSession?.id) return
    const controller = new AbortController()
    let disposed = false
    const update = async () => {
      try {
        const next = await api.cast.status(dlnaSession.id, controller.signal)
        if (!disposed) setDlnaSession(next)
      } catch {
        // A television may temporarily ignore status polling while opening a stream.
      }
    }
    const timer = window.setInterval(() => { void update() }, 2_000)
    void update()
    return () => {
      disposed = true
      controller.abort()
      window.clearInterval(timer)
    }
  }, [dlnaSession?.id])

  const googleActive = Boolean(google.connected && google.sourceId && google.mediaId === item.id)
  const activeRemote = useMemo<ActiveRemotePlayback | null>(() => {
    if (dlnaSession) {
      return {
        kind: 'dlna',
        deviceName: dlnaSession.device.name,
        playing: sessionIsPlaying(dlnaSession),
        positionMs: dlnaSession.positionMs ?? positionMs,
        durationMs: dlnaSession.durationMs ?? durationMs,
        volume: dlnaSession.volume ?? 1,
        muted: Boolean(dlnaSession.muted),
        canSetVolume: dlnaSession.device.canSetVolume,
      }
    }
    if (googleActive) {
      return {
        kind: 'google-cast',
        deviceName: google.deviceName || 'Google Cast',
        playing: google.playing,
        positionMs: google.positionMs,
        durationMs: google.durationMs || durationMs,
        volume: google.volume,
        muted: google.muted,
        canSetVolume: true,
      }
    }
    return null
  }, [dlnaSession, durationMs, google, googleActive, positionMs])

  useEffect(() => {
    progressRef.current = activeRemote?.positionMs ?? null
    return () => {
      progressRef.current = null
    }
  }, [activeRemote?.positionMs, progressRef])

  const openSystemPicker = useCallback(async () => {
    const video = videoRef.current as RemoteVideoElement | null
    if (!video) return
    setError('')
    try {
      if (typeof video.webkitShowPlaybackTargetPicker === 'function') {
        video.webkitShowPlaybackTargetPicker()
        return
      }
      await (video.remote as RemotePlaybackLike | undefined)?.prompt?.()
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Seadmevalikut ei saanud avada.'))
    }
  }, [videoRef])

  const startDLNA = useCallback(async (device: CastDevice) => {
    setConnecting(true)
    setError('')
    videoRef.current?.pause()
    try {
      const next = await api.cast.start({
        deviceId: device.id,
        mediaId: item.id,
        title: item.type === 'episode' && item.seriesTitle ? item.seriesTitle : item.title,
        subtitle: item.type === 'episode' ? item.title : item.year ? String(item.year) : undefined,
        preferences: { ...preferences, startPositionMs: Math.max(0, Math.round(positionMs)) },
      })
      setDlnaSession(next)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Teleris esitamise käivitamine ebaõnnestus.'))
    } finally {
      setConnecting(false)
    }
  }, [item, positionMs, preferences, videoRef])

  const startGoogleCast = useCallback(async () => {
    setConnecting(true)
    setError('')
    videoRef.current?.pause()
    let sourceID = ''
    try {
      await googleCast.requestSession()
      const source = await api.cast.source(item.id, { ...preferences, startPositionMs: Math.max(0, Math.round(positionMs)) })
      sourceID = source.id
      await googleCast.load(source, item)
    } catch (caught) {
      if (sourceID) void api.cast.stopSource(sourceID).catch(() => undefined)
      setError(caught instanceof Error ? caught.message : tr('Google Casti käivitamine ebaõnnestus.'))
    } finally {
      setConnecting(false)
    }
  }, [item, positionMs, preferences, videoRef])

  const command = useCallback(async (action: 'play' | 'pause' | 'seek' | 'set-volume' | 'set-muted', value?: number | boolean) => {
    setError('')
    try {
      if (dlnaSession) {
        const payload: CastCommand = action === 'seek'
          ? { action, positionMs: Number(value) }
          : action === 'set-volume'
            ? { action, volume: Number(value) }
            : action === 'set-muted'
              ? { action, muted: Boolean(value) }
              : { action }
        setDlnaSession(await api.cast.command(dlnaSession.id, payload))
        return
      }
      if (!googleActive) return
      if (action === 'play' || action === 'pause') googleCast.playOrPause()
      else if (action === 'seek') googleCast.seek(Number(value))
      else if (action === 'set-volume') googleCast.setVolume(Number(value))
      else if (action === 'set-muted') googleCast.setMuted(Boolean(value))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Castimise käsk ebaõnnestus.'))
    }
  }, [dlnaSession, googleActive])

  const stop = useCallback(async () => {
    setConnecting(true)
    setError('')
    try {
      if (dlnaSession) {
        await api.cast.stop(dlnaSession.id)
        setDlnaSession(null)
        return
      }
      if (googleActive) {
        const sourceID = google.sourceId
        await googleCast.stop()
        if (sourceID) await api.cast.stopSource(sourceID).catch(() => undefined)
        googleCast.clearSource()
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Castimise lõpetamine ebaõnnestus.'))
    } finally {
      setConnecting(false)
    }
  }, [dlnaSession, google.sourceId, googleActive])

  const togglePlayback = useCallback(() => {
    if (!activeRemote) return
    void command(activeRemote.playing ? 'pause' : 'play')
  }, [activeRemote, command])

  const play = useCallback(() => {
    if (activeRemote && !activeRemote.playing) void command('play')
  }, [activeRemote, command])

  const pause = useCallback(() => {
    if (activeRemote?.playing) void command('pause')
  }, [activeRemote, command])

  const seek = useCallback((nextPositionMs: number) => {
    if (activeRemote) void command('seek', nextPositionMs)
  }, [activeRemote, command])

  const setVolume = useCallback((nextVolume: number) => {
    if (activeRemote?.canSetVolume) void command('set-volume', nextVolume)
  }, [activeRemote?.canSetVolume, command])

  const setMuted = useCallback((nextMuted: boolean) => {
    if (activeRemote?.canSetVolume) void command('set-muted', nextMuted)
  }, [activeRemote?.canSetVolume, command])

  return {
    devices,
    discovering,
    connecting,
    error,
    systemSupported,
    systemAvailable,
    systemConnected,
    google,
    googleActive,
    dlnaSession,
    activeRemote,
    refreshDevices,
    openSystemPicker,
    startDLNA,
    startGoogleCast,
    togglePlayback,
    play,
    pause,
    seek,
    setVolume,
    setMuted,
    stop,
    clearError: () => setError(''),
  }
}
