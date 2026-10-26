import { act, fireEvent, render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AmbientBackdrop, optimizeAmbientImageUrl } from './AmbientBackdrop'

class ReadyImage {
  complete = true
  naturalWidth = 500
  decoding = 'async'
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  private value = ''

  set src(value: string) {
    this.value = value
    queueMicrotask(() => this.onload?.())
  }

  get src() {
    return this.value
  }

  decode() {
    return Promise.resolve()
  }
}

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('AmbientBackdrop', () => {
  it('uses a smaller TMDb image for heavily blurred ambient artwork only', () => {
    expect(optimizeAmbientImageUrl('https://image.tmdb.org/t/p/w1280/example.jpg'))
      .toBe('https://image.tmdb.org/t/p/w500/example.jpg')
    expect(optimizeAmbientImageUrl('https://cdn.example.com/t/p/w1280/example.jpg'))
      .toBe('https://cdn.example.com/t/p/w1280/example.jpg')
    expect(optimizeAmbientImageUrl('/local/backdrop.jpg')).toBe('/local/backdrop.jpg')
  })

  it('keeps two persistent layers and stages the decoded replacement in a hidden buffer', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('Image', ReadyImage)
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => window.setTimeout(() => callback(0), 1))
    vi.stubGlobal('cancelAnimationFrame', (handle: number) => window.clearTimeout(handle))

    const view = render(<AmbientBackdrop url="/one.jpg" />)

    await act(async () => {
      await Promise.resolve()
    })
    expect(view.container.querySelectorAll('.ambient-backdrop__layer')).toHaveLength(2)
    const initialBase = view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--base')
    expect(initialBase?.style.backgroundImage).toContain('/one.jpg')
    expect(view.container.querySelector('.ambient-backdrop__layer--incoming')).toBeNull()

    view.rerender(<AmbientBackdrop url="/two.jpg" />)
    await act(async () => {
      await Promise.resolve()
    })

    expect(view.container.querySelectorAll('.ambient-backdrop__layer')).toHaveLength(2)
    const stagedLayer = view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--incoming')
    expect(stagedLayer?.style.backgroundImage).toContain('/two.jpg')
    expect(stagedLayer?.classList.contains('ambient-backdrop__layer--revealed')).toBe(false)
    expect(view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--base')?.style.backgroundImage)
      .toContain('/one.jpg')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2)
    })

    const incomingLayer = view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--incoming')
    expect(incomingLayer?.classList.contains('ambient-backdrop__layer--revealed')).toBe(true)
    fireEvent.transitionEnd(incomingLayer!, { propertyName: 'opacity' })

    expect(view.container.querySelectorAll('.ambient-backdrop__layer')).toHaveLength(2)
    expect(view.container.querySelector('.ambient-backdrop__layer--incoming')).toBeNull()
    expect(view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--base')?.style.backgroundImage)
      .toContain('/two.jpg')
  })

  it('keeps only the latest decoded request while the current crossfade settles', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('Image', ReadyImage)
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => window.setTimeout(() => callback(0), 1))
    vi.stubGlobal('cancelAnimationFrame', (handle: number) => window.clearTimeout(handle))

    const view = render(<AmbientBackdrop url="/rapid-one.jpg" />)
    await act(async () => {
      await Promise.resolve()
    })

    view.rerender(<AmbientBackdrop url="/rapid-two.jpg" />)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2)
    })

    const firstIncoming = view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--incoming')
    expect(firstIncoming?.style.backgroundImage).toContain('/rapid-two.jpg')
    expect(firstIncoming?.classList.contains('ambient-backdrop__layer--revealed')).toBe(true)

    view.rerender(<AmbientBackdrop url="/rapid-three.jpg" />)
    await act(async () => {
      await Promise.resolve()
    })
    view.rerender(<AmbientBackdrop url="/rapid-four.jpg" />)
    await act(async () => {
      await Promise.resolve()
    })

    expect([...view.container.querySelectorAll<HTMLElement>('.ambient-backdrop__layer')].some(
      (layer) => layer.style.backgroundImage.includes('/rapid-three.jpg'),
    )).toBe(false)

    fireEvent.transitionEnd(firstIncoming!, { propertyName: 'opacity' })

    expect(view.container.querySelector('.ambient-backdrop__layer--incoming')).toBeNull()
    expect(view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--base')?.style.backgroundImage)
      .toContain('/rapid-two.jpg')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })

    const stagedLatest = view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--incoming')
    expect(stagedLatest?.style.backgroundImage).toContain('/rapid-four.jpg')
    expect(stagedLatest?.classList.contains('ambient-backdrop__layer--revealed')).toBe(false)
    expect(view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--base')?.style.backgroundImage)
      .toContain('/rapid-two.jpg')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2)
    })

    const latestIncoming = view.container.querySelector<HTMLElement>('.ambient-backdrop__layer--incoming')
    expect(latestIncoming?.style.backgroundImage).toContain('/rapid-four.jpg')
    expect(latestIncoming?.classList.contains('ambient-backdrop__layer--revealed')).toBe(true)
    expect(view.container.querySelectorAll('.ambient-backdrop__layer')).toHaveLength(2)
  })
})
