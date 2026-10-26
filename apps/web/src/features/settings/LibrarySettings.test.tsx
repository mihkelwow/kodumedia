import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { Library } from '../../api/types'
import { LibrarySettings } from './LibrarySettings'

const libraries: Library[] = [
  {
    id: 'series',
    name: 'Seriaalid',
    kind: 'series',
    system: true,
    sources: [{ id: 'source-1', path: 'D:\\Media\\Series', createdAt: '2026-07-19T00:00:00Z' }],
    createdAt: '2026-07-19T00:00:00Z',
    updatedAt: '2026-07-19T00:00:00Z',
  },
]

describe('LibrarySettings', () => {
  it('uses the shared CVI select and a flat library composition', () => {
    const { container } = render(<LibrarySettings libraries={libraries} onChanged={vi.fn(async () => undefined)} />)

    expect(screen.getByRole('heading', { name: 'Meediakogud ja kaustad' })).toBeTruthy()
    expect(screen.getByText('D:\\Media\\Series')).toBeTruthy()

    const kindSelect = screen.getByRole('combobox', { name: 'Sisutüüp' })
    expect(kindSelect.closest('.select-control')).toBeTruthy()
    expect(kindSelect.closest('.create-library--standalone')).toBeTruthy()

    const section = container.querySelector('#settings-libraries')
    expect(section?.classList.contains('settings-panel--flat')).toBe(true)

    const libraryItem = screen.getByRole('heading', { name: 'Seriaalid' }).closest('.library-settings-item')
    expect(libraryItem?.querySelector('.library-source-group')).toBeTruthy()
    expect(libraryItem?.querySelector('.library-source-row__copy')).toBeTruthy()

    expect(container.querySelector('.library-settings-card')).toBeNull()
    expect(container.querySelector('.library-source-section')).toBeNull()
    expect(container.querySelector('.source-picker-action')).toBeNull()
    expect(libraryItem?.querySelector('.library-settings-item')).toBeNull()
  })
})
