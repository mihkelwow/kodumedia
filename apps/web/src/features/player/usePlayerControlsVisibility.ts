import { useCallback, useEffect, useRef, useState } from 'react'

interface PlayerControlsVisibilityOptions {
  playing: boolean
  locked: boolean
  hideDelayMs?: number
}

/** Keeps playback controls immediate on input and hides them only after inactivity. */
export function usePlayerControlsVisibility({
  playing,
  locked,
  hideDelayMs = 2600,
}: PlayerControlsVisibilityOptions) {
  const timerRef = useRef<number | null>(null)
  const [visible, setVisible] = useState(true)

  const clearTimer = useCallback(() => {
    if (timerRef.current === null) return
    window.clearTimeout(timerRef.current)
    timerRef.current = null
  }, [])

  const reveal = useCallback(() => {
    setVisible(true)
    clearTimer()
    if (playing && !locked) {
      timerRef.current = window.setTimeout(() => {
        timerRef.current = null
        setVisible(false)
      }, hideDelayMs)
    }
  }, [clearTimer, hideDelayMs, locked, playing])

  useEffect(() => {
    reveal()
    return clearTimer
  }, [clearTimer, reveal])

  return { visible, reveal }
}
