import { Button } from '../../shared/components/Button'
import { tr } from '../../i18n/i18n'

interface SettingsSaveBarProps {
  dirty: boolean
  busy: boolean
  saving: boolean
  testing: boolean
  onReset: () => void
  onTest: () => void
  onSave: () => void
}

/** Appears only when the form differs from the saved server configuration. */
export function SettingsSaveBar({ dirty, busy, saving, testing, onReset, onTest, onSave }: SettingsSaveBarProps) {
  if (!dirty) return null

  return (
    <div id="settings-save" className="system-settings-actions system-settings-actions--dirty" role="region" aria-label={tr('Salvestamata muudatused')}>
      <span className="system-settings-actions__hint" aria-live="polite">
        <strong>{tr('Sul on salvestamata muudatusi.')}</strong>
        <small>{tr('Muudatused jõustuvad pärast serverisse salvestamist.')}</small>
      </span>
      <div className="system-settings-actions__buttons">
        <Button type="button" className="system-settings-actions__reset" variant="ghost" icon="close" title={tr('Loobu muudatustest')} disabled={busy} onClick={onReset}>{tr('Loobu muudatustest')}</Button>
        <Button type="button" className="system-settings-actions__test" variant="secondary" icon="refresh" title={tr('Kontrolli väärtusi')} disabled={busy} onClick={onTest}>{testing ? tr('Kontrollin…') : tr('Kontrolli väärtusi')}</Button>
        <Button type="button" className="system-settings-actions__save" variant="primary" icon="check" disabled={busy} onClick={onSave}>{saving ? tr('Salvestan…') : tr('Salvesta seaded')}</Button>
      </div>
    </div>
  )
}
