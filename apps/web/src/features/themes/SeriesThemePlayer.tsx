import { useCallback, useEffect, useRef, useState, type MutableRefObject } from 'react'
import { api } from '../../api/client'
import type { ThemePlaybackConfiguration } from '../../api/types'
import { SERIES_THEME_SETTINGS_CHANGED_EVENT } from './events'

const SESSION_KEY = 'kodumedia:series-themes-played'
const DEFAULT_CONFIGURATION: ThemePlaybackConfiguration = {
  seriesPlayback: 'always',
  volume: 0.15,
  fadeInMs: 2000,
  fadeOutMs: 2000,
}

function readPlayedSeries(): Set<string> {
  if (typeof window === 'undefined') return new Set()
  try {
    const parsed = JSON.parse(window.sessionStorage.getItem(SESSION_KEY) || '[]')
    return new Set(Array.isArray(parsed) ? parsed.filter((value): value is string => typeof value === 'string') : [])
  } catch {
    return new Set()
  }
}

function rememberPlayedSeries(seriesId: string): void {
  if (typeof window === 'undefined') return
  try {
    const played = readPlayedSeries()
    played.add(seriesId)
    window.sessionStorage.setItem(SESSION_KEY, JSON.stringify([...played]))
  } catch {
    // Session storage is optional; playback still works without it.
  }
}

function cancelFrame(frame: MutableRefObject<number | null>): void {
  if (frame.current !== null) {
    window.cancelAnimationFrame(frame.current)
    frame.current = null
  }
}

function retryDelay(response: Response, fallbackMs: number): number {
  const seconds = Number.parseInt(response.headers.get('Retry-After') || '', 10)
  if (!Number.isFinite(seconds) || seconds <= 0) return fallbackMs
  return Math.min(30_000, Math.max(1_000, seconds * 1000))
}

export function SeriesThemePlayer({ seriesId }: { seriesId?: string }) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const currentSeriesRef = useRef<string | null>(null)
  const transitionRef = useRef(0)
  const frameRef = useRef<number | null>(null)
  const rampResolveRef = useRef<(() => void) | null>(null)
  const autoplayCleanupRef = useRef<(() => void) | null>(null)
  const preparationCleanupRef = useRef<(() => void) | null>(null)
  const sourceCleanupRef = useRef<(() => void) | null>(null)
  const unavailableSeriesRef = useRef(new Set<string>())
  const [configuration, setConfiguration] = useState<ThemePlaybackConfiguration | null>(null)

  const loadConfiguration = useCallback(async () => {
    try {
      setConfiguration(await api.themes.settings())
    } catch {
      setConfiguration({ ...DEFAULT_CONFIGURATION, seriesPlayback: 'off' })
    }
  }, [])

  useEffect(() => {
    void loadConfiguration()
    const refresh = () => { void loadConfiguration() }
    window.addEventListener(SERIES_THEME_SETTINGS_CHANGED_EVENT, refresh)
    return () => window.removeEventListener(SERIES_THEME_SETTINGS_CHANGED_EVENT, refresh)
  }, [loadConfiguration])

  const removeAutoplayListeners = useCallback(() => {
    autoplayCleanupRef.current?.()
    autoplayCleanupRef.current = null
  }, [])

  const cancelPreparation = useCallback(() => {
    preparationCleanupRef.current?.()
    preparationCleanupRef.current = null
  }, [])

  const removeSourceListeners = useCallback(() => {
    sourceCleanupRef.current?.()
    sourceCleanupRef.current = null
  }, [])

  const cancelRamp = useCallback(() => {
    cancelFrame(frameRef)
    const resolve = rampResolveRef.current
    rampResolveRef.current = null
    resolve?.()
  }, [])

  const rampVolume = useCallback((target: number, durationMs: number): Promise<void> => {
    const audio = audioRef.current
    cancelRamp()
    if (!audio) return Promise.resolve()

    const startVolume = audio.volume
    const startedAt = performance.now()
    return new Promise((resolve) => {
      const finish = () => {
        frameRef.current = null
        rampResolveRef.current = null
        resolve()
      }
      if (durationMs <= 0 || Math.abs(startVolume - target) < 0.001) {
        audio.volume = target
        finish()
        return
      }
      rampResolveRef.current = resolve
      const tick = (now: number) => {
        const progress = Math.min(1, (now - startedAt) / durationMs)
        audio.volume = startVolume + (target - startVolume) * progress
        if (progress < 1) frameRef.current = window.requestAnimationFrame(tick)
        else finish()
      }
      frameRef.current = window.requestAnimationFrame(tick)
    })
  }, [cancelRamp])

  const clearSource = useCallback(() => {
    const audio = audioRef.current
    removeAutoplayListeners()
    cancelPreparation()
    removeSourceListeners()
    cancelRamp()
    if (audio) {
      audio.pause()
      audio.removeAttribute('src')
      audio.load()
      audio.volume = 0
    }
    currentSeriesRef.current = null
  }, [cancelPreparation, cancelRamp, removeAutoplayListeners, removeSourceListeners])

  useEffect(() => {
    const audio = audioRef.current
    const settings = configuration
    if (!audio || !settings) return

    const targetSeriesId = seriesId || null
    if (targetSeriesId && settings.seriesPlayback !== 'off' && currentSeriesRef.current === targetSeriesId) return

    const transition = ++transitionRef.current
    const reconcile = async () => {
      if (!targetSeriesId || settings.seriesPlayback === 'off') {
        removeAutoplayListeners()
        cancelPreparation()
        if (!audio.paused && !audio.ended) await rampVolume(0, settings.fadeOutMs)
        if (transition === transitionRef.current) clearSource()
        return
      }

      removeAutoplayListeners()
      cancelPreparation()
      if (currentSeriesRef.current && !audio.paused && !audio.ended) {
        await rampVolume(0, settings.fadeOutMs)
      }
      if (transition !== transitionRef.current) return
      clearSource()

      if (settings.seriesPlayback === 'first-open' && readPlayedSeries().has(targetSeriesId)) return
      if (unavailableSeriesRef.current.has(targetSeriesId)) return

      currentSeriesRef.current = targetSeriesId
      const themeUrl = api.themes.seriesUrl(targetSeriesId)
      const controller = new AbortController()
      let retryTimer: number | null = null
      preparationCleanupRef.current = () => {
        controller.abort()
        if (retryTimer !== null) window.clearTimeout(retryTimer)
      }

      const isCurrent = () => (
        transition === transitionRef.current
        && currentSeriesRef.current === targetSeriesId
        && !controller.signal.aborted
      )

      const markUnavailable = () => {
        if (!isCurrent()) return
        unavailableSeriesRef.current.add(targetSeriesId)
        clearSource()
      }

      const attemptPlay = async () => {
        if (!isCurrent()) return
        try {
          await audio.play()
          removeAutoplayListeners()
        } catch (caught) {
          if (!isCurrent()) return
          const errorName = caught instanceof DOMException ? caught.name : ''
          if (errorName !== 'NotAllowedError') {
            markUnavailable()
            return
          }
          const retry = () => { void attemptPlay() }
          window.addEventListener('pointerdown', retry, { once: true, capture: true })
          window.addEventListener('keydown', retry, { once: true, capture: true })
          autoplayCleanupRef.current = () => {
            window.removeEventListener('pointerdown', retry, true)
            window.removeEventListener('keydown', retry, true)
          }
        }
      }

      const startAudio = () => {
        if (!isCurrent()) return
        if (retryTimer !== null) window.clearTimeout(retryTimer)
        preparationCleanupRef.current = null
        audio.loop = false
        audio.preload = 'auto'
        audio.volume = 0
        audio.src = themeUrl

        const beginFadeIn = () => {
          if (!isCurrent()) return
          rememberPlayedSeries(targetSeriesId)
          void rampVolume(Math.min(1, Math.max(0, settings.volume)), settings.fadeInMs)
        }
        audio.addEventListener('playing', beginFadeIn, { once: true })
        audio.addEventListener('error', markUnavailable, { once: true })
        sourceCleanupRef.current = () => {
          audio.removeEventListener('playing', beginFadeIn)
          audio.removeEventListener('error', markUnavailable)
        }
        void attemptPlay()
      }

      const poll = async () => {
        if (!isCurrent()) return
        try {
          const response = await fetch(themeUrl, {
            method: 'HEAD',
            cache: 'no-store',
            credentials: 'same-origin',
            signal: controller.signal,
          })
          if (!isCurrent()) return
          if (response.ok) {
            startAudio()
            return
          }
          if (response.status === 404 || response.status === 400 || response.status === 422) {
            markUnavailable()
            return
          }
          const delay = response.status === 202
            ? retryDelay(response, 2000)
            : retryDelay(response, 5000)
          retryTimer = window.setTimeout(() => { void poll() }, delay)
        } catch (caught) {
          if (!isCurrent() || (caught instanceof DOMException && caught.name === 'AbortError')) return
          retryTimer = window.setTimeout(() => { void poll() }, 5000)
        }
      }

      await poll()
    }

    void reconcile()
  }, [cancelPreparation, clearSource, configuration, rampVolume, removeAutoplayListeners, seriesId])

  useEffect(() => () => {
    transitionRef.current += 1
    clearSource()
  }, [clearSource])

  return <audio ref={audioRef} hidden aria-hidden="true" />
}
