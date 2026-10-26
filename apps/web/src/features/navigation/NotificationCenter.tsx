import { useCallback, useEffect, useState } from 'react'
import { api } from '../../api/client'
import type { NotificationItem, NotificationPage } from '../../api/types'
import { AppLink } from '../../shared/components/AppLink'
import { Icon } from '../../shared/components/Icon'
import { IconButton } from '../../shared/components/IconButton'
import { Popover } from '../../shared/components/Popover'
import { formatDate } from '../../shared/utils/format'
import { useTranslation } from '../../i18n/i18n'

interface NotificationCenterProps {
  pathname: string
  onNavigate: (path: string) => void
}

function episodeCode(item: NotificationItem): string {
  return `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}`
}

export function NotificationCenter({ pathname, onNavigate }: NotificationCenterProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [page, setPage] = useState<NotificationPage>({ items: [], total: 0, unread: 0, limit: 5, offset: 0 })
  const [loading, setLoading] = useState(true)

  const load = useCallback(async (): Promise<NotificationPage | undefined> => {
    try {
      const result = await api.notifications.list(5, 0)
      setPage(result)
      return result
    } catch {
      // Keep the header usable when notification synchronization is temporarily unavailable.
      return undefined
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
    const markRead = () => setPage((current) => ({
      ...current,
      unread: 0,
      items: current.items.map((item) => item.readAt ? item : { ...item, readAt: new Date().toISOString() }),
    }))
    const refreshOnFocus = () => void load()
    window.addEventListener('kodumedia:notifications-read', markRead)
    window.addEventListener('focus', refreshOnFocus)
    return () => {
      window.removeEventListener('kodumedia:notifications-read', markRead)
      window.removeEventListener('focus', refreshOnFocus)
    }
  }, [load])

  const openNotifications = async () => {
    const latest = await load()
    if ((latest?.unread ?? page.unread) <= 0) return
    setPage((current) => ({
      ...current,
      unread: 0,
      items: current.items.map((item) => item.readAt ? item : { ...item, readAt: new Date().toISOString() }),
    }))
    try {
      await api.notifications.markAllRead()
      window.dispatchEvent(new Event('kodumedia:notifications-read'))
    } catch {
      void load()
    }
  }

  const changeOpen = (nextOpen: boolean) => {
    setOpen(nextOpen)
    if (nextOpen) void openNotifications()
  }

  const openAll = (path: string) => {
    setOpen(false)
    onNavigate(path)
  }

  return (
    <Popover
      open={open}
      onOpenChange={changeOpen}
      placement="bottom-end"
      rootClassName="header-notifications"
      className="player-menu-panel header-notifications__panel"
      ariaLabel={t('Teavitused')}
      role="menu"
      autoFocus
      trigger={(
        <IconButton
          className={`app-header__action-link header-notifications__button ${pathname === '/notifications' ? 'app-header__action-link--active' : ''} ${page.unread > 0 ? 'header-notifications__button--unread' : ''}`}
          icon="bell"
          label={t('Teavitused')}
        />
      )}
    >
      <div className="player-menu-panel__header">
        <strong>{t('Teavitused')}</strong>
        <small>{page.total > 0 ? t('Viimased teavitused') : t('Uusi teavitusi pole')}</small>
      </div>

      <div className="player-menu-panel__options header-notifications__list">
        {loading ? (
          <div className="header-notifications__empty">{t('Laen teavitusi…')}</div>
        ) : page.items.length === 0 ? (
          <div className="header-notifications__empty">{t('Uusi teavitusi pole')}</div>
        ) : page.items.map((item) => (
          <button
            key={item.id}
            type="button"
            className="player-menu-option header-notifications__item"
            role="menuitem"
            onClick={() => openAll('/notifications')}
          >
            <span className="header-notifications__item-icon"><Icon name="tv" size={16} /></span>
            <span className="player-menu-option__copy">
              <strong>{item.seriesTitle}</strong>
              <small>{episodeCode(item)}{item.episodeTitle ? ` · ${item.episodeTitle}` : ''}</small>
              <small>{item.releaseDate ? t('Ilmus {{date}}', { date: formatDate(item.releaseDate) }) : t('Uus osa on ilmunud')}</small>
            </span>
          </button>
        ))}
      </div>

      <div className="player-menu-panel__footer">
        <AppLink
          href="/notifications"
          onNavigate={openAll}
          className="player-menu-option player-menu-option--action header-notifications__all"
          role="menuitem"
          data-focusable="true"
        >
          <span className="player-menu-option__copy"><strong>{t('Vaata kõiki teavitusi')}</strong></span>
          <Icon name="chevronRight" size={16} />
        </AppLink>
      </div>
    </Popover>
  )
}
