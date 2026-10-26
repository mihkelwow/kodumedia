import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useDirectionalNavigation } from './useDirectionalNavigation'

function Harness() {
  useDirectionalNavigation()
  return (
    <div>
      <button data-focusable="true">First</button>
      <button data-focusable="true">Second</button>
    </div>
  )
}

function expose(element: HTMLElement, left: number) {
  Object.defineProperty(element, 'offsetParent', { configurable: true, value: document.body })
  vi.spyOn(element, 'getBoundingClientRect').mockReturnValue({
    x: left,
    y: 0,
    top: 0,
    right: left + 80,
    bottom: 40,
    left,
    width: 80,
    height: 40,
    toJSON: () => ({}),
  })
  element.scrollIntoView = vi.fn()
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useDirectionalNavigation', () => {
  it('establishes the first focus before moving spatially and keeps it visible', () => {
    render(<Harness />)
    const first = screen.getByRole('button', { name: 'First' })
    const second = screen.getByRole('button', { name: 'Second' })
    expose(first, 0)
    expose(second, 120)

    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(first)
    expect(first.scrollIntoView).toHaveBeenCalledWith({ behavior: 'auto', block: 'nearest', inline: 'nearest' })

    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(second)
    expect(second.scrollIntoView).toHaveBeenCalledWith({ behavior: 'auto', block: 'nearest', inline: 'nearest' })
  })

  it('does not run after a component has handled an arrow key', () => {
    render(<Harness />)
    const first = screen.getByRole('button', { name: 'First' })
    const second = screen.getByRole('button', { name: 'Second' })
    expose(first, 0)
    expose(second, 120)
    first.focus()

    const event = new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true, cancelable: true })
    event.preventDefault()
    window.dispatchEvent(event)

    expect(document.activeElement).toBe(first)
  })
})
