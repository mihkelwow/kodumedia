import { useEffect, useMemo, useRef, useState } from 'react'
import type { MediaItem } from '../api/types'
import { api } from '../api/client'
import { Button } from '../shared/components/Button'
import { EmptyState } from '../shared/components/EmptyState'
import { Icon } from '../shared/components/Icon'
import { Spinner } from '../shared/components/Spinner'
import { MediaRow } from '../features/media/MediaRow'
import { FeaturedHero } from '../features/media/FeaturedHero'
import { tr } from '../i18n/i18n'

interface HomeData {
  continueWatching: MediaItem[]
  movies: MediaItem[]
  series: MediaItem[]
  episodes: MediaItem[]
  releases: MediaItem[]
  rated: MediaItem[]
  horror: MediaItem[]
  comedy: MediaItem[]
  actionAdventure: MediaItem[]
  romance: MediaItem[]
  animation: MediaItem[]
  scienceFiction: MediaItem[]
}

type HomeCategoryKey = 'horror' | 'comedy' | 'actionAdventure' | 'romance' | 'animation' | 'scienceFiction'

type HomeCategories = Pick<HomeData, HomeCategoryKey>

interface HomeCategoryDefinition {
  key: HomeCategoryKey
  genres: string[]
}

const HOME_CATEGORY_LIMIT = 18
const HOME_CATEGORY_CANDIDATE_LIMIT = 100
const HOME_CATEGORY_DEFINITIONS: HomeCategoryDefinition[] = [
  { key: 'horror', genres: ['Horror', 'Õudus'] },
  { key: 'comedy', genres: ['Comedy', 'Komöödia'] },
  { key: 'actionAdventure', genres: ['Action', 'Märul', 'Adventure', 'Seiklus'] },
  { key: 'romance', genres: ['Romance', 'Romantika'] },
  { key: 'animation', genres: ['Animation', 'Animatsioon'] },
  { key: 'scienceFiction', genres: ['Science Fiction', 'Sci-Fi', 'Ulme'] },
]

const emptyCategories: HomeCategories = {
  horror: [],
  comedy: [],
  actionAdventure: [],
  romance: [],
  animation: [],
  scienceFiction: [],
}

const emptyData: HomeData = {
  continueWatching: [],
  movies: [],
  series: [],
  episodes: [],
  releases: [],
  rated: [],
  ...emptyCategories,
}

const HOME_CACHE_KEY = 'kodumedia.home.v2'
const HOME_CACHE_MAX_AGE = 7 * 24 * 60 * 60 * 1000
let homeMemoryCache: HomeData | null = null
let pendingHomeCache: HomeData | null = null
let cancelScheduledHomeCacheWrite: (() => void) | null = null

interface StoredHomeData {
  storedAt: number
  data: HomeData
}

function hasHomeContent(data: HomeData): boolean {
  return data.movies.length > 0 || data.series.length > 0 || data.episodes.length > 0 || data.rated.length > 0 || data.releases.length > 0
}

function parseHomeCache(value: string | null): HomeData | null {
  if (!value) return null
  try {
    const stored = JSON.parse(value) as StoredHomeData | null
    if (!stored || !Number.isFinite(stored.storedAt) || Date.now() - stored.storedAt > HOME_CACHE_MAX_AGE) return null
    if (!stored.data || !Array.isArray(stored.data.movies) || !Array.isArray(stored.data.series)) return null
    return { ...emptyData, ...stored.data }
  } catch {
    return null
  }
}

function readHomeCacheFrom(storageName: 'sessionStorage' | 'localStorage'): HomeData | null {
  try {
    return parseHomeCache(window[storageName].getItem(HOME_CACHE_KEY))
  } catch {
    return null
  }
}

function readHomeCache(): HomeData | null {
  if (homeMemoryCache) return homeMemoryCache
  if (typeof window === 'undefined') return null
  const cached = readHomeCacheFrom('sessionStorage') ?? readHomeCacheFrom('localStorage')
  if (!cached) return null
  homeMemoryCache = cached
  return cached
}

function writeHomeCache(data: HomeData) {
  homeMemoryCache = data
  if (typeof window === 'undefined') return
  pendingHomeCache = data
  if (cancelScheduledHomeCacheWrite) return

  const persist = () => {
    cancelScheduledHomeCacheWrite = null
    const next = pendingHomeCache
    pendingHomeCache = null
    if (!next) return
    const serialized = JSON.stringify({ storedAt: Date.now(), data: next } satisfies StoredHomeData)
    for (const storageName of ['sessionStorage', 'localStorage'] as const) {
      try {
        window[storageName].setItem(HOME_CACHE_KEY, serialized)
      } catch {
        // A full or blocked storage area must not delay the homepage.
      }
    }
  }

  const idleWindow = window as Window & {
    requestIdleCallback?: (callback: () => void, options?: { timeout: number }) => number
    cancelIdleCallback?: (handle: number) => void
  }
  if (typeof idleWindow.requestIdleCallback === 'function') {
    const handle = idleWindow.requestIdleCallback(persist, { timeout: 1500 })
    cancelScheduledHomeCacheWrite = () => idleWindow.cancelIdleCallback?.(handle)
    return
  }
  const handle = window.setTimeout(persist, 60)
  cancelScheduledHomeCacheWrite = () => window.clearTimeout(handle)
}

export function resetHomeCacheForTests() {
  cancelScheduledHomeCacheWrite?.()
  cancelScheduledHomeCacheWrite = null
  pendingHomeCache = null
  homeMemoryCache = null
  if (typeof window === 'undefined') return
  for (const storageName of ['sessionStorage', 'localStorage'] as const) {
    try {
      window[storageName].removeItem(HOME_CACHE_KEY)
    } catch {
      // Tests and privacy modes may expose a disabled storage implementation.
    }
  }
}

function preloadImage(url: string | undefined): Promise<void> {
  if (!url || typeof Image === 'undefined') return Promise.resolve()
  if (typeof navigator !== 'undefined' && navigator.userAgent.toLowerCase().includes('jsdom')) return Promise.resolve()
  return new Promise((resolve) => {
    const image = new Image()
    let settled = false
    const finish = () => {
      if (settled) return
      settled = true
      resolve()
    }
    const timeout = window.setTimeout(finish, 180)
    image.onload = () => {
      window.clearTimeout(timeout)
      if (typeof image.decode === 'function') void image.decode().catch(() => undefined).finally(finish)
      else finish()
    }
    image.onerror = () => {
      window.clearTimeout(timeout)
      finish()
    }
    image.referrerPolicy = 'no-referrer'
    image.src = url
    if (image.complete) {
      window.clearTimeout(timeout)
      if (typeof image.decode === 'function') void image.decode().catch(() => undefined).finally(finish)
      else finish()
    }
  })
}

function normalizeGenre(value: string): string {
  return value.trim().toLowerCase()
}

const categoryByGenre = new Map<string, HomeCategoryKey>()
for (const category of HOME_CATEGORY_DEFINITIONS) {
  for (const genre of category.genres) categoryByGenre.set(normalizeGenre(genre), category.key)
}
const animationGenres = new Set(HOME_CATEGORY_DEFINITIONS.find((category) => category.key === 'animation')?.genres.map(normalizeGenre) ?? [])

function uniqueItems(...collections: MediaItem[][]): MediaItem[] {
  const seen = new Set<string>()
  const result: MediaItem[] = []
  for (const collection of collections) {
    for (const item of collection) {
      if (seen.has(item.id)) continue
      seen.add(item.id)
      result.push(item)
    }
  }
  return result
}

function homeCategoryFor(item: MediaItem): HomeCategoryKey | null {
  const genres = item.metadata.genres ?? []

  // Animation is intentionally exclusive even when another genre has a higher
  // position in the source metadata.
  if (genres.some((genre) => animationGenres.has(normalizeGenre(genre)))) return 'animation'

  // TMDb and the local catalog preserve genre order. The first homepage genre
  // in that list acts as the film's strongest category weight.
  for (const genre of genres) {
    const category = categoryByGenre.get(normalizeGenre(genre))
    if (category && category !== 'animation') return category
  }
  return null
}

function categoryRating(item: MediaItem): number {
  return item.metadata.imdbRating ?? 0
}

function distributeMovieCategories(candidates: MediaItem[]): HomeCategories {
  const result: HomeCategories = {
    horror: [],
    comedy: [],
    actionAdventure: [],
    romance: [],
    animation: [],
    scienceFiction: [],
  }

  const sorted = uniqueItems(candidates).sort((left, right) => {
    const ratingDifference = categoryRating(right) - categoryRating(left)
    if (ratingDifference !== 0) return ratingDifference
    return left.sortTitle.localeCompare(right.sortTitle)
  })

  for (const item of sorted) {
    const category = homeCategoryFor(item)
    if (!category || result[category].length >= HOME_CATEGORY_LIMIT) continue
    result[category].push(item)
  }
  return result
}

async function loadMovieCategories(): Promise<HomeCategories> {
  const categoryPages = await Promise.all(
    HOME_CATEGORY_DEFINITIONS.map(async (category) => {
      try {
        const page = await api.media.list({
          type: 'movie',
          genre: category.genres,
          sort: 'rating',
          order: 'desc',
          limit: HOME_CATEGORY_CANDIDATE_LIMIT,
        })
        return page.items
      } catch {
        return []
      }
    }),
  )
  return distributeMovieCategories(uniqueItems(...categoryPages))
}

type PrimaryHomeData = Pick<HomeData, 'movies' | 'series' | 'releases' | 'rated'>
type SecondaryHomeData = Pick<HomeData, 'continueWatching' | 'episodes'>

async function loadPrimaryHomeData(): Promise<PrimaryHomeData> {
  const [movies, series, releases, rated] = await Promise.all([
    api.media.list({ type: 'movie', sort: 'modifiedAt', order: 'desc', limit: 18 }),
    api.media.list({ type: 'series', sort: 'modifiedAt', order: 'desc', limit: 18 }),
    api.media.list({ type: ['movie', 'series'], sort: 'releaseDate', order: 'desc', limit: 18 }),
    api.media.list({ type: ['movie', 'series'], sort: 'rating', order: 'desc', limit: 18 }),
  ])
  return {
    movies: movies.items,
    series: series.items,
    releases: releases.items,
    rated: rated.items.filter((item) => Boolean(item.metadata.imdbRating)),
  }
}

async function loadSecondaryHomeData(): Promise<SecondaryHomeData> {
  const [continueWatching, episodes] = await Promise.all([
    api.watch.continueWatching(5).catch(() => []),
    api.media.list({ type: 'episode', sort: 'releaseDate', order: 'desc', limit: 18 }),
  ])
  return { continueWatching, episodes: episodes.items }
}

function isFeatureEligible(item: MediaItem): boolean {
  if (item.type !== 'movie' && item.type !== 'series') return false
  if (!hasTrailerCandidate(item)) return false
  if (!item.watch) return true
  return !item.watch.completed && item.watch.progress < 0.05
}

function hasTrailerCandidate(item: MediaItem): boolean {
  if (item.metadata.trailerYoutubeKey?.trim()) return true
  return Boolean(item.metadata.trailerCandidates?.some((candidate) => candidate.youtubeKey?.trim()))
}

function featureRank(item: MediaItem): number {
  let score = 0
  if (hasTrailerCandidate(item)) score += 10_000
  if (item.metadata.logoUrl) score += 1_500
  if (item.metadata.backdropUrl) score += 600
  if (item.metadata.imdbRating) score += item.metadata.imdbRating * 25
  if (item.releaseDate) {
    const timestamp = Date.parse(item.releaseDate)
    if (Number.isFinite(timestamp)) score += timestamp / 1_000_000_000_000
  }
  return score
}

function chooseFeatured(data: HomeData): MediaItem[] {
  const candidates = uniqueItems(data.rated, data.releases, data.movies, data.series).filter(isFeatureEligible)
  candidates.sort((left, right) => featureRank(right) - featureRank(left))
  if (!candidates.length) return []

  const pool = candidates.slice(0, 20)
  const dateKey = new Date().toISOString().slice(0, 10)
  let hash = 0
  for (const character of dateKey) hash = ((hash << 5) - hash + character.charCodeAt(0)) | 0
  const offset = Math.abs(hash) % pool.length
  return [...pool.slice(offset), ...pool.slice(0, offset)].slice(0, 10)
}

export function HomePage({ onOpen, onNavigate, refreshToken = 0 }: { onOpen: (item: MediaItem) => void; onNavigate: (path: string) => void; refreshToken?: number }) {
  const initialDataRef = useRef<HomeData | null>(null)
  if (!initialDataRef.current) initialDataRef.current = readHomeCache() ?? emptyData
  const [data, setData] = useState<HomeData>(initialDataRef.current)
  const [loading, setLoading] = useState(() => !hasHomeContent(initialDataRef.current ?? emptyData))
  const [error, setError] = useState('')
  const dataRef = useRef(data)
  dataRef.current = data

  useEffect(() => {
    let cancelled = false
    const hadCachedContent = hasHomeContent(dataRef.current)
    if (!hadCachedContent) setLoading(true)
    setError('')

    void loadPrimaryHomeData()
      .then(async (primary) => {
        const next: HomeData = { ...dataRef.current, ...primary }
        const featured = chooseFeatured(next)
        if (!hadCachedContent) {
          const firstBackground = featured[0]?.metadata.backdropUrl || featured[0]?.metadata.posterUrl
          await preloadImage(firstBackground)
        }
        if (cancelled) return
        dataRef.current = next
        setData(next)
        writeHomeCache(next)
        setLoading(false)
      })
      .catch((caught) => {
        if (cancelled) return
        if (!hadCachedContent) setError(caught instanceof Error ? caught.message : tr('Meedia laadimine ebaõnnestus.'))
        setLoading(false)
      })

    void loadSecondaryHomeData().then((secondary) => {
      if (cancelled) return
      const next = { ...dataRef.current, ...secondary }
      dataRef.current = next
      setData(next)
      writeHomeCache(next)
    })

    void loadMovieCategories().then((categories) => {
      if (cancelled) return
      const next = { ...dataRef.current, ...categories }
      dataRef.current = next
      setData(next)
      writeHomeCache(next)
    })

    return () => { cancelled = true }
  }, [refreshToken])

  const featuredItems = useMemo(() => chooseFeatured(data), [data])
  const empty = !loading && !error && data.movies.length === 0 && data.series.length === 0 && data.episodes.length === 0

  return (
    <div className="page page--home cinematic-page">
      <h1 className="sr-only">{tr('Avaleht')}</h1>

      {loading && !hasHomeContent(data) ? <div className="page-loader"><Spinner label={tr('Laen avalehte')} /></div> : null}
      {error ? <div className="notice notice--error"><Icon name="alert" />{error}</div> : null}

      {empty ? (
        <EmptyState
          icon="server"
          title={tr('Meediakogu ootab sisu')}
          description={tr('Lisa Filmide, Seriaalide või Muusika kogule serveris olev kaust. Rakendus tuvastab sisu ja rühmitab seriaalid automaatselt.')}
          action={<Button variant="primary" icon="plus" onClick={() => onNavigate('/settings')}>{tr('Lisa meediakaust')}</Button>}
        />
      ) : null}

      {!empty && hasHomeContent(data) ? (
        <div className="home-rows">
          {featuredItems.length ? <FeaturedHero items={featuredItems} onOpen={onOpen} /> : null}
          <MediaRow title={tr('Jätka vaatamist')} items={data.continueWatching} onOpen={onOpen} variant="landscape" />
          <MediaRow title={tr('Viimati lisatud filmid')} items={data.movies} onOpen={onOpen} />
          <MediaRow title={tr('Viimati lisatud seriaalid')} items={data.series} onOpen={onOpen} />
          <MediaRow title={tr('Uued episoodid')} items={data.episodes} onOpen={onOpen} />
          <MediaRow title={tr('Kõrgeima IMDb skooriga')} items={data.rated} onOpen={onOpen} />
          <MediaRow title={tr('Õudusfilmid')} items={data.horror} onOpen={onOpen} />
          <MediaRow title={tr('Komöödiad')} items={data.comedy} onOpen={onOpen} />
          <MediaRow title={tr('Märul ja seiklus')} items={data.actionAdventure} onOpen={onOpen} />
          <MediaRow title={tr('Romantilised filmid')} items={data.romance} onOpen={onOpen} />
          <MediaRow title={tr('Animatsioon')} items={data.animation} onOpen={onOpen} />
          <MediaRow title={tr('Ulmefilmid')} items={data.scienceFiction} onOpen={onOpen} />
        </div>
      ) : null}
    </div>
  )
}
