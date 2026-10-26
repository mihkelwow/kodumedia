import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FileBrowserDialog } from './FileBrowserDialog'

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}

describe('FileBrowserDialog', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const raw = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
      const url = new URL(raw, 'http://localhost')
      const path = url.searchParams.get('path') || ''
      if (path === '') {
        return jsonResponse({ roots: [{ name: 'C:', path: 'C:\\', directory: true }], entries: [] })
      }
      if (path === 'C:\\') {
        return jsonResponse({ currentPath: 'C:\\', parentPath: '', roots: [], entries: [{ name: 'Media', path: 'C:\\Media', directory: true }] })
      }
      if (path === 'C:\\Media') {
        return jsonResponse({ currentPath: 'C:\\Media', parentPath: 'C:\\', roots: [], entries: [{ name: 'Filmid', path: 'C:\\Media\\Filmid', directory: true }] })
      }
      throw new Error(`Unexpected path: ${path}`)
    }))
  })

  it('browses server drives and returns the selected directory', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    render(<FileBrowserDialog title="Vali meediakaust" mode="directory" onClose={vi.fn()} onSelect={onSelect} />)

    const dialog = screen.getByRole('dialog')
    expect(dialog.classList.contains('file-browser')).toBe(true)
    expect(dialog.parentElement?.parentElement).toBe(document.body)

    await user.click(await screen.findByRole('option', { name: /C:/ }))
    await user.click(await screen.findByRole('option', { name: /Media/ }))
    await user.click(screen.getByRole('button', { name: 'Vali see kaust' }))

    expect(onSelect).toHaveBeenCalledWith('C:\\Media')
  })
})
