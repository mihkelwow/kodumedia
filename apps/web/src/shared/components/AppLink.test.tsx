import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AppLink } from './AppLink'

describe('AppLink', () => {
  it('uses client-side navigation for an ordinary left click', () => {
    const navigate = vi.fn()
    render(<AppLink href="/settings" onNavigate={navigate}>Seaded</AppLink>)

    fireEvent.click(screen.getByRole('link', { name: 'Seaded' }))
    expect(navigate).toHaveBeenCalledWith('/settings')
  })

  it('does not intercept a link intended for a new tab', () => {
    const navigate = vi.fn()
    render(<AppLink href="/settings" target="_blank" onNavigate={navigate}>Seaded</AppLink>)

    fireEvent.click(screen.getByRole('link', { name: 'Seaded' }))
    expect(navigate).not.toHaveBeenCalled()
  })
})
