import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach } from 'vitest'
import { resetLocaleForTests } from '../i18n/i18n'

beforeEach(() => resetLocaleForTests('et-EE'))

afterEach(() => cleanup())

Object.defineProperty(window, 'matchMedia', {
  configurable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }),
})

class TestIntersectionObserver {
  readonly root = null
  readonly rootMargin = '0px'
  readonly thresholds = [0]

  constructor(private readonly callback: IntersectionObserverCallback) {}

  observe(target: Element) {
    this.callback([{
      boundingClientRect: target.getBoundingClientRect(),
      intersectionRatio: 1,
      intersectionRect: target.getBoundingClientRect(),
      isIntersecting: true,
      rootBounds: null,
      target,
      time: performance.now(),
    }], this as unknown as IntersectionObserver)
  }

  disconnect() {}
  takeRecords() { return [] }
  unobserve() {}
}

class TestResizeObserver {
  constructor(_callback: ResizeObserverCallback) {}

  observe(_target: Element) {}

  disconnect() {}
  unobserve() {}
}

Object.defineProperty(globalThis, 'IntersectionObserver', {
  configurable: true,
  value: TestIntersectionObserver,
})

Object.defineProperty(globalThis, 'ResizeObserver', {
  configurable: true,
  value: TestResizeObserver,
})
