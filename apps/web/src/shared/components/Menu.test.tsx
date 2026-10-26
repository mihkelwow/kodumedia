import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { Menu, MenuItem } from './Menu'

describe('Menu', () => {
  it('moves focus through items with arrow, Home and End keys', async () => {
    const user = userEvent.setup()
    render(
      <Menu aria-label="Collections">
        <MenuItem>Movies</MenuItem>
        <MenuItem>Series</MenuItem>
        <MenuItem>Music</MenuItem>
      </Menu>,
    )

    const movies = screen.getByRole('menuitem', { name: 'Movies' })
    const series = screen.getByRole('menuitem', { name: 'Series' })
    const music = screen.getByRole('menuitem', { name: 'Music' })
    movies.focus()

    await user.keyboard('{ArrowDown}')
    expect(document.activeElement).toBe(series)
    await user.keyboard('{End}')
    expect(document.activeElement).toBe(music)
    await user.keyboard('{ArrowDown}')
    expect(document.activeElement).toBe(movies)
    await user.keyboard('{ArrowUp}')
    expect(document.activeElement).toBe(music)
    await user.keyboard('{Home}')
    expect(document.activeElement).toBe(movies)
  })
})
