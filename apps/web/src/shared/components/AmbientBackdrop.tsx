import { useCallback, useEffect, useRef, useState } from 'react'

interface AmbientBackdropProps {
  url?: string
  subtle?: boolean
}

type LayerIndex = 0 | 1

interface BackdropLayers {
  urls: [string, string]
  base: LayerIndex
  incoming: LayerIndex | null
  revealed: boolean
}

const decodedBackdropUrls = new Set<string>()
const pendingBackdropLoads = new Map<string, Promise<void>>()
const decodedBackdropLimit = 48
const backdropTransitionFallbackMs = 700
const ambientLayerIndexes: LayerIndex[] = [0, 1]

function otherLayer(index: LayerIndex): LayerIndex {
  return index === 0 ? 1 : 0
}

export function optimizeAmbientImageUrl(url: string): string {
  const trimmed = url.trim()
  if (!trimmed) return ''
  try {
    const parsed = new URL(trimmed)
    if (parsed.hostname !== 'image.tmdb.org') return trimmed
    parsed.pathname = parsed.pathname.replace(
      /\/t\/p\/(?:w780|w1280|w1920|original)\//,
      '/t/p/w500/',
    )
    return parsed.toString()
  } catch {
    return trimmed
  }
}

function rememberDecodedBackdrop(url: string) {
  decodedBackdropUrls.delete(url)
  decodedBackdropUrls.add(url)
  if (decodedBackdropUrls.size <= decodedBackdropLimit) return
  const oldest = decodedBackdropUrls.values().next().value
  if (oldest) decodedBackdropUrls.delete(oldest)
}

function preloadBackdrop(url: string): Promise<void> {
  if (decodedBackdropUrls.has(url)) return Promise.resolve()
  const pending = pendingBackdropLoads.get(url)
  if (pending) return pending

  const request = new Promise<void>((resolve, reject) => {
    const image = new Image()
    let settled = false
    const finish = () => {
      if (settled) return
      settled = true
      rememberDecodedBackdrop(url)
      resolve()
    }
    const decode = () => {
      if (typeof image.decode === 'function') void image.decode().catch(() => undefined).finally(finish)
      else finish()
    }
    image.decoding = 'async'
    image.onload = decode
    image.onerror = () => {
      if (settled) return
      settled = true
      reject(new Error(`Backdrop failed to load: ${url}`))
    }
    image.src = url
    if (image.complete && image.naturalWidth > 0) decode()
  })
  pendingBackdropLoads.set(url, request)
  void request.finally(() => pendingBackdropLoads.delete(url)).catch(() => undefined)
  return request
}

/**
 * Keeps two filtered artwork layers mounted and promoted for the component's
 * lifetime. A decoded replacement is written into the fully transparent buffer
 * before only that buffer fades over the settled base layer.
 */
export function AmbientBackdrop({ url, subtle = false }: AmbientBackdropProps) {
  const normalizedUrl = optimizeAmbientImageUrl(url ?? '')
  const [layers, setLayers] = useState<BackdropLayers>({
    urls: ['', ''],
    base: 0,
    incoming: null,
    revealed: true,
  })
  const requestedRef = useRef(normalizedUrl)
  const currentRef = useRef('')
  const baseIndexRef = useRef<LayerIndex>(0)
  const transitioningRef = useRef(false)
  const queuedReadyRef = useRef('')
  const revealFrameRef = useRef<number | null>(null)
  const queuedFrameRef = useRef<number | null>(null)
  const cleanupTimerRef = useRef<number>(0)
  const beginTransitionRef = useRef<(nextUrl: string) => void>(() => undefined)

  const finishTransition = useCallback((settledUrl: string, settledIndex: LayerIndex) => {
    if (currentRef.current !== settledUrl) return
    if (revealFrameRef.current !== null) {
      window.cancelAnimationFrame(revealFrameRef.current)
      revealFrameRef.current = null
    }
    window.clearTimeout(cleanupTimerRef.current)
    cleanupTimerRef.current = 0
    transitioningRef.current = false
    baseIndexRef.current = settledIndex
    setLayers((current) => (
      current.incoming === settledIndex && current.urls[settledIndex] === settledUrl
        ? { ...current, base: settledIndex, incoming: null, revealed: false }
        : current
    ))

    const queuedUrl = queuedReadyRef.current
    queuedReadyRef.current = ''
    if (queuedUrl && queuedUrl !== settledUrl && requestedRef.current === queuedUrl) {
      queuedFrameRef.current = window.requestAnimationFrame(() => {
        queuedFrameRef.current = null
        if (requestedRef.current === queuedUrl && currentRef.current !== queuedUrl) {
          beginTransitionRef.current(queuedUrl)
        }
      })
    }
  }, [])

  const beginTransition = useCallback((nextUrl: string) => {
    if (!nextUrl || nextUrl === currentRef.current) return

    if (!currentRef.current) {
      const initialIndex = baseIndexRef.current
      currentRef.current = nextUrl
      setLayers((current) => {
        const urls: [string, string] = [...current.urls]
        urls[initialIndex] = nextUrl
        return { urls, base: initialIndex, incoming: null, revealed: false }
      })
      return
    }

    const incomingIndex = otherLayer(baseIndexRef.current)
    currentRef.current = nextUrl
    transitioningRef.current = true
    setLayers((current) => {
      const urls: [string, string] = [...current.urls]
      urls[incomingIndex] = nextUrl
      return { ...current, urls, incoming: incomingIndex, revealed: false }
    })

    revealFrameRef.current = window.requestAnimationFrame(() => {
      revealFrameRef.current = window.requestAnimationFrame(() => {
        revealFrameRef.current = null
        setLayers((current) => (
          current.incoming === incomingIndex && current.urls[incomingIndex] === nextUrl
            ? { ...current, revealed: true }
            : current
        ))
        window.clearTimeout(cleanupTimerRef.current)
        cleanupTimerRef.current = window.setTimeout(
          () => finishTransition(nextUrl, incomingIndex),
          backdropTransitionFallbackMs,
        )
      })
    })
  }, [finishTransition])

  beginTransitionRef.current = beginTransition

  useEffect(() => {
    requestedRef.current = normalizedUrl
    if (
      !normalizedUrl
      || normalizedUrl === currentRef.current
      || normalizedUrl === queuedReadyRef.current
    ) return

    let cancelled = false
    void preloadBackdrop(normalizedUrl)
      .then(() => {
        if (cancelled || requestedRef.current !== normalizedUrl) return
        if (transitioningRef.current) {
          queuedReadyRef.current = normalizedUrl
          return
        }
        beginTransition(normalizedUrl)
      })
      .catch(() => undefined)

    return () => {
      cancelled = true
    }
  }, [beginTransition, normalizedUrl])

  useEffect(() => (
    () => {
      if (revealFrameRef.current !== null) window.cancelAnimationFrame(revealFrameRef.current)
      if (queuedFrameRef.current !== null) window.cancelAnimationFrame(queuedFrameRef.current)
      window.clearTimeout(cleanupTimerRef.current)
    }
  ), [])

  const backdropClassName = [
    'ambient-backdrop',
    subtle ? 'ambient-backdrop--subtle' : '',
  ].filter(Boolean).join(' ')

  return (
    <div className={backdropClassName} aria-hidden="true">
      {ambientLayerIndexes.map((index) => {
        const isBase = layers.base === index
        const isIncoming = layers.incoming === index
        const layerClassName = [
          'ambient-backdrop__layer',
          `ambient-backdrop__layer--slot-${index}`,
          isBase ? 'ambient-backdrop__layer--base' : '',
          isIncoming ? 'ambient-backdrop__layer--incoming' : '',
          isIncoming && layers.revealed ? 'ambient-backdrop__layer--revealed' : '',
        ].filter(Boolean).join(' ')

        return (
          <div
            key={index}
            className={layerClassName}
            style={layers.urls[index] ? { backgroundImage: `url(${layers.urls[index]})` } : undefined}
            onTransitionEnd={isIncoming ? (event) => {
              if (event.propertyName === 'opacity') finishTransition(layers.urls[index], index)
            } : undefined}
          />
        )
      })}
    </div>
  )
}
