import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { Sidebar } from './Sidebar'

describe('Sidebar branding', () => {
  it('uses the Kodu logo instead of the legacy text mark', () => {
    render(<Sidebar libraries={[]} pathname="/" onNavigate={vi.fn()} />)

    const logo = screen.getByRole('img', { name: 'Kodu' })
    expect(logo.getAttribute('src')).toBe('/kodu-media-logo.png')
    expect(screen.queryByText('Media')).toBeNull()
    expect(screen.queryByText('koduserver')).toBeNull()
  })
})
