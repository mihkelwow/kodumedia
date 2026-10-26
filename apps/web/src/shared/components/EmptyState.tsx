import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'

export function EmptyState({ icon = 'film', title, description, action }: { icon?: IconName; title: string; description: string; action?: ReactNode }) {
  return (
    <section className="empty-state">
      <div className="empty-state__icon"><Icon name={icon} size={30} /></div>
      <h2>{title}</h2>
      <p>{description}</p>
      {action ? <div className="empty-state__action">{action}</div> : null}
    </section>
  )
}
