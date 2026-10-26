import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { PlayerTrackMenu } from './PlayerTrackMenu'

function Harness({ onSelect = () => undefined }: { onSelect?: (id: string) => void }) {
  const [open, setOpen] = useState(false)
  return (
    <PlayerTrackMenu
      icon="captions"
      ariaLabel="Subtiitrid"
      title="Subtiitrid"
      open={open}
      onOpenChange={setOpen}
      options={[
        { id: '', title: 'Väljas', selected: false },
        { id: 'et', title: 'Eesti', selected: true },
        { id: 'en', title: 'English', selected: false },
      ]}
      onSelect={onSelect}
    />
  )
}

describe('PlayerTrackMenu', () => {
  it('uses the shared menu keyboard model and restores focus after Escape', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    const trigger = screen.getByRole('button', { name: 'Subtiitrid' })
    await user.click(trigger)

    const first = screen.getByRole('menuitemradio', { name: 'Väljas' })
    const second = screen.getByRole('menuitemradio', { name: 'Eesti' })
    await waitFor(() => expect(document.activeElement).toBe(first))

    await user.keyboard('{ArrowDown}')
    expect(document.activeElement).toBe(second)

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('menu', { name: 'Subtiitrid' })).toBeNull()
    await waitFor(() => expect(document.activeElement).toBe(trigger))
  })

  it('selects a track without introducing a parallel native select', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    render(<Harness onSelect={onSelect} />)

    await user.click(screen.getByRole('button', { name: 'Subtiitrid' }))
    await user.click(screen.getByRole('menuitemradio', { name: 'English' }))

    expect(onSelect).toHaveBeenCalledWith('en')
    expect(document.querySelector('.player-menu select')).toBeNull()
  })
})
