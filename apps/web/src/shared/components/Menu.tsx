import type { ButtonHTMLAttributes, HTMLAttributes, KeyboardEvent, ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

function enabledItems(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>('[role="menuitem"], [role="menuitemradio"], [role="menuitemcheckbox"]'))
    .filter((item) => !item.hasAttribute('disabled') && item.getAttribute('aria-disabled') !== 'true')
}

/** Vertical menu with shared arrow-key navigation. */
export function Menu({ children, className = '', onKeyDown, ...props }: HTMLAttributes<HTMLDivElement> & { children: ReactNode }) {
  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    onKeyDown?.(event)
    if (event.defaultPrevented || !['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return

    const items = enabledItems(event.currentTarget)
    if (items.length === 0) return
    const activeIndex = items.findIndex((item) => item === document.activeElement)
    let nextIndex = activeIndex
    if (event.key === 'Home') nextIndex = 0
    else if (event.key === 'End') nextIndex = items.length - 1
    else if (event.key === 'ArrowDown') nextIndex = activeIndex < 0 ? 0 : (activeIndex + 1) % items.length
    else nextIndex = activeIndex < 0 ? items.length - 1 : (activeIndex - 1 + items.length) % items.length

    event.preventDefault()
    items[nextIndex]?.focus({ preventScroll: true })
  }

  return (
    <div
      className={`ui-menu ${className}`.trim()}
      role="menu"
      aria-orientation="vertical"
      onKeyDown={handleKeyDown}
      {...props}
    >
      {children}
    </div>
  )
}

interface MenuItemProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon?: IconName
  description?: ReactNode
  selected?: boolean
}

export function MenuItem({ icon, description, selected = false, children, className = '', ...props }: MenuItemProps) {
  return (
    <button
      type="button"
      className={`ui-menu-item ${selected ? 'ui-menu-item--selected' : ''} ${className}`.trim()}
      role="menuitem"
      data-focusable="true"
      {...props}
    >
      {icon ? <span className="ui-menu-item__icon"><Icon name={icon} size={17} /></span> : null}
      <span className="ui-menu-item__copy">
        <strong>{children}</strong>
        {description ? <small>{description}</small> : null}
      </span>
      {selected ? <Icon name="check" size={16} /> : null}
    </button>
  )
}
