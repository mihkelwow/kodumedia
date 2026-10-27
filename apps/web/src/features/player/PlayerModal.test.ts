import { describe, expect, it } from 'vitest'
import type { PlaybackPlan, PlaybackSession } from '../../api/types'
import { actualPositionMs, bufferPolicyForPlan, bufferedAheadSeconds, hlsBufferSizeBytes, isBufferedAt, shouldStartBufferedPlayback, shouldUseEncoderFallback, shouldUseNativeHls, startupBufferTargetSeconds } from './PlayerModal'

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
    expect(shouldUseNativeHls(true, true, true)).toBe(true)
    expect(shouldUseNativeHls(true, false, true)).toBe(false)
    expect(shouldUseNativeHls(false, true, true)).toBe(false)
  })

  it('keeps Chromium on hls.js even when it advertises partial native HLS support', () => {
    expect(shouldUseNativeHls(true, true, false)).toBe(false)
  })

  it('keeps the complete server-defined reserve for startup and rebuffering', () => {
    const plan = {
      mode: 'transcode',
      reason: 'quality engine',
      container: 'fmp4-hls',
      startupBufferSeconds: 24,
      rebufferBufferSeconds: 36,
      desiredBufferSeconds: 90,
      lowBufferSeconds: 24,
    } satisfies PlaybackPlan

    expect(bufferPolicyForPlan(plan, 7_200)).toEqual({
      startupSeconds: 24,
      rebufferSeconds: 36,
      desiredSeconds: 90,
      lowSeconds: 24,
    })
  })

  it('never starts an HLS stream with the old two-second fail-open buffer', () => {
    expect(shouldStartBufferedPlayback(23.6, 24, 0, 2)).toBe(true)
    expect(shouldStartBufferedPlayback(18, 24, 14_999, 3)).toBe(false)
    expect(shouldStartBufferedPlayback(18, 24, 15_000, 3)).toBe(true)
    expect(shouldStartBufferedPlayback(8, 24, 60_000, 4)).toBe(false)
    expect(shouldStartBufferedPlayback(0, 24, 60_000, 1)).toBe(false)
  })

  it('requires sustained low buffer and server encoder evidence before lowering quality', () => {
    const transcode = {
      mode: 'transcode',
      reason: 'quality engine',
      container: 'fmp4-hls',
      recoveryLevel: 0,
    } satisfies PlaybackPlan
    const remux = { ...transcode, mode: 'remux' as const }

    expect(shouldUseEncoderFallback(transcode, 4, 6, 12_000, true)).toBe(true)
    expect(shouldUseEncoderFallback(transcode, 4, 6, 11_999, true)).toBe(false)
    expect(shouldUseEncoderFallback(transcode, 4, 6, 60_000, false)).toBe(false)
    expect(shouldUseEncoderFallback(transcode, 8, 6, 60_000, true)).toBe(false)
    expect(shouldUseEncoderFallback(remux, 0, 6, 60_000, true)).toBe(false)
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
    const bytes = hlsBufferSizeBytes(90_000_000, 120)
    expect(bytes).toBeGreaterThan(90_000_000 * 45 / 8)
    expect(bytes).toBeLessThanOrEqual(1024 * 1024 * 1024)
  })

})
