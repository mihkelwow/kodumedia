import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  icon?: IconName
  children?: ReactNode
  compact?: boolean
}

export function Button({ variant = 'secondary', icon, compact = false, children, className = '', type = 'button', ...props }: ButtonProps) {
  return (
    <button
      type={type}
      className={`button button--${variant} ${compact ? 'button--compact' : ''} ${className}`.trim()}
      data-focusable="true"
      data-pressable="true"
      {...props}
    >
      {icon ? <Icon name={icon} size={compact ? 17 : 19} /> : null}
      {children ? <span>{children}</span> : null}
    </button>
  )
}
