import { useEffect, useRef } from 'react'
import { tr } from '../../i18n/i18n'
import { Icon } from '../../shared/components/Icon'

interface SettingsFeedbackProps {
  error: string
  message: string
  onDismiss: () => void
}

/** Keeps completion and error feedback in view regardless of the active settings section. */
export function SettingsFeedback({ error, message, onDismiss }: SettingsFeedbackProps) {
  const element = useRef<HTMLDivElement>(null)
  const text = error || message

  useEffect(() => {
    if (error) element.current?.focus({ preventScroll: true })
  }, [error])

  useEffect(() => {
    if (!message || error) return
    const timeout = window.setTimeout(onDismiss, 6000)
    return () => window.clearTimeout(timeout)
  }, [error, message, onDismiss])

  if (!text) return null

  return (
    <div
      ref={element}
      className={`settings-feedback ${error ? 'settings-feedback--error' : 'settings-feedback--success'}`}
      role={error ? 'alert' : 'status'}
      aria-live={error ? 'assertive' : 'polite'}
      tabIndex={error ? -1 : undefined}
    >
      <span className="settings-feedback__icon"><Icon name={error ? 'alert' : 'check'} size={18} /></span>
      <span className="settings-feedback__copy">{text}</span>
      <button type="button" className="settings-feedback__close" aria-label={tr('Sulge teade')} onClick={onDismiss} data-focusable="true">
        <Icon name="close" size={17} />
      </button>
    </div>
  )
}
