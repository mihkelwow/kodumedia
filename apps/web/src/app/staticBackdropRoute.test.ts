import { describe, expect, it } from 'vitest'
import { parsePath } from './router'
import { usesStaticAppBackdrop } from './staticBackdropRoute'

describe('static application backdrop routes', () => {
  it.each([
    '/',
    '/library/library-movies',
    '/library/library-series',
    '/music',
    '/settings',
  ])('uses the shared static backdrop on %s', (pathname) => {
    expect(usesStaticAppBackdrop(parsePath(pathname))).toBe(true)
  })

  it.each([
    '/library/custom-library',
    '/music/tracks',
    '/music/artist/Miles%20Davis',
    '/music/album/album-1',
    '/notifications',
  ])('does not extend the collection backdrop to %s', (pathname) => {
    expect(usesStaticAppBackdrop(parsePath(pathname))).toBe(false)
  })
})
