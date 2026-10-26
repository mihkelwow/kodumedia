import {
  memo,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type TouchEvent as ReactTouchEvent,
} from 'react'
import useEmblaCarousel from 'embla-carousel-react'
import type { MediaItem, TrailerCandidate } from '../../api/types'
import { Icon } from '../../shared/components/Icon'
import { tr } from '../../i18n/i18n'
import { MediaPlayButton } from './MediaPlayButton'

type YouTubePlayerState = number

interface YouTubePlayer {
  destroy: () => void
  mute: () => void
  unMute: () => void
  playVideo: () => void
  pauseVideo: () => void
  getPlayerState: () => YouTubePlayerState
}

interface YouTubePlayerEvent {
  target: YouTubePlayer
  data?: YouTubePlayerState
}

interface YouTubePlayerOptions {
  videoId: string
  playerVars: Record<string, string | number>
  events: {
    onReady: (event: YouTubePlayerEvent) => void
    onStateChange: (event: YouTubePlayerEvent) => void
    onError: (event: YouTubePlayerEvent) => void
  }
}

interface YouTubeNamespace {
  Player: new (element: HTMLElement, options: YouTubePlayerOptions) => YouTubePlayer
  PlayerState: {
    UNSTARTED: number
    ENDED: number
    PLAYING: number
    PAUSED: number
    BUFFERING: number
    CUED: number
  }
}

declare global {
  interface Window {
    YT?: YouTubeNamespace
    onYouTubeIframeAPIReady?: () => void
  }
}

interface TrailerFailure {
  code: number
  expiresAt: number
}

interface HeroGestureSample {
  x: number
  time: number
}

interface HeroGestureIntent {
  source: 'pointer' | 'touch'
  identifier: number
  startIndex: number
  startX: number
  startY: number
  samples: HeroGestureSample[]
}

const legacyTrailerFailureCacheKey = 'kodumedia.trailer-failures.v1'
const trailerFailureCacheKey = 'kodumedia.trailer-failures.v2'
const trailerFailureTTL = 30 * 24 * 60 * 60 * 1000
const permanentTrailerErrorCodes = new Set([2, 100, 101, 150])
const heroGestureSampleWindow = 100
const heroGestureProjectionMs = 90
let youtubeApiPromise: Promise<void> | null = null

function loadYouTubeApi(): Promise<void> {
  if (typeof window === 'undefined') return Promise.resolve()
  if (window.YT?.Player) return Promise.resolve()
  if (youtubeApiPromise) return youtubeApiPromise

  youtubeApiPromise = new Promise<void>((resolve, reject) => {
    const previousReady = window.onYouTubeIframeAPIReady
    window.onYouTubeIframeAPIReady = () => {
      previousReady?.()
      resolve()
    }

    const existing = document.querySelector<HTMLScriptElement>('script[data-kodumedia-youtube-api]')
    if (existing) {
      existing.addEventListener('error', () => reject(new Error('YouTube player API failed to load')), { once: true })
      return
    }

    const script = document.createElement('script')
    script.src = 'https://www.youtube.com/iframe_api'
    script.async = true
    script.dataset.kodumediaYoutubeApi = 'true'
    script.addEventListener('error', () => reject(new Error('YouTube player API failed to load')), { once: true })
    document.head.appendChild(script)
  })

  return youtubeApiPromise
}

function shouldAutoplayTrailer(): boolean {
  if (typeof window === 'undefined') return false
  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return false
  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection
  return connection?.saveData !== true
}

function heroMeta(item: MediaItem): string[] {
  const values: string[] = []
  if (item.year) values.push(String(item.year))
  if (item.metadata.genres?.length) values.push(item.metadata.genres.slice(0, 2).join(' · '))
  if (item.metadata.imdbRating) values.push(`IMDb ${item.metadata.imdbRating.toFixed(1)}`)
  return values
}

function normalizedTrailerCandidates(item: MediaItem): TrailerCandidate[] {
  const result: TrailerCandidate[] = []
  const seen = new Set<string>()
  const append = (candidate: TrailerCandidate | undefined) => {
    const youtubeKey = candidate?.youtubeKey?.trim() ?? ''
    if (!youtubeKey || seen.has(youtubeKey)) return
    seen.add(youtubeKey)
    result.push({ youtubeKey, name: candidate?.name?.trim() || undefined })
  }
  append(item.metadata.trailerYoutubeKey ? { youtubeKey: item.metadata.trailerYoutubeKey, name: item.metadata.trailerName } : undefined)
  for (const candidate of item.metadata.trailerCandidates ?? []) append(candidate)
  return result
}

function readTrailerFailures(): Record<string, TrailerFailure> {
  if (typeof window === 'undefined') return {}
  try {
    // Version 1 cached every player, API and network failure for 30 days. Drop it
    // instead of carrying its false positives into the corrected cache.
    window.localStorage.removeItem(legacyTrailerFailureCacheKey)
    const value = JSON.parse(window.localStorage.getItem(trailerFailureCacheKey) || '{}') as Record<string, TrailerFailure>
    const now = Date.now()
    let changed = false
    for (const [key, failure] of Object.entries(value)) {
      if (
        !failure
        || !Number.isFinite(failure.code)
        || !permanentTrailerErrorCodes.has(failure.code)
        || !Number.isFinite(failure.expiresAt)
        || failure.expiresAt <= now
      ) {
        delete value[key]
        changed = true
      }
    }
    if (changed) window.localStorage.setItem(trailerFailureCacheKey, JSON.stringify(value))
    return value
  } catch {
    return {}
  }
}

function trailerIsCachedFailure(key: string): boolean {
  return (readTrailerFailures()[key]?.expiresAt ?? 0) > Date.now()
}

function markTrailerFailure(key: string, code: number | undefined) {
  if (typeof window === 'undefined' || !key || !code || !permanentTrailerErrorCodes.has(code)) return
  try {
    const values = readTrailerFailures()
    values[key] = { code, expiresAt: Date.now() + trailerFailureTTL }
    window.localStorage.setItem(trailerFailureCacheKey, JSON.stringify(values))
  } catch {
    // A blocked localStorage must not prevent trailer fallback.
  }
}

function markTrailerWorking(key: string) {
  if (typeof window === 'undefined' || !key) return
  try {
    const values = readTrailerFailures()
    if (!(key in values)) return
    delete values[key]
    window.localStorage.setItem(trailerFailureCacheKey, JSON.stringify(values))
  } catch {
    // A blocked localStorage must not affect playback.
  }
}

function nextTrailerIndex(candidates: TrailerCandidate[], start: number): number {
  for (let index = Math.max(0, start); index < candidates.length; index += 1) {
    const candidate = candidates[index]
    if (candidate && !trailerIsCachedFailure(candidate.youtubeKey)) return index
  }
  return candidates.length
}

function warmHeroArtwork(items: MediaItem[]) {
  if (typeof Image === 'undefined') return
  const urls = items
    .flatMap((candidate) => [candidate.metadata.backdropUrl, candidate.metadata.logoUrl])
    .filter((value): value is string => Boolean(value))
  const run = () => {
    for (const url of urls) {
      const image = new Image()
      image.referrerPolicy = 'no-referrer'
      image.src = url
    }
  }
  const idleWindow = window as Window & {
    requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number
  }
  if (typeof idleWindow.requestIdleCallback === 'function') {
    idleWindow.requestIdleCallback(run, { timeout: 1200 })
  } else {
    window.setTimeout(run, 0)
  }
}

export const FeaturedHero = memo(function FeaturedHero({
  items,
  onOpen,
  onActiveChange,
}: {
  items: MediaItem[]
  onOpen: (item: MediaItem) => void
  onActiveChange?: (item: MediaItem) => void
}) {
  const containerRef = useRef<HTMLElement | null>(null)
  const gestureIntentRef = useRef<HeroGestureIntent | null>(null)
  const playerMountRef = useRef<HTMLDivElement | null>(null)
  const playerRef = useRef<YouTubePlayer | null>(null)
  const playbackWatchdogRef = useRef<number>(0)
  const mutedRef = useRef(true)
  const [activeIndex, setActiveIndex] = useState(0)
  const [trailerIndex, setTrailerIndex] = useState(0)
  const [inView, setInView] = useState(false)
  const [playerReady, setPlayerReady] = useState(false)
  const [muted, setMuted] = useState(true)
  const [autoplayActive, setAutoplayActive] = useState(true)
  const autoplayAllowed = useMemo(shouldAutoplayTrailer, [])
  const hasCarousel = items.length > 1
  const [carouselRef, carouselApi] = useEmblaCarousel({
    align: 'start',
    containScroll: false,
    dragFree: false,
    duration: 26,
    loop: hasCarousel,
    skipSnaps: false,
    watchDrag: hasCarousel,
  })
  const item = items[Math.min(activeIndex, Math.max(0, items.length - 1))]
  const trailerCandidates = useMemo(() => item ? normalizedTrailerCandidates(item) : [], [item])
  const trailer = trailerCandidates[trailerIndex]
  const trailerKey = trailer?.youtubeKey ?? ''

  const selectItem = useCallback((nextIndex: number) => {
    const count = items.length
    if (count <= 1) return
    const wrappedIndex = ((nextIndex % count) + count) % count
    setAutoplayActive(true)
    setActiveIndex(wrappedIndex)
    carouselApi?.scrollTo(wrappedIndex)
  }, [carouselApi, items.length])

  useEffect(() => {
    if (activeIndex < items.length) return
    setActiveIndex(Math.max(0, items.length - 1))
  }, [activeIndex, items.length])

  const itemSignature = useMemo(() => items.map((candidate) => candidate.id).join('|'), [items])

  useEffect(() => {
    if (!carouselApi) return
    const syncSelectedSlide = () => {
      setActiveIndex(carouselApi.selectedScrollSnap())
      setAutoplayActive(true)
    }
    syncSelectedSlide()
    carouselApi.on('select', syncSelectedSlide)
    carouselApi.on('reInit', syncSelectedSlide)
    return () => {
      carouselApi.off('select', syncSelectedSlide)
      carouselApi.off('reInit', syncSelectedSlide)
    }
  }, [carouselApi])

  useEffect(() => {
    warmHeroArtwork(items.slice(0, 4))
  }, [itemSignature, items])

  useEffect(() => {
    setAutoplayActive(true)
  }, [itemSignature])

  useEffect(() => {
    if (item) onActiveChange?.(item)
  }, [item, onActiveChange])

  useEffect(() => {
    const element = containerRef.current
    if (!element) return
    if (typeof IntersectionObserver === 'undefined') {
      setInView(true)
      return
    }
    const observer = new IntersectionObserver(
      ([entry]) => setInView(Boolean(entry?.isIntersecting && entry.intersectionRatio >= 0.45)),
      { threshold: [0, 0.45, 0.8] },
    )
    observer.observe(element)
    return () => observer.disconnect()
  }, [item?.id])

  useEffect(() => {
    const nextIndex = nextTrailerIndex(trailerCandidates, 0)
    setTrailerIndex(nextIndex)
    setPlayerReady(false)
    mutedRef.current = true
    setMuted(true)
  }, [trailerCandidates])

  const failCurrentTrailer = useCallback((errorCode?: number) => {
    if (!trailerKey) return
    markTrailerFailure(trailerKey, errorCode)
    setPlayerReady(false)
    mutedRef.current = true
    setMuted(true)
    setTrailerIndex((current) => nextTrailerIndex(trailerCandidates, current + 1))
  }, [trailerCandidates, trailerKey])

  const advanceAfterTrailer = useCallback(() => {
    setPlayerReady(false)
    mutedRef.current = true
    setMuted(true)
    if (activeIndex < items.length - 1) {
      selectItem(activeIndex + 1)
      return
    }
    setAutoplayActive(false)
  }, [activeIndex, items.length, selectItem])

  useEffect(() => {
    if (!trailerKey || !autoplayAllowed || !autoplayActive || !inView || !playerMountRef.current) return
    let cancelled = false
    let startTimer = window.setTimeout(() => {
      void loadYouTubeApi()
        .then(() => {
          if (cancelled || !window.YT?.Player || !playerMountRef.current) return
          playerRef.current?.destroy()
          playerMountRef.current.replaceChildren()
          const host = document.createElement('div')
          playerMountRef.current.appendChild(host)
          playerRef.current = new window.YT.Player(host, {
            videoId: trailerKey,
            playerVars: {
              autoplay: 0,
              controls: 0,
              disablekb: 1,
              enablejsapi: 1,
              fs: 0,
              iv_load_policy: 3,
              loop: 0,
              modestbranding: 1,
              origin: window.location.origin,
              playsinline: 1,
              rel: 0,
            },
            events: {
              onReady: ({ target }) => {
                if (cancelled) return
                target.mute()
                target.playVideo()
                mutedRef.current = true
                setMuted(true)
                window.clearTimeout(playbackWatchdogRef.current)
                playbackWatchdogRef.current = window.setTimeout(() => {
                  if (cancelled) return
                  const state = target.getPlayerState()
                  if (state !== window.YT?.PlayerState.PLAYING) failCurrentTrailer()
                }, 8000)
              },
              onStateChange: ({ target, data }) => {
                if (cancelled) return
                if (data === window.YT?.PlayerState.PLAYING) {
                  window.clearTimeout(playbackWatchdogRef.current)
                  markTrailerWorking(trailerKey)
                  setPlayerReady(true)
                }
                if (data === window.YT?.PlayerState.ENDED) advanceAfterTrailer()
              },
              onError: ({ data }) => {
                if (!cancelled) failCurrentTrailer(data)
              },
            },
          })
        })
        .catch(() => {
          if (!cancelled) failCurrentTrailer()
        })
    }, 850)

    return () => {
      cancelled = true
      window.clearTimeout(startTimer)
      startTimer = 0
      window.clearTimeout(playbackWatchdogRef.current)
      playbackWatchdogRef.current = 0
      playerRef.current?.destroy()
      playerRef.current = null
      playerMountRef.current?.replaceChildren()
      setPlayerReady(false)
    }
  }, [advanceAfterTrailer, autoplayActive, autoplayAllowed, failCurrentTrailer, inView, item?.id, trailerKey])

  useEffect(() => {
    const handleVisibility = () => {
      if (document.hidden) {
        playerRef.current?.pauseVideo()
      } else if (inView && autoplayActive) {
        if (mutedRef.current) playerRef.current?.mute()
        else playerRef.current?.unMute()
        playerRef.current?.playVideo()
      }
    }
    document.addEventListener('visibilitychange', handleVisibility)
    return () => document.removeEventListener('visibilitychange', handleVisibility)
  }, [autoplayActive, inView])

  if (!item) return null

  const setAudioMuted = (nextMuted: boolean) => {
    if (nextMuted) playerRef.current?.mute()
    else {
      playerRef.current?.unMute()
      playerRef.current?.playVideo()
    }
    mutedRef.current = nextMuted
    setMuted(nextMuted)
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (!hasCarousel || event.defaultPrevented) return
    if (event.key === 'ArrowLeft') {
      event.preventDefault()
      selectItem(activeIndex - 1)
    } else if (event.key === 'ArrowRight') {
      event.preventDefault()
      selectItem(activeIndex + 1)
    }
  }

  const startGestureIntent = (source: HeroGestureIntent['source'], identifier: number, x: number, y: number, time: number) => {
    if (!hasCarousel) return
    gestureIntentRef.current = {
      source,
      identifier,
      startIndex: carouselApi?.selectedScrollSnap() ?? activeIndex,
      startX: x,
      startY: y,
      samples: [{ x, time }],
    }
  }

  const updateGestureIntent = (source: HeroGestureIntent['source'], identifier: number, x: number, time: number) => {
    const intent = gestureIntentRef.current
    if (!intent || intent.source !== source || intent.identifier !== identifier) return
    intent.samples.push({ x, time })
    const cutoff = time - heroGestureSampleWindow
    while (intent.samples.length > 2 && (intent.samples[1]?.time ?? time) < cutoff) intent.samples.shift()
  }

  const completeGestureIntent = (
    source: HeroGestureIntent['source'],
    identifier: number,
    x: number,
    y: number,
    time: number,
  ) => {
    const intent = gestureIntentRef.current
    if (!intent || intent.source !== source || intent.identifier !== identifier) return
    updateGestureIntent(source, identifier, x, time)
    gestureIntentRef.current = null

    const deltaX = x - intent.startX
    const deltaY = y - intent.startY
    const firstSample = intent.samples[0]
    const lastSample = intent.samples.at(-1)
    const elapsed = Math.max(1, (lastSample?.time ?? time) - (firstSample?.time ?? time))
    const velocityX = ((lastSample?.x ?? x) - (firstSample?.x ?? x)) / elapsed
    const projectedX = deltaX + velocityX * heroGestureProjectionMs
    const width = containerRef.current?.clientWidth ?? 0
    const distanceThreshold = Math.min(64, Math.max(44, width * 0.14))
    const horizontallyIntentional = Math.abs(deltaX) > Math.abs(deltaY) * 1.05
    const commits = Math.abs(deltaX) >= distanceThreshold || Math.abs(projectedX) >= distanceThreshold * 1.35
    if (!horizontallyIntentional || !commits) return

    const direction = Math.abs(deltaX) >= distanceThreshold ? deltaX : projectedX
    const targetIndex = ((intent.startIndex + (direction < 0 ? 1 : -1)) % items.length + items.length) % items.length
    setAutoplayActive(true)
    setActiveIndex(targetIndex)
    carouselApi?.scrollTo(targetIndex)
  }

  const touchByIdentifier = (touches: ReactTouchEvent<HTMLElement>['changedTouches'], identifier: number) => {
    for (let index = 0; index < touches.length; index += 1) {
      const touch = touches.item?.(index) ?? touches[index]
      if (touch?.identifier === identifier) return touch
    }
    return null
  }
  const title = item.title

  return (
    <section
      ref={containerRef}
      className={`featured-hero ${playerReady ? 'featured-hero--playing' : ''}`}
      aria-label={`${tr('Esiletõstetud')}: ${title}`}
      aria-roledescription={hasCarousel ? tr('Karussell') : undefined}
      onKeyDown={handleKeyDown}
      onPointerDown={(event: ReactPointerEvent<HTMLElement>) => {
        if (event.pointerType !== 'touch' && event.pointerType !== 'pen') return
        startGestureIntent('pointer', event.pointerId, event.clientX, event.clientY, event.timeStamp)
      }}
      onPointerMove={(event: ReactPointerEvent<HTMLElement>) => {
        updateGestureIntent('pointer', event.pointerId, event.clientX, event.timeStamp)
      }}
      onPointerUp={(event: ReactPointerEvent<HTMLElement>) => {
        completeGestureIntent('pointer', event.pointerId, event.clientX, event.clientY, event.timeStamp)
      }}
      onPointerCancel={(event: ReactPointerEvent<HTMLElement>) => {
        if (gestureIntentRef.current?.source === 'pointer' && gestureIntentRef.current.identifier === event.pointerId) gestureIntentRef.current = null
      }}
      onTouchStart={(event: ReactTouchEvent<HTMLElement>) => {
        if (event.touches.length !== 1) return
        const touch = event.touches.item?.(0) ?? event.touches[0]
        if (touch) startGestureIntent('touch', touch.identifier, touch.clientX, touch.clientY, event.timeStamp)
      }}
      onTouchMove={(event: ReactTouchEvent<HTMLElement>) => {
        const intent = gestureIntentRef.current
        if (!intent || intent.source !== 'touch') return
        const touch = touchByIdentifier(event.changedTouches, intent.identifier)
        if (touch) updateGestureIntent('touch', intent.identifier, touch.clientX, event.timeStamp)
      }}
      onTouchEnd={(event: ReactTouchEvent<HTMLElement>) => {
        const intent = gestureIntentRef.current
        if (!intent || intent.source !== 'touch') return
        const touch = touchByIdentifier(event.changedTouches, intent.identifier)
        if (touch) completeGestureIntent('touch', intent.identifier, touch.clientX, touch.clientY, event.timeStamp)
      }}
      onTouchCancel={() => {
        if (gestureIntentRef.current?.source === 'touch') gestureIntentRef.current = null
      }}
    >
      <div className="featured-hero__viewport" ref={carouselRef}>
        <div className="featured-hero__slides">
          {items.map((candidate, index) => {
            const candidateBackground = candidate.metadata.backdropUrl || candidate.metadata.posterUrl
            const candidateMeta = heroMeta(candidate)
            const candidateIsActive = index === activeIndex
            return (
              <article
                key={candidate.id}
                className="featured-hero__slide"
                aria-label={`${index + 1} / ${items.length}: ${candidate.title}`}
                aria-hidden={!candidateIsActive}
                inert={!candidateIsActive}
              >
                {candidateBackground ? (
                  candidateIsActive ? (
                    <img className="featured-hero__backdrop" src={candidateBackground} alt="" decoding="async" loading="eager" fetchPriority="high" referrerPolicy="no-referrer" draggable={false} />
                  ) : (
                    <img className="featured-hero__backdrop" src={candidateBackground} alt="" decoding="async" loading="lazy" fetchPriority="low" referrerPolicy="no-referrer" draggable={false} />
                  )
                ) : null}
                {candidateIsActive && trailerKey && autoplayAllowed && autoplayActive ? (
                  <div className="featured-hero__trailer" aria-hidden="true">
                    <div ref={playerMountRef} />
                  </div>
                ) : null}
                <div className="featured-hero__shade" aria-hidden="true" />

                <div className="featured-hero__content" aria-live={candidateIsActive ? 'polite' : 'off'}>
                  <span className="featured-hero__eyebrow"><Icon name="sparkles" size={15} />{tr('Esiletõstetud')}</span>
                  {candidate.metadata.logoUrl ? (
                    <img className="featured-hero__logo" src={candidate.metadata.logoUrl} alt={candidate.title} loading={candidateIsActive ? 'eager' : 'lazy'} referrerPolicy="no-referrer" draggable={false} />
                  ) : (
                    <h2>{candidate.title}</h2>
                  )}
                  {candidateMeta.length ? <div className="featured-hero__meta">{candidateMeta.map((value) => <span key={value}>{value}</span>)}</div> : null}
                  {candidate.metadata.summary ? <p>{candidate.metadata.summary}</p> : null}
                  <div className="featured-hero__actions">
                    <MediaPlayButton onClick={() => onOpen(candidate)}>{tr('Vaata kohe')}</MediaPlayButton>
                  </div>
                </div>
              </article>
            )
          })}
        </div>
      </div>

      {playerReady ? (
        <button
          type="button"
          className="featured-hero__sound"
          data-focusable="true"
          data-pressable="true"
          aria-label={muted ? tr('Lülita heli sisse') : tr('Lülita heli välja')}
          aria-pressed={!muted}
          onClick={() => setAudioMuted(!muted)}
        >
          <Icon name={muted ? 'mute' : 'volume'} size={20} />
        </button>
      ) : null}

      {hasCarousel ? (
        <div className="featured-hero__carousel-controls" aria-label={tr('Esiletõstetud sisu valik')}>
          <button
            type="button"
            className="featured-hero__arrow featured-hero__arrow--previous"
            data-focusable="true"
            data-pressable="true"
            aria-label={tr('Eelmine esiletõstetud sisu')}
            onClick={() => selectItem(activeIndex - 1)}
          >
            <Icon name="chevronRight" size={22} />
          </button>
          <div className="featured-hero__pagination" role="tablist" aria-label={tr('Esiletõstetud sisu')}>
            {items.map((candidate, index) => (
              <button
                key={candidate.id}
                type="button"
                role="tab"
                className="featured-hero__dot"
                data-focusable="true"
                data-pressable="true"
                aria-label={`${index + 1}: ${candidate.title}`}
                aria-selected={index === activeIndex}
                onClick={() => selectItem(index)}
              />
            ))}
          </div>
          <span className="featured-hero__counter" aria-hidden="true">{activeIndex + 1} / {items.length}</span>
          <button
            type="button"
            className="featured-hero__arrow"
            data-focusable="true"
            data-pressable="true"
            aria-label={tr('Järgmine esiletõstetud sisu')}
            onClick={() => selectItem(activeIndex + 1)}
          >
            <Icon name="chevronRight" size={22} />
          </button>
        </div>
      ) : null}
    </section>
  )
})
