import { describe, expect, it } from 'vitest'
import type { PlaybackPlan, PlaybackSession } from '../../api/types'
import { actualPositionMs, bufferPolicyForPlan, bufferedAheadSeconds, hlsBufferSizeBytes, isBufferedAt, shouldUseNativeHls, startupBufferTargetSeconds } from './PlayerModal'

describe('player timeline math', () => {
  it('uses the original media position for a restarted HLS session', () => {
    const video = { currentTime: 12.5 } as HTMLVideoElement
    const session = {
      mediaId: 'movie',
      plan: { mode: 'transcode', reason: '', container: 'fmp4-hls', startPositionMs: 3_600_000 },
    } satisfies PlaybackSession
    expect(actualPositionMs(video, session)).toBe(3_612_500)
  })

  it('measures only the buffered range ahead of the current frame', () => {
    const video = {
      currentTime: 5,
      buffered: {
        length: 2,
        start: (index: number) => index === 0 ? 0 : 20,
        end: (index: number) => index === 0 ? 11 : 30,
      },
    } as unknown as HTMLVideoElement
    expect(bufferedAheadSeconds(video)).toBe(6)
    expect(isBufferedAt(video, 8)).toBe(true)
    expect(isBufferedAt(video, 15)).toBe(false)
  })

  it('buffers feature films more than short clips before autoplay', () => {
    expect(startupBufferTargetSeconds(6_000)).toBe(8)
    expect(startupBufferTargetSeconds(45)).toBe(3)
  })

  it('keeps Safari on its native HLS path so AirPlay receives a network URL', () => {
    expect(shouldUseNativeHls(true, true)).toBe(true)
    expect(shouldUseNativeHls(true, false)).toBe(false)
    expect(shouldUseNativeHls(false, true)).toBe(false)
  })

  it('uses server-defined startup and rebuffer targets for high-quality transcodes', () => {
    const plan = {
      mode: 'transcode',
      reason: 'quality engine',
      container: 'fmp4-hls',
      startupBufferSeconds: 20,
      rebufferBufferSeconds: 30,
      desiredBufferSeconds: 55,
      lowBufferSeconds: 18,
    } satisfies PlaybackPlan

    expect(bufferPolicyForPlan(plan, 7_200)).toEqual({
      startupSeconds: 20,
      rebufferSeconds: 30,
      desiredSeconds: 55,
      lowSeconds: 18,
    })
  })

  it('keeps direct playback buffer targets lightweight', () => {
    const plan = {
      mode: 'direct',
      reason: 'direct play',
      container: 'mp4',
      startupBufferSeconds: 4,
      rebufferBufferSeconds: 8,
      desiredBufferSeconds: 20,
      lowBufferSeconds: 4,
    } satisfies PlaybackPlan

    expect(bufferPolicyForPlan(plan, 7_200)).toEqual({
      startupSeconds: 4,
      rebufferSeconds: 8,
      desiredSeconds: 20,
      lowSeconds: 4,
    })
  })


  it('allows a high-bitrate 4K stream to reach the rebuffer target without an early byte cap', () => {
    const bytes = hlsBufferSizeBytes(90_000_000, 55)
    expect(bytes).toBeGreaterThan(90_000_000 * 30 / 8)
    expect(bytes).toBeLessThanOrEqual(512 * 1024 * 1024)
  })

})
