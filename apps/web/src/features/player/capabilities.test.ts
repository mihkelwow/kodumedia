import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  clearClientCapabilityCache,
  detectClientCapabilities,
} from './capabilities'

afterEach(() => {
  clearClientCapabilityCache()
  vi.restoreAllMocks()
})

describe('client capability cache', () => {
  it('invalidates the cached probe when the active display HDR signal changes', async () => {
    let hdrEnabled = false
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: (query: string) => ({
        matches: hdrEnabled && (
          query.includes('dynamic-range: high') ||
          query.includes('rec2020')
        ),
        media: query,
        onchange: null,
        addListener: () => undefined,
        removeListener: () => undefined,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        dispatchEvent: () => false,
      }),
    })

    const decodingInfo = vi.fn(async () => ({
      supported: true,
      smooth: true,
      powerEfficient: true,
    }))
    Object.defineProperty(navigator, 'mediaCapabilities', {
      configurable: true,
      value: { decodingInfo },
    })

    const technical = {
      videoCodec: 'hevc',
      width: 3840,
      height: 2160,
      frameRate: 24,
      bitrate: 25_000_000,
      bitDepth: 10,
      hdrFormat: 'hdr10',
    }

    const first = await detectClientCapabilities(technical)
    const firstCallCount = decodingInfo.mock.calls.length
    expect(firstCallCount).toBeGreaterThan(0)
    expect(first.hdrFormats).toEqual([])

    await detectClientCapabilities(technical)
    expect(decodingInfo).toHaveBeenCalledTimes(firstCallCount)

    hdrEnabled = true
    const second = await detectClientCapabilities(technical)
    expect(decodingInfo.mock.calls.length).toBeGreaterThan(firstCallCount)
    expect(second.hdrFormats).toContain('hdr10')
  })
})
