import type { ButtonHTMLAttributes } from 'react'
import { Icon, type IconName } from './Icon'

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  icon: IconName
  label: string
  size?: 'small' | 'medium'
  variant?: 'ghost' | 'surface' | 'primary'
  iconSize?: number
}

export function IconButton({ icon, label, size = 'medium', variant = 'ghost', iconSize, className = '', type = 'button', ...props }: IconButtonProps) {
  return (
    <button
      type={type}
      className={`icon-button icon-button--${size} icon-button--${variant} ${className}`.trim()}
      aria-label={label}
      title={props.title ?? label}
      data-focusable="true"
      data-pressable="true"
      {...props}
    >
      <Icon name={icon} size={iconSize ?? (size === 'small' ? 17 : 19)} />
    </button>
  )
}
