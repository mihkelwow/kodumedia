import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { normalizePathname, parsePath, useRouter } from './router'

describe('router path parsing', () => {
  it('normalizes trailing slashes', () => {
    expect(normalizePathname('/settings/')).toBe('/settings')
    expect(parsePath('/settings/')).toEqual({ name: 'settings' })
    expect(parsePath('/notifications/')).toEqual({ name: 'notifications' })
    expect(parsePath('/music/')).toEqual({ name: 'music' })
    expect(parsePath('/music/tracks/')).toEqual({ name: 'music', tracks: true })
    expect(parsePath('/music/artist/Miles%20Davis')).toEqual({ name: 'music', artist: 'Miles Davis' })
    expect(parsePath('/music/album/album-1')).toEqual({ name: 'music', albumId: 'album-1' })
  })

  it('decodes a library identifier', () => {
    expect(parsePath('/library/my%20movies')).toEqual({ name: 'library', libraryId: 'my movies' })
  })

  it('returns notFound for malformed percent encoding instead of throwing', () => {
    expect(parsePath('/library/%E0%A4%A')).toEqual({ name: 'notFound' })
  })
})

describe('browser history integration', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/')
    vi.spyOn(window, 'scrollTo').mockImplementation(() => undefined)
  })

  it('pushes a route and synchronizes when the browser emits popstate', () => {
    const { result } = renderHook(() => useRouter())

    act(() => result.current.navigate('/library/library-movies'))
    expect(result.current.pathname).toBe('/library/library-movies')
    expect(result.current.route).toEqual({ name: 'library', libraryId: 'library-movies' })

    act(() => {
      window.history.replaceState({}, '', '/')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })
    expect(result.current.pathname).toBe('/')
    expect(result.current.route).toEqual({ name: 'home' })
  })
})
