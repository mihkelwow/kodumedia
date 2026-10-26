import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SearchOverlay } from './SearchOverlay'

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('SearchOverlay', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('searches globally and opens a result', async () => {
    const user = userEvent.setup()
    const onOpen = vi.fn()
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
      expect(url).toContain('/api/v1/media?')
      expect(url).toContain('search=Ballerina')
      return jsonResponse({
        items: [{
          id: 'movie-1', libraryId: 'library-movies', type: 'movie', title: 'Ballerina', sortTitle: 'Ballerina', year: 2025,
          file: { relativePath: 'Ballerina.mkv', name: 'Ballerina.mkv', extension: '.mkv', sizeBytes: 1, modifiedAt: '2026-01-01T00:00:00Z' },
          technical: {}, externalIds: {}, metadata: { imdbRating: 7.2 }, addedAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z',
        }],
        total: 1, limit: 80, offset: 0,
      })
    }))

    render(<SearchOverlay onClose={vi.fn()} onOpen={onOpen} />)
    await user.type(screen.getByRole('textbox', { name: 'Otsing' }), 'Ballerina')

    const result = await screen.findByRole('button', { name: 'Ava Ballerina' }, { timeout: 3000 })
    expect(screen.getByRole('heading', { name: 'Filmid' })).toBeTruthy()
    await user.click(result)
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ id: 'movie-1' }))
    await waitFor(() => expect(document.body.style.overflow).toBe('hidden'))
  })

  it('closes when the user clicks the background outside the search content', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<SearchOverlay onClose={onClose} onOpen={vi.fn()} />)

    const overlay = document.querySelector('.search-overlay')
    expect(overlay).toBeTruthy()
    await user.click(overlay as Element)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

})
