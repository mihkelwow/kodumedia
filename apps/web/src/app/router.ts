import { useCallback, useEffect, useMemo, useState } from 'react'

export type Route =
  | { name: 'home' }
  | { name: 'library'; libraryId: string }
  | { name: 'settings' }
  | { name: 'notifications' }
  | { name: 'music'; artist?: string; albumId?: string; tracks?: boolean }
  | { name: 'notFound' }

export interface NavigateOptions {
  replace?: boolean
  scroll?: boolean
}

export function normalizePathname(pathname: string): string {
  const trimmed = pathname.trim()
  const withLeadingSlash = trimmed === '' ? '/' : trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  if (withLeadingSlash === '/') return '/'
  return withLeadingSlash.replace(/\/+$/, '') || '/'
}

function decodePathSegment(value: string): string | null {
  try {
    return decodeURIComponent(value)
  } catch {
    return null
  }
}

export function parsePath(pathname: string): Route {
  const normalized = normalizePathname(pathname)
  if (normalized === '/') return { name: 'home' }
  if (normalized === '/settings') return { name: 'settings' }
  if (normalized === '/notifications') return { name: 'notifications' }
  if (normalized === '/music') return { name: 'music' }
  if (normalized === '/music/tracks') return { name: 'music', tracks: true }
  const musicArtistMatch = normalized.match(/^\/music\/artist\/([^/]+)$/)
  if (musicArtistMatch?.[1]) {
    const artist = decodePathSegment(musicArtistMatch[1])
    if (artist) return { name: 'music', artist }
  }
  const musicAlbumMatch = normalized.match(/^\/music\/album\/([^/]+)$/)
  if (musicAlbumMatch?.[1]) {
    const albumId = decodePathSegment(musicAlbumMatch[1])
    if (albumId) return { name: 'music', albumId }
  }

  const libraryMatch = normalized.match(/^\/library\/([^/]+)$/)
  if (libraryMatch?.[1]) {
    const libraryId = decodePathSegment(libraryMatch[1])
    if (libraryId) return { name: 'library', libraryId }
  }
  return { name: 'notFound' }
}

function currentPathname(): string {
  return normalizePathname(window.location.pathname)
}

export function useRouter() {
  const [pathname, setPathname] = useState(currentPathname)

  useEffect(() => {
    const synchronizeWithBrowser = () => setPathname(currentPathname())
    window.addEventListener('popstate', synchronizeWithBrowser)
    return () => window.removeEventListener('popstate', synchronizeWithBrowser)
  }, [])

  const navigate = useCallback((target: string, options: NavigateOptions = {}) => {
    const url = new URL(target, window.location.href)
    if (url.origin !== window.location.origin) {
      window.location.assign(url.href)
      return
    }

    const nextPathname = normalizePathname(url.pathname)
    const nextLocation = `${nextPathname}${url.search}${url.hash}`
    const currentLocation = `${currentPathname()}${window.location.search}${window.location.hash}`
    if (nextLocation === currentLocation) return

    const method = options.replace ? 'replaceState' : 'pushState'
    window.history[method]({}, '', nextLocation)
    setPathname(nextPathname)

    if (options.scroll !== false) {
      window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
    }
  }, [])

  const route = useMemo(() => parsePath(pathname), [pathname])
  return { route, pathname, navigate }
}
