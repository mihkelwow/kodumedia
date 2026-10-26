import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { resetLocaleForTests } from '../../i18n/i18n'
import { NotificationCenter } from './NotificationCenter'

function response(body: unknown, status = 200): Response {
  return new Response(status === 204 ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('NotificationCenter', () => {
  beforeEach(() => {
    resetLocaleForTests('et-EE')
  })

  it('shows an unread badge and marks notifications read when the menu opens', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input)
      if (path.startsWith('/api/v1/notifications?')) {
        return response({
          items: [{
            id: 'series:one:s01e02',
            kind: 'series_episode_released',
            seriesId: 'series-one',
            seriesTitle: 'Näidissari',
            season: 1,
            episode: 2,
            episodeTitle: 'Uus algus',
            releaseDate: '2026-07-19T00:00:00Z',
            createdAt: '2026-07-20T10:00:00Z',
          }],
          total: 1,
          unread: 1,
          limit: 5,
          offset: 0,
        })
      }
      if (path === '/api/v1/notifications/read' && init?.method === 'POST') return response(undefined, 204)
      throw new Error(`Unexpected request: ${path}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()

    render(<NotificationCenter pathname="/" onNavigate={vi.fn()} />)

    const button = await screen.findByRole('button', { name: 'Teavitused' })
    await vi.waitFor(() => expect(button.className).toContain('header-notifications__button--unread'))
    await user.click(button)

    expect(await screen.findByRole('menu', { name: 'Teavitused' })).toBeTruthy()
    expect(screen.getByText('Näidissari')).toBeTruthy()
    expect(button.className).not.toContain('header-notifications__button--unread')
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/notifications/read', expect.objectContaining({ method: 'POST' }))
  })
})
