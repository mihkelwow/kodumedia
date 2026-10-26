import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { StorageReport } from '../../api/types'
import { resetLocaleForTests } from '../../i18n/i18n'
import { StorageSettings } from './StorageSettings'

const report: StorageReport = {
  dataDirectory: 'C:\\KoduMedia\\data',
  dataSizeBytes: 3 * 1024 * 1024 * 1024,
  diskFreeBytes: 80 * 1024 * 1024 * 1024,
  diskTotalBytes: 500 * 1024 * 1024 * 1024,
  requiredReserveBytes: 50 * 1024 * 1024 * 1024,
  generatedAt: '2026-07-25T20:00:00Z',
  categories: [
    {
      id: 'optimized-media',
      sizeBytes: 2 * 1024 * 1024 * 1024,
      limitBytes: 50 * 1024 * 1024 * 1024,
      reclaimableBytes: 2 * 1024 * 1024 * 1024,
      clearable: true,
    },
    { id: 'updates', sizeBytes: 20 * 1024 * 1024, retentionCount: 2 },
  ],
}

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('StorageSettings', () => {
  beforeEach(() => {
    resetLocaleForTests('et-EE')
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.pathname : input.url
      if (url === '/api/v1/system/storage' && (!init?.method || init.method === 'GET')) {
        return jsonResponse(report)
      }
      if (url === '/api/v1/system/storage' && init?.method === 'POST') {
        return jsonResponse({
          freedBytes: 2 * 1024 * 1024 * 1024,
          report: {
            ...report,
            dataSizeBytes: 1024 * 1024 * 1024,
            categories: report.categories.map((category) => category.id === 'optimized-media' ? { ...category, sizeBytes: 0, reclaimableBytes: 0 } : category),
          },
        })
      }
      throw new Error(`Unexpected request: ${url} ${init?.method || 'GET'}`)
    }))
  })

  it('shows category limits and clears only through the protected localhost action', async () => {
    const user = userEvent.setup()
    render(<StorageSettings />)

    expect(await screen.findByText('Optimeeritud videod')).toBeTruthy()
    expect(screen.getByText('Säilitatakse kaks eelmist versiooni.')).toBeTruthy()
    expect(screen.getByText('50 GB')).toBeTruthy()

    await user.click(screen.getByRole('button', { name: 'Tühjenda optimeeritud vahemälu' }))
    await screen.findByText('Kettahooldus vabastas 2.0 GB.')

    expect(confirm).toHaveBeenCalledTimes(1)
    const post = vi.mocked(fetch).mock.calls.find(([, init]) => init?.method === 'POST')
    expect(post).toBeTruthy()
    expect(post?.[1]?.headers).toEqual(expect.objectContaining({ 'X-KoduMedia-Storage': 'manage' }))
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ action: 'clear-optimized' })
    await waitFor(() => expect(screen.getByText('0 B')).toBeTruthy())
  })
})
