import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ProgressBar } from './ProgressBar'

describe('ProgressBar', () => {
  it('uses a transform and exposes native progress semantics', () => {
    render(<ProgressBar value={37} label="Edenemine" />)
    const root = screen.getByRole('progressbar', { name: 'Edenemine' })
    const fill = root.firstElementChild as HTMLElement
    expect(root.getAttribute('aria-valuenow')).toBe('37')
    expect(fill.style.transform).toBe('scaleX(0.37)')
    expect(fill.style.width).toBe('')
  })
})
