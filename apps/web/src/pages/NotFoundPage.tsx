import { tr } from '../i18n/i18n'
import { Button } from '../shared/components/Button'
import { EmptyState } from '../shared/components/EmptyState'

export function NotFoundPage({ onNavigate }: { onNavigate: (path: string) => void }) {
  return (
    <div className="page page--centered">
      <EmptyState
        icon="search"
        title={tr('Seda vaadet ei leitud')}
        description={tr('Aadress ei vasta ühelegi meediakogule ega seadete lehele.')}
        action={<Button variant="primary" icon="home" onClick={() => onNavigate('/')}>{tr('Tagasi avalehele')}</Button>}
      />
    </div>
  )
}
