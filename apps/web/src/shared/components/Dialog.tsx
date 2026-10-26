import { useEffect, useId, useRef, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'

let bodyScrollLocks = 0
let previousBodyOverflow = ''

function lockBodyScroll() {
  if (bodyScrollLocks === 0) {
    previousBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  }
  bodyScrollLocks += 1
}

function unlockBodyScroll() {
  bodyScrollLocks = Math.max(0, bodyScrollLocks - 1)
  if (bodyScrollLocks === 0) document.body.style.overflow = previousBodyOverflow
}

function focusableElements(root: HTMLElement): HTMLElement[] {
  const selector = [
    'a[href]',
    'button:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
    '[contenteditable="true"]',
  ].join(',')
  return Array.from(root.querySelectorAll<HTMLElement>(selector))
    .filter((element) => !element.hidden && element.getAttribute('aria-hidden') !== 'true')
}

interface DialogProps {
  children: ReactNode
  onClose: () => void
  ariaLabel?: string
  ariaLabelledBy?: string
  className?: string
  layerClassName?: string
  closeOnBackdrop?: boolean
  initialFocusRef?: RefObject<HTMLElement | null>
  motion?: 'standard' | 'instant'
}

/**
 * Shared modal primitive. It owns Escape, focus trapping, focus restoration and
 * the body scroll lock so feature components cannot drift into subtly different
 * behaviour.
 */
export function Dialog({
  children,
  onClose,
  ariaLabel,
  ariaLabelledBy,
  className = '',
  layerClassName = '',
  closeOnBackdrop = true,
  initialFocusRef,
  motion = 'standard',
}: DialogProps) {
  const generatedId = useId()
  const surfaceRef = useRef<HTMLDivElement>(null)
  const previousFocusRef = useRef<HTMLElement | null>(null)
  const onCloseRef = useRef(onClose)

  useEffect(() => {
    onCloseRef.current = onClose
  }, [onClose])

  useEffect(() => {
    previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    lockBodyScroll()

    const surface = surfaceRef.current
    const focusTarget = initialFocusRef?.current ?? (surface ? focusableElements(surface)[0] : null) ?? surface
    window.requestAnimationFrame(() => focusTarget?.focus({ preventScroll: true }))

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onCloseRef.current()
        return
      }
      if (event.key !== 'Tab' || !surfaceRef.current) return

      const focusable = focusableElements(surfaceRef.current)
      if (focusable.length === 0) {
        event.preventDefault()
        surfaceRef.current.focus({ preventScroll: true })
        return
      }

      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (!first || !last) return
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }

    window.addEventListener('keydown', handleKeyDown, true)
    return () => {
      window.removeEventListener('keydown', handleKeyDown, true)
      unlockBodyScroll()
      const previous = previousFocusRef.current
      if (previous?.isConnected) window.requestAnimationFrame(() => previous.focus({ preventScroll: true }))
    }
  }, [initialFocusRef])

  const dialog = (
    <div
      className={`ui-dialog-layer ${layerClassName}`.trim()}
      data-motion={motion}
      role="presentation"
      onPointerDown={(event) => {
        if (closeOnBackdrop && event.target === event.currentTarget) onCloseRef.current()
      }}
    >
      <div
        ref={surfaceRef}
        id={generatedId}
        className={`ui-dialog-surface ${className}`.trim()}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        tabIndex={-1}
      >
        {children}
      </div>
    </div>
  )

  return createPortal(dialog, document.body)
}
