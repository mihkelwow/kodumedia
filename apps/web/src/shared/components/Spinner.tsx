import { tr } from '../../i18n/i18n'

export function Spinner({ label = tr('Laen') }: { label?: string }) {
  return (
    <div className="spinner" role="status" aria-label={label}>
      <span />
    </div>
  )
}
