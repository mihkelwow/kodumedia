import type { ReactNode } from 'react'
import { Icon, type IconName } from '../../shared/components/Icon'

interface SettingsGroupProps {
  id: string
  title: string
  description: string
  icon: IconName
  children: ReactNode
  featured?: boolean
  aside?: ReactNode
  scope?: string
  behavior?: string
}

export function SettingsScope({ scope, behavior }: { scope: string; behavior?: string }) {
  return (
    <div className="settings-scope" aria-label={`${scope}${behavior ? `. ${behavior}` : ''}`}>
      <span><Icon name="info" size={14} />{scope}</span>
      {behavior ? <span>{behavior}</span> : null}
    </div>
  )
}

export function SettingsGroup({ id, title, description, icon, children, featured = false, aside, scope, behavior }: SettingsGroupProps) {
  return (
    <section id={id} className={`settings-group${featured ? ' settings-group--featured' : ''}`}>
      <header className="settings-group__heading">
        <span className="settings-group__icon"><Icon name={icon} size={21} /></span>
        <div className="settings-group__copy">
          <h2>{title}</h2>
          <p>{description}</p>
          {scope ? <SettingsScope scope={scope} behavior={behavior} /> : null}
        </div>
        {aside ? <div className="settings-group__aside">{aside}</div> : null}
      </header>
      <div className="settings-group__body">{children}</div>
    </section>
  )
}
