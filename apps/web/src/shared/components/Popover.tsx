import {
  cloneElement,
  useEffect,
  useId,
  useRef,
  type AriaRole,
  type MouseEvent as ReactMouseEvent,
  type ReactElement,
  type ReactNode,
} from 'react'

type Placement = 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end'

interface PopoverProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  trigger: ReactElement<any>
  children: ReactNode
  placement?: Placement
  className?: string
  rootClassName?: string
  ariaLabel?: string
  autoFocus?: boolean
  role?: AriaRole
}

function firstFocusable(root: HTMLElement): HTMLElement | null {
  return root.querySelector<HTMLElement>(
    'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )
}

/** Anchored, origin-aware popover with shared dismissal and focus restoration. */
export function Popover({
  open,
  onOpenChange,
  trigger,
  children,
  placement = 'bottom-start',
  className = '',
  rootClassName = '',
  ariaLabel,
  autoFocus = false,
  role = 'dialog',
}: PopoverProps) {
  const rootRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLElement | null>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const onOpenChangeRef = useRef(onOpenChange)
  const id = useId()

  useEffect(() => {
    onOpenChangeRef.current = onOpenChange
  }, [onOpenChange])

  useEffect(() => {
    if (!open) return
    const close = (restoreFocus: boolean) => {
      onOpenChangeRef.current(false)
      if (restoreFocus) window.requestAnimationFrame(() => triggerRef.current?.focus({ preventScroll: true }))
    }
    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) close(false)
    }
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.preventDefault()
      close(true)
    }
    document.addEventListener('pointerdown', closeOnOutsidePointer, true)
    window.addEventListener('keydown', closeOnEscape, true)
    if (autoFocus) {
      window.requestAnimationFrame(() => {
        const panel = panelRef.current
        if (panel) firstFocusable(panel)?.focus({ preventScroll: true })
      })
    }
    return () => {
      document.removeEventListener('pointerdown', closeOnOutsidePointer, true)
      window.removeEventListener('keydown', closeOnEscape, true)
    }
  }, [autoFocus, open])

  const originalClick = trigger.props.onClick as ((event: ReactMouseEvent<HTMLElement>) => void) | undefined
  const triggerElement = cloneElement(trigger, {
    onClick: (event: ReactMouseEvent<HTMLElement>) => {
      triggerRef.current = event.currentTarget
      originalClick?.(event)
      if (!event.defaultPrevented) onOpenChangeRef.current(!open)
    },
    'aria-expanded': open,
    'aria-controls': open ? id : undefined,
    'aria-haspopup': trigger.props['aria-haspopup'] ?? role,
  })

  return (
    <div ref={rootRef} className={`ui-popover-root ${rootClassName}`.trim()}>
      {triggerElement}
      {open ? (
        <div
          ref={panelRef}
          id={id}
          className={`ui-popover ui-popover--${placement} ${className}`.trim()}
          role={role}
          aria-label={ariaLabel}
          data-placement={placement}
        >
          {children}
        </div>
      ) : null}
    </div>
  )
}
