import type { PlaybackOptions } from '../../api/types'
import { api } from '../../api/client'

interface CachedOptions {
  expiresAt: number
  promise: Promise<PlaybackOptions>
}

const optionCache = new Map<string, CachedOptions>()
const optionsTtlMs = 60_000

export function getPlaybackOptions(mediaId: string): Promise<PlaybackOptions> {
  const now = Date.now()
  const cached = optionCache.get(mediaId)
  if (cached && cached.expiresAt > now) return cached.promise

  const promise = api.playback.options(mediaId).catch((error) => {
    const current = optionCache.get(mediaId)
    if (current?.promise === promise) optionCache.delete(mediaId)
    throw error
  })
  optionCache.set(mediaId, { expiresAt: now + optionsTtlMs, promise })
  return promise
}

export function preloadPlayback(mediaId: string): void {
  if (!mediaId) return
  // Browser module loading and playback-option discovery are safe to warm while
  // the detail view is open. Actual FFmpeg work still starts only after Play.
  void import('hls.js').catch(() => undefined)
  void getPlaybackOptions(mediaId).catch(() => undefined)
}
