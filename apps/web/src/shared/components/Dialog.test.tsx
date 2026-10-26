import { useState } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Dialog } from './Dialog'

function DialogHarness({ onClose = vi.fn() }: { onClose?: () => void }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>Open dialog</button>
      {open ? (
        <Dialog onClose={() => { setOpen(false); onClose() }} ariaLabel="Shared dialog">
          <button type="button">First action</button>
          <button type="button">Last action</button>
        </Dialog>
      ) : null}
    </>
  )
}

describe('Dialog', () => {
  it('locks scrolling, traps focus, closes on Escape and restores focus', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<DialogHarness onClose={onClose} />)

    const trigger = screen.getByRole('button', { name: 'Open dialog' })
    await user.click(trigger)
    expect(screen.getByRole('dialog', { name: 'Shared dialog' })).toBeTruthy()
    expect(document.body.style.overflow).toBe('hidden')
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole('button', { name: 'First action' })))

    await user.tab({ shift: true })
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Last action' }))

    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog')).toBeNull()
    await waitFor(() => expect(document.activeElement).toBe(trigger))
    expect(document.body.style.overflow).toBe('')
  })
})
