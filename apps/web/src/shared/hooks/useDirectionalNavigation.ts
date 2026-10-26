import { useEffect } from 'react'

function isEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  return ['INPUT', 'TEXTAREA', 'SELECT', 'VIDEO', 'AUDIO'].includes(target.tagName) || target.isContentEditable
}

export function useDirectionalNavigation(enabled = true) {
  useEffect(() => {
    if (!enabled) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented || !['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(event.key) || isEditable(event.target)) return
      const focusables = Array.from(document.querySelectorAll<HTMLElement>('[data-focusable="true"]'))
        .filter((element) => !element.hasAttribute('disabled') && element.offsetParent !== null)
      if (focusables.length === 0) return

      const active = document.activeElement instanceof HTMLElement && focusables.includes(document.activeElement)
        ? document.activeElement
        : null
      if (!active) {
        const first = focusables[0]
        if (!first) return
        event.preventDefault()
        first.focus({ preventScroll: true })
        first.scrollIntoView({ behavior: 'auto', block: 'nearest', inline: 'nearest' })
        return
      }
      const origin = active.getBoundingClientRect()
      const originX = origin.left + origin.width / 2
      const originY = origin.top + origin.height / 2

      const candidates = focusables
        .filter((element) => element !== active)
        .map((element) => {
          const rect = element.getBoundingClientRect()
          const x = rect.left + rect.width / 2
          const y = rect.top + rect.height / 2
          const dx = x - originX
          const dy = y - originY
          const valid =
            (event.key === 'ArrowRight' && dx > 4) ||
            (event.key === 'ArrowLeft' && dx < -4) ||
            (event.key === 'ArrowDown' && dy > 4) ||
            (event.key === 'ArrowUp' && dy < -4)
          if (!valid) return null
          const primary = event.key === 'ArrowLeft' || event.key === 'ArrowRight' ? Math.abs(dx) : Math.abs(dy)
          const secondary = event.key === 'ArrowLeft' || event.key === 'ArrowRight' ? Math.abs(dy) : Math.abs(dx)
          return { element, score: primary + secondary * 2.4 }
        })
        .filter((candidate): candidate is { element: HTMLElement; score: number } => candidate !== null)
        .sort((a, b) => a.score - b.score)

      const next = candidates[0]?.element
      if (next) {
        event.preventDefault()
        next.focus({ preventScroll: true })
        next.scrollIntoView({ behavior: 'auto', block: 'nearest', inline: 'nearest' })
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [enabled])
}
