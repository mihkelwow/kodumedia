import type { InputHTMLAttributes, ReactNode } from 'react'

interface SwitchProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: ReactNode
  description?: ReactNode
  danger?: boolean
  className?: string
}

/**
 * Shared settings toggle built on the existing KoduMedia check-field CVI.
 * The native checkbox remains the interaction primitive for keyboard, touch and assistive technology.
 */
export function Switch({ label, description, danger = false, className = '', ...props }: SwitchProps) {
  return (
    <label className={`check-field ${danger ? 'check-field--danger' : ''} ${className}`.trim()}>
      <input type="checkbox" data-focusable="true" {...props} />
      <span>
        <strong>{label}</strong>
        {description ? <small>{description}</small> : null}
      </span>
    </label>
  )
}
