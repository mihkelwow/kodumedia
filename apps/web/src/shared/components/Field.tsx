import { cloneElement, isValidElement, type ReactElement, type ReactNode } from 'react'

interface FieldProps {
  id: string
  label: ReactNode
  children: ReactElement<any>
  help?: ReactNode
  error?: ReactNode
  className?: string
  compact?: boolean
}

export function Field({ id, label, children, help, error, className = '', compact = false }: FieldProps) {
  const helpId = help ? `${id}-help` : undefined
  const errorId = error ? `${id}-error` : undefined
  const describedBy = [children.props['aria-describedby'], helpId, errorId].filter(Boolean).join(' ') || undefined
  const control = isValidElement(children)
    ? cloneElement(children, { 'aria-describedby': describedBy } as Record<string, unknown>)
    : children

  return (
    <label className={`ui-field ${compact ? 'ui-field--compact' : ''} ${className}`.trim()} htmlFor={id}>
      <span className="ui-field__label">{label}</span>
      {control}
      {help ? <small id={helpId} className="ui-field__help">{help}</small> : null}
      {error ? <small id={errorId} className="ui-field__error" role="alert">{error}</small> : null}
    </label>
  )
}
