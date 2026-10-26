import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { MediaItem, NotificationItem, NotificationPage } from '../api/types'
import { Button } from '../shared/components/Button'
import { Icon } from '../shared/components/Icon'
import { Spinner } from '../shared/components/Spinner'
import { formatDate } from '../shared/utils/format'
import { useTranslation } from '../i18n/i18n'

interface NotificationsPageProps {
  onOpenSeries: (series: MediaItem) => void
}

function episodeCode(item: NotificationItem): string {
  return `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}`
}

export function NotificationsPage({ onOpenSeries }: NotificationsPageProps) {
  const { t } = useTranslation()
  const [page, setPage] = useState<NotificationPage | null>(null)
  const [error, setError] = useState('')
  const [openingID, setOpeningID] = useState('')

  const load = useCallback(async () => {
    setError('')
    try {
      const result = await api.notifications.list(500, 0)
      setPage(result)
      if (result.unread > 0) {
        await api.notifications.markAllRead()
        window.dispatchEvent(new Event('kodumedia:notifications-read'))
        setPage((current) => current ? {
          ...current,
          unread: 0,
          items: current.items.map((item) => item.readAt ? item : { ...item, readAt: new Date().toISOString() }),
        } : current)
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t('Teavitusi ei saanud laadida.'))
    }
  }, [t])

  useEffect(() => { void load() }, [load])

  const openSeries = async (item: NotificationItem) => {
    setOpeningID(item.id)
    try {
      onOpenSeries(await api.media.get(item.seriesId))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t('Seriaali ei saanud avada.'))
    } finally {
      setOpeningID('')
    }
  }

  return (
    <section className="page notifications-page" aria-labelledby="notifications-title">
      <header className="notifications-page__header">
        <div>
          <span className="eyebrow">{t('KoduMedia')}</span>
          <h1 id="notifications-title">{t('Teavitused')}</h1>
          <p>{t('Siin näed seriaalide uusi ilmunud osi, mida sinu serveris veel ei ole, ning varasemaid teavitusi.')}</p>
        </div>
        <Button variant="secondary" icon="refresh" onClick={() => void load()}>{t('Värskenda')}</Button>
      </header>

      {error ? <p className="inline-error"><Icon name="alert" size={17} />{error}</p> : null}

      {!page && !error ? (
        <div className="notifications-page__state"><Spinner label={t('Laen teavitusi')} /><span>{t('Laen teavitusi…')}</span></div>
      ) : null}

      {page?.items.length === 0 ? (
        <div className="notifications-empty">
          <Icon name="bell" size={30} />
          <h2>{t('Teavitusi pole')}</h2>
          <p>{t('Kui jälgitavale seriaalile ilmub uus osa, mida serveris veel pole, kuvatakse see siin.')}</p>
        </div>
      ) : null}

      {page?.items.length ? (
        <div className="notifications-list" aria-label={t('Kõik teavitused')}>
          {page.items.map((item) => (
            <button
              type="button"
              className={`notification-row ${item.resolvedAt ? 'notification-row--resolved' : ''}`}
              key={item.id}
              onClick={() => void openSeries(item)}
              disabled={openingID === item.id}
              data-focusable="true"
            >
              <span className="notification-row__icon"><Icon name="tv" size={18} /></span>
              <span className="notification-row__copy">
                <span className="notification-row__title">
                  <strong>{item.seriesTitle}</strong>
                  <small>{item.resolvedAt ? t('Nüüd serveris olemas') : t('Pole veel serveris')}</small>
                </span>
                <span className="notification-row__episode">{episodeCode(item)}{item.episodeTitle ? ` · ${item.episodeTitle}` : ''}</span>
                <span className="notification-row__date">{item.releaseDate ? t('Ilmus {{date}}', { date: formatDate(item.releaseDate) }) : t('Uus osa on ilmunud')}</span>
              </span>
              <Icon name="chevronRight" size={19} />
            </button>
          ))}
        </div>
      ) : null}
    </section>
  )
}
