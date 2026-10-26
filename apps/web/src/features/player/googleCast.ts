import type { CastSource, MediaItem } from '../../api/types'
import { tr } from '../../i18n/i18n'

interface GoogleCastWindow extends Window {
  __onGCastApiAvailable?: (available: boolean) => void
  cast?: any
  chrome?: any
}

export interface GoogleCastSnapshot {
  supported: boolean
  loading: boolean
  available: boolean
  connected: boolean
  deviceName?: string
  playing: boolean
  positionMs: number
  durationMs: number
  volume: number
  muted: boolean
  sourceId?: string
  mediaId?: string
  basePositionMs: number
  error?: string
}

const initialSnapshot: GoogleCastSnapshot = {
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

class GoogleCastManager {
  private snapshot: GoogleCastSnapshot = initialSnapshot
  private listeners = new Set<() => void>()
  private initializePromise: Promise<void> | null = null
  private remotePlayer: any
  private remoteController: any
  private context: any
  private activeSource: CastSource | null = null

  subscribe = (listener: () => void) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  getSnapshot = () => this.snapshot

  initialize(): Promise<void> {
    if (this.initializePromise) return this.initializePromise
    this.initializePromise = this.initializeInternal()
    return this.initializePromise
  }

  private initializeInternal(): Promise<void> {
    if (typeof window === 'undefined') return Promise.resolve()
    const ios = /iPad|iPhone|iPod/.test(window.navigator.userAgent)
    if (!window.isSecureContext || ios) {
      this.update({
        supported: false,
        loading: false,
        error: !window.isSecureContext
          ? tr('Google Cast vajab turvalist HTTPS-ühendust.')
          : tr("Google Cast ei ole iOS-i veebibrauseris saadaval."),
      })
      return Promise.resolve()
    }
    const castWindow = window as GoogleCastWindow
    if (castWindow.cast?.framework && castWindow.chrome?.cast) {
      this.setupFramework(castWindow)
      return Promise.resolve()
    }
    this.update({ supported: true, loading: true, error: undefined })
    return new Promise((resolve) => {
      let settled = false
      const finish = () => {
        if (settled) return
        settled = true
        resolve()
      }
      const previous = castWindow.__onGCastApiAvailable
      castWindow.__onGCastApiAvailable = (available: boolean) => {
        previous?.(available)
        if (available && castWindow.cast?.framework && castWindow.chrome?.cast) {
          this.setupFramework(castWindow)
        } else {
          this.update({ supported: false, loading: false, error: tr('Google Casti teenust ei saanud laadida.') })
        }
        finish()
      }
      const existing = document.querySelector<HTMLScriptElement>('script[data-kodumedia-google-cast]')
      if (!existing) {
        const script = document.createElement('script')
        script.src = 'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1'
        script.async = true
        script.dataset.kodumediaGoogleCast = 'true'
        script.addEventListener('error', () => {
          this.update({ supported: false, loading: false, error: tr('Google Casti teenust ei saanud laadida.') })
          finish()
        }, { once: true })
        document.head.append(script)
      }
      window.setTimeout(() => {
        if (!settled) {
          this.update({ supported: false, loading: false, error: tr('Google Casti seadmete otsing aegus.') })
          finish()
        }
      }, 10_000)
    })
  }

  private setupFramework(castWindow: GoogleCastWindow) {
    const framework = castWindow.cast.framework
    this.context = framework.CastContext.getInstance()
    this.context.setOptions({
      receiverApplicationId: castWindow.chrome.cast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
      autoJoinPolicy: castWindow.chrome.cast.AutoJoinPolicy.ORIGIN_SCOPED,
      resumeSavedSession: true,
    })
    this.remotePlayer = new framework.RemotePlayer()
    this.remoteController = new framework.RemotePlayerController(this.remotePlayer)
    this.remoteController.addEventListener(framework.RemotePlayerEventType.ANY_CHANGE, () => this.refresh())
    this.context.addEventListener(framework.CastContextEventType.CAST_STATE_CHANGED, () => this.refresh())
    if (framework.CastContextEventType.SESSION_STATE_CHANGED) {
      this.context.addEventListener(framework.CastContextEventType.SESSION_STATE_CHANGED, () => this.refresh())
    }
    this.update({ supported: true, loading: false, error: undefined })
    this.refresh()
  }

  async requestSession(): Promise<void> {
    await this.initialize()
    if (!this.context) throw new Error(this.snapshot.error || tr('Google Cast ei ole saadaval.'))
    await this.context.requestSession()
    this.refresh()
  }

  async load(source: CastSource, item: MediaItem): Promise<void> {
    const castWindow = window as GoogleCastWindow
    const session = this.context?.getCurrentSession?.()
    if (!session || !castWindow.chrome?.cast?.media) throw new Error(tr('Google Casti ühendus puudub.'))
    const mediaInfo = new castWindow.chrome.cast.media.MediaInfo(source.url, source.contentType)
    const metadata = new castWindow.chrome.cast.media.GenericMediaMetadata()
    metadata.title = item.type === 'episode' && item.seriesTitle ? item.seriesTitle : item.title
    metadata.subtitle = item.type === 'episode' ? item.title : item.year ? String(item.year) : ''
    const artwork = item.metadata.posterUrl || item.metadata.backdropUrl
    if (artwork) {
      metadata.images = [new castWindow.chrome.cast.Image(new URL(artwork, window.location.href).toString())]
    }
    mediaInfo.metadata = metadata
    mediaInfo.streamType = castWindow.chrome.cast.media.StreamType.BUFFERED
    const request = new castWindow.chrome.cast.media.LoadRequest(mediaInfo)
    request.autoplay = true
    request.currentTime = Math.max(0, (source.startPositionMs ?? 0) / 1000)
    await session.loadMedia(request)
    this.activeSource = source
    this.refresh()
  }

  playOrPause() {
    this.remoteController?.playOrPause?.()
  }

  seek(positionMS: number) {
    if (!this.remotePlayer || !this.remoteController) return
    this.remotePlayer.currentTime = Math.max(0, (positionMS - (this.activeSource?.basePositionMs ?? 0)) / 1000)
    this.remoteController.seek()
  }

  setVolume(volume: number) {
    if (!this.remotePlayer || !this.remoteController) return
    this.remotePlayer.volumeLevel = Math.max(0, Math.min(1, volume))
    this.remoteController.setVolumeLevel()
  }

  setMuted(muted: boolean) {
    if (!this.remotePlayer || !this.remoteController || Boolean(this.remotePlayer.isMuted) === muted) return
    this.remoteController.muteOrUnmute()
  }

  async stop(): Promise<void> {
    try {
      this.remoteController?.stop?.()
      this.context?.endCurrentSession?.(true)
    } finally {
      this.activeSource = null
      this.refresh()
    }
  }

  clearSource() {
    this.activeSource = null
    this.refresh()
  }

  private refresh() {
    const castWindow = typeof window === 'undefined' ? undefined : window as GoogleCastWindow
    const framework = castWindow?.cast?.framework
    if (!framework || !this.context || !this.remotePlayer) return
    const castState = this.context.getCastState?.()
    const noDevices = framework.CastState?.NO_DEVICES_AVAILABLE
    const session = this.context.getCurrentSession?.()
    const deviceName = session?.getCastDevice?.()?.friendlyName
    const basePositionMs = this.activeSource?.basePositionMs ?? 0
    const remotePositionMs = Math.max(0, Math.round((Number(this.remotePlayer.currentTime) || 0) * 1000))
    this.update({
      supported: true,
      loading: false,
      available: castState !== noDevices,
      connected: Boolean(this.remotePlayer.isConnected && session),
      deviceName: deviceName || undefined,
      playing: Boolean(this.remotePlayer.isConnected && this.remotePlayer.isMediaLoaded && !this.remotePlayer.isPaused),
      positionMs: basePositionMs + remotePositionMs,
      durationMs: this.activeSource?.durationMs ?? Math.round((Number(this.remotePlayer.duration) || 0) * 1000),
      volume: Number.isFinite(this.remotePlayer.volumeLevel) ? Number(this.remotePlayer.volumeLevel) : 1,
      muted: Boolean(this.remotePlayer.isMuted),
      sourceId: this.activeSource?.id,
      mediaId: this.activeSource?.mediaId,
      basePositionMs,
      error: undefined,
    })
  }

  private update(changes: Partial<GoogleCastSnapshot>) {
    this.snapshot = { ...this.snapshot, ...changes }
    this.listeners.forEach((listener) => listener())
  }
}

export const googleCast = new GoogleCastManager()
