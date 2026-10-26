import type { Route } from './router'

const staticLibraryIds = new Set(['library-movies', 'library-series'])

/**
 * The lightweight brand backdrop belongs only to collection-level surfaces.
 * Media and music detail routes keep their existing artwork-driven backgrounds.
 */
export function usesStaticAppBackdrop(route: Route): boolean {
  if (route.name === 'home' || route.name === 'settings') return true
  if (route.name === 'library') return staticLibraryIds.has(route.libraryId)
  if (route.name === 'music') return !route.artist && !route.albumId && !route.tracks
  return false
}
