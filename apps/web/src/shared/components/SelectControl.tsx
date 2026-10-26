import type { ReactNode, SelectHTMLAttributes } from 'react'
import { Icon } from './Icon'

interface SelectControlProps extends SelectHTMLAttributes<HTMLSelectElement> {
  children: ReactNode
  controlClassName?: string
}

/**
 * KoduMedia CVI select. The native select is retained for keyboard, screen reader,
 * touch and platform menu behaviour; the wrapper supplies the shared visual shell.
 */
export function SelectControl({ children, controlClassName = '', ...props }: SelectControlProps) {
  return (
    <span className={`select-control ${controlClassName}`.trim()}>
      <select data-focusable="true" {...props}>{children}</select>
      <span className="select-control__chevron" aria-hidden="true">
        <Icon name="chevronDown" size={15} />
      </span>
    </span>
  )
}
