import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

interface DisclosureProps {
  title: ReactNode
  description?: ReactNode
  children: ReactNode
  icon?: IconName
  className?: string
  defaultOpen?: boolean
}

/** Native, keyboard-accessible disclosure for infrequent or advanced controls. */
export function Disclosure({
  title,
  description,
  children,
  icon = 'sliders',
  className = '',
  defaultOpen = false,
}: DisclosureProps) {
  return (
    <details className={`ui-disclosure ${className}`.trim()} open={defaultOpen || undefined}>
      <summary data-focusable="true">
        <span className="ui-disclosure__icon"><Icon name={icon} size={17} /></span>
        <span className="ui-disclosure__copy">
          <strong>{title}</strong>
          {description ? <small>{description}</small> : null}
        </span>
        <Icon className="ui-disclosure__chevron" name="chevronDown" size={17} />
      </summary>
      <div className="ui-disclosure__content">{children}</div>
    </details>
  )
}
