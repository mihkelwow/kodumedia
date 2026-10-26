import { useCallback, useEffect, useState } from 'react'
import { api } from '../../api/client'
import type { StorageCategory, StorageReport } from '../../api/types'
import { tr, type TranslationKey } from '../../i18n/i18n'
import { Button } from '../../shared/components/Button'
import { Icon, type IconName } from '../../shared/components/Icon'
import { ProgressBar } from '../../shared/components/ProgressBar'
import { Spinner } from '../../shared/components/Spinner'
import { formatBytes } from '../../shared/utils/format'

const categoryPresentation: Record<StorageCategory['id'], { title: TranslationKey; description: TranslationKey; icon: IconName }> = {
  updates: { title: 'Uuenduste tööfailid', description: 'Kaks eelmist rollback-varukoopiat ja viimase uuenduse diagnostika', icon: 'download' },
  'optimized-media': { title: 'Optimeeritud videod', description: 'Vaatamise ajal loodud korduvkasutatavad videosegmendid', icon: 'film' },
  transcodes: { title: 'Ajutised transkodeeringud', description: 'Ainult aktiivse taasesituse tööfailid', icon: 'play' },
  'music-artwork': { title: 'Muusika pildid', description: 'Normaliseeritud ja deduplitseeritud albumi- ning loo artwork', icon: 'music' },
  'music-metadata': { title: 'Muusika metaandmed', description: 'Sõrmejäljed, audioanalüüs ja kataloogivastused', icon: 'sparkles' },
  themes: { title: 'Seriaalide tunnusmuusika', description: 'ThemerrDB kaudu hangitud tunnusmeloodiad', icon: 'tv' },
  'subtitles-cache': { title: 'Teisendatud subtiitrid', description: 'Meediafailidest loodud WebVTT vahemälu', icon: 'captions' },
  'subtitles-downloaded': { title: 'Allalaaditud subtiitrid', description: 'Kasutaja valitud subtiitrifaile ei kustutata automaatselt', icon: 'download' },
  logs: { title: 'Logid', description: 'Serveri ja viimaste uuenduste piiratud diagnostikalogid', icon: 'info' },
}

function localClient(): boolean {
  const host = window.location.hostname.toLowerCase()
  return host === 'localhost' || host === '127.0.0.1' || host === '::1' || host === '[::1]'
}

function bytes(value: number): string {
  return formatBytes(value) || '0 B'
}

export function StorageSettings() {
  const [report, setReport] = useState<StorageReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<'cleanup' | 'clear-optimized' | ''>('')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const canManage = localClient()

  const load = useCallback(async () => {
    setError('')
    try {
      setReport(await api.system.storage())
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Kettakasutust ei saanud laadida.'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  const manage = async (action: 'cleanup' | 'clear-optimized') => {
    if (action === 'clear-optimized' && !window.confirm(tr('Kas tühjendada kogu mitteaktiivne optimeeritud videovahemälu? Seda saab hiljem vaatamise ajal uuesti luua.'))) return
    setBusy(action)
    setError('')
    setMessage('')
    try {
      const result = await api.system.manageStorage(action)
      setReport(result.report)
      setMessage(result.freedBytes > 0
        ? tr('Kettahooldus vabastas {{size}}.', { size: bytes(result.freedBytes) })
        : tr('Hooldus ei leidnud praegu vabastatavat ruumi.'))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Kettahooldus ebaõnnestus.'))
    } finally {
      setBusy('')
    }
  }

  if (loading) return <div className="settings-loading"><Spinner label={tr('Kontrollin kettakasutust')} />{tr('Arvutan vahemälude mahtu…')}</div>

  return (
    <div className="settings-subsection-list storage-settings">
      {error ? <div className="notice notice--error"><Icon name="alert" size={18} />{error}</div> : null}
      {message ? <div className="notice notice--success"><Icon name="check" size={18} />{message}</div> : null}
      {report ? (
        <>
          <section className="settings-form-section settings-subsection storage-summary">
            <header className="settings-form-section__heading">
              <span className="settings-form-section__icon"><Icon name="server" size={20} /></span>
              <span><strong>{tr('Andmekataloog')}</strong><small title={report.dataDirectory}>{report.dataDirectory}</small></span>
            </header>
            <div className="storage-summary__facts">
              <span><small>{tr('Kasutus kokku')}</small><strong>{bytes(report.dataSizeBytes)}</strong></span>
              <span><small>{tr('Vaba kettaruum')}</small><strong>{bytes(report.diskFreeBytes)}</strong></span>
              <span><small>{tr('Kaitstud reserv')}</small><strong>{bytes(report.requiredReserveBytes)}</strong></span>
            </div>
            <ProgressBar
              value={report.diskTotalBytes > 0 ? (report.diskFreeBytes / report.diskTotalBytes) * 100 : 0}
              label={tr('Vaba kettaruum')}
              className="storage-summary__bar"
            />
          </section>

          <div className="storage-category-grid">
            {report.categories.map((category) => {
              const presentation = categoryPresentation[category.id]
              const usage = category.limitBytes ? (category.sizeBytes / category.limitBytes) * 100 : 0
              return (
                <article className="storage-category" key={category.id}>
                  <span className="storage-category__icon"><Icon name={presentation.icon} size={19} /></span>
                  <div className="storage-category__copy">
                    <strong>{tr(presentation.title)}</strong>
                    <small>{tr(presentation.description)}</small>
                    <span className="storage-category__facts">
                      <b>{bytes(category.sizeBytes)}</b>
                      {category.limitBytes ? tr('Piir: {{size}}', { size: bytes(category.limitBytes) }) : null}
                      {category.retentionCount ? tr('Säilitatakse kaks eelmist versiooni.') : null}
                    </span>
                    {category.limitBytes ? <ProgressBar value={usage} /> : null}
                  </div>
                </article>
              )
            })}
          </div>

          <section className="settings-form-section settings-subsection storage-actions">
            <header className="settings-form-section__heading">
              <span className="settings-form-section__icon"><Icon name="trash" size={20} /></span>
              <span><strong>{tr('Ohutu hooldus')}</strong><small>{tr('Eemalda orvud ja jõusta salvestatud mahupiirid')}</small></span>
            </header>
            <p className="field-help">{tr('Hooldus ei kustuta meediafaile ega kasutaja allalaaditud subtiitreid. Aktiivse taasesituse faile ja parajasti kasutatavat vahemälu ei puudutata.')}</p>
            <div className="storage-actions__buttons">
              <Button icon="refresh" disabled={!canManage || Boolean(busy)} onClick={() => void manage('cleanup')}>
                {busy === 'cleanup' ? tr('Puhastan…') : tr('Puhasta orvud ja rakenda piirid')}
              </Button>
              <Button variant="danger" icon="trash" disabled={!canManage || Boolean(busy)} onClick={() => void manage('clear-optimized')}>
                {busy === 'clear-optimized' ? tr('Puhastan…') : tr('Tühjenda optimeeritud vahemälu')}
              </Button>
            </div>
            {!canManage ? <p className="field-help"><strong>{tr('Ainult localhost:')}</strong> {tr('Kettalt faile eemaldavaid tegevusi saab käivitada ainult serveriarvuti brauserist.')}</p> : null}
          </section>
        </>
      ) : null}
    </div>
  )
}
