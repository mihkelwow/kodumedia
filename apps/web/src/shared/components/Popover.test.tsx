import { useState } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { Menu, MenuItem } from './Menu'
import { Popover } from './Popover'

function PopoverHarness() {
  const [open, setOpen] = useState(false)
  return (
    <div>
      <Popover
        open={open}
        onOpenChange={setOpen}
        placement="bottom-end"
        ariaLabel="Actions"
        role="menu"
        autoFocus
        trigger={<button type="button">Open actions</button>}
      >
        <Menu role="presentation">
          <MenuItem>First item</MenuItem>
          <MenuItem>Second item</MenuItem>
        </Menu>
      </Popover>
      <button type="button">Outside</button>
    </div>
  )
}

describe('Popover', () => {
  it('opens from its trigger, focuses content and restores focus on Escape', async () => {
    const user = userEvent.setup()
    render(<PopoverHarness />)

    const trigger = screen.getByRole('button', { name: 'Open actions' })
    await user.click(trigger)
    const menu = screen.getByRole('menu', { name: 'Actions' })
    expect(menu.getAttribute('data-placement')).toBe('bottom-end')
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole('menuitem', { name: 'First item' })))

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('menu', { name: 'Actions' })).toBeNull()
    await waitFor(() => expect(document.activeElement).toBe(trigger))
  })

  it('dismisses when the user points outside', async () => {
    const user = userEvent.setup()
    render(<PopoverHarness />)
    await user.click(screen.getByRole('button', { name: 'Open actions' }))
    expect(screen.getByRole('menu', { name: 'Actions' })).toBeTruthy()
    await user.click(screen.getByRole('button', { name: 'Outside' }))
    expect(screen.queryByRole('menu', { name: 'Actions' })).toBeNull()
  })
})
