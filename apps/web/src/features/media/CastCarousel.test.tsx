import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { PersonCredit } from '../../api/types'
import { CastCarousel } from './CastCarousel'

const people: PersonCredit[] = [
  { name: 'First Person', character: 'Lead' },
  { name: 'Second Person', character: 'Support' },
]

describe('CastCarousel pointer input', () => {
  it('leaves touch scrolling native and only captures mouse dragging', () => {
    render(<CastCarousel people={people} />)
    const scroller = screen.getByLabelText('Näitlejad ja tegijad')
    Object.defineProperty(scroller, 'clientWidth', { configurable: true, value: 300 })
    Object.defineProperty(scroller, 'scrollWidth', { configurable: true, value: 700 })
    const setPointerCapture = vi.fn()
    Object.defineProperty(scroller, 'setPointerCapture', { configurable: true, value: setPointerCapture })

    fireEvent.pointerDown(scroller, { pointerId: 1, pointerType: 'touch', button: 0, clientX: 180 })
    expect(setPointerCapture).not.toHaveBeenCalled()
    expect(scroller.classList.contains('cast-list--dragging')).toBe(false)

    fireEvent.pointerDown(scroller, { pointerId: 2, pointerType: 'mouse', button: 0, clientX: 180 })
    expect(setPointerCapture).toHaveBeenCalledWith(2)
    expect(scroller.classList.contains('cast-list--dragging')).toBe(true)
  })
})
