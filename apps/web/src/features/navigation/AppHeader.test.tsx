import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { Library } from '../../api/types'
import { ThemeProvider } from '../theme/ThemeProvider'
import { AppHeader } from './AppHeader'

const libraries: Library[] = [
  { id: 'library-movies', name: 'Filmid', kind: 'movies', system: true, sources: [], createdAt: '', updatedAt: '' },
  { id: 'library-series', name: 'Seriaalid', kind: 'series', system: true, sources: [], createdAt: '', updatedAt: '' },
  { id: 'library-music', name: 'Muusika', kind: 'music', system: true, sources: [], createdAt: '', updatedAt: '' },
]

describe('AppHeader', () => {
  it('renders the cinematic navigation and opens global search', async () => {
    const user = userEvent.setup()
    const onOpenSearch = vi.fn()
    render(
      <ThemeProvider>
        <AppHeader libraries={libraries} pathname="/" onNavigate={vi.fn()} onOpenSearch={onOpenSearch} />
      </ThemeProvider>,
    )

    expect(screen.getByRole('link', { name: 'Kodu avaleht' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Avaleht' }).getAttribute('aria-current')).toBe('page')
    expect(screen.getByRole('link', { name: 'Filmid' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Seriaalid' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Muusika' })).toBeTruthy()
    const navigationLabels = within(screen.getByRole('navigation', { name: 'Põhinavigatsioon' }))
      .getAllByRole('link')
      .map((link) => link.textContent)
    expect(navigationLabels).toEqual(['Avaleht', 'Filmid', 'Seriaalid', 'Muusika'])

    await user.click(screen.getByRole('button', { name: 'Otsi meediast' }))
    expect(onOpenSearch).toHaveBeenCalledTimes(1)
  })
})
