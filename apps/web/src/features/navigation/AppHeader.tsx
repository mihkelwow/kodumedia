import { useMemo, useState } from 'react'
import type { Library } from '../../api/types'
import { AppLink } from '../../shared/components/AppLink'
import { IconButton } from '../../shared/components/IconButton'
import { Menu } from '../../shared/components/Menu'
import { Popover } from '../../shared/components/Popover'
import { Icon } from '../../shared/components/Icon'
import { useTheme } from '../theme/ThemeProvider'
import { tr } from '../../i18n/i18n'
import { NotificationCenter } from './NotificationCenter'

interface AppHeaderProps {
  libraries: Library[]
  pathname: string
  onNavigate: (path: string) => void
  onOpenSearch: () => void
}

function libraryPath(library: Library): string {
  if (library.system && library.kind === 'music') return '/music'
  return `/library/${encodeURIComponent(library.id)}`
}

export function AppHeader({ libraries, pathname, onNavigate, onOpenSearch }: AppHeaderProps) {
  const { theme, toggleTheme } = useTheme()
  const [collectionsOpen, setCollectionsOpen] = useState(false)
  const systemLibraries = useMemo(() => {
    const order: Record<string, number> = { movies: 0, series: 1, music: 2 }
    return libraries
      .filter((library) => library.system && library.kind in order)
      .sort((left, right) => (order[left.kind] ?? 99) - (order[right.kind] ?? 99))
  }, [libraries])
  const customLibraries = useMemo(() => libraries.filter((library) => !library.system), [libraries])

  const labels: Record<string, string> = { movies: tr('Filmid'), series: tr('Seriaalid'), music: tr('Muusika') }
  const primary = [
    { path: '/', label: tr('Avaleht') },
    ...systemLibraries.map((library) => ({ path: libraryPath(library), label: labels[library.kind] ?? library.name })),
  ]

  return (
    <header className="app-header">
      <AppLink href="/" onNavigate={onNavigate} className="app-header__brand" aria-label={tr('Kodu avaleht')} data-focusable="true">
        <img className="app-header__brand-logo app-header__brand-logo--light" src="/kodu-media-logo.png" alt="Kodu" />
        <img className="app-header__brand-logo app-header__brand-logo--dark" src="/kodu-media-logo-dark.png" alt="" aria-hidden="true" />
      </AppLink>

      <div className="app-header__nav-wrap">
        <nav className="app-header__nav" aria-label={tr('Põhinavigatsioon')}>
          <IconButton
            className="app-header__search"
            icon="search"
            label={tr('Otsi meediast')}
            title={tr('Otsi (Ctrl+K)')}
            onClick={onOpenSearch}
          />
          {primary.map((item) => {
            const active = pathname === item.path || (item.path === '/music' && pathname.startsWith('/music/'))
            return (
              <AppLink
                key={item.path}
                href={item.path}
                onNavigate={onNavigate}
                className={`app-header__nav-link ${active ? 'app-header__nav-link--active' : ''}`}
                aria-current={active ? 'page' : undefined}
                data-focusable="true"
              >
                {item.label}
              </AppLink>
            )
          })}
          {customLibraries.length > 0 ? (
            <Popover
              open={collectionsOpen}
              onOpenChange={setCollectionsOpen}
              placement="bottom-start"
              rootClassName="app-header__collections"
              className="app-header__collections-menu"
              ariaLabel={tr('Kogud')}
              role="menu"
              autoFocus
              trigger={(
                <button
                  type="button"
                  className={`app-header__nav-link ${customLibraries.some((library) => pathname === libraryPath(library)) ? 'app-header__nav-link--active' : ''}`}
                  data-focusable="true"
                >
                  {tr('Kogud')}
                </button>
              )}
            >
              <Menu role="presentation">
                {customLibraries.map((library) => {
                  const active = pathname === libraryPath(library)
                  return (
                    <AppLink
                      key={library.id}
                      href={libraryPath(library)}
                      onNavigate={(path) => { setCollectionsOpen(false); onNavigate(path) }}
                      className={`ui-menu-item ${active ? 'ui-menu-item--selected' : ''}`}
                      role="menuitem"
                      data-focusable="true"
                    >
                      <span className="ui-menu-item__copy"><strong>{library.name}</strong></span>
                      {active ? <Icon name="check" size={16} /> : null}
                    </AppLink>
                  )
                })}
              </Menu>
            </Popover>
          ) : null}
        </nav>
      </div>

      <div className="app-header__actions">
        <NotificationCenter pathname={pathname} onNavigate={onNavigate} />
        <IconButton
          icon={theme === 'dark' ? 'sun' : 'moon'}
          onClick={toggleTheme}
          label={theme === 'dark' ? tr('Kasuta heledat teemat') : tr('Kasuta tumedat teemat')}
          title={theme === 'dark' ? tr('Hele teema') : tr('Tume teema')}
        />
        <AppLink
          href="/settings"
          onNavigate={onNavigate}
          className={`icon-button icon-button--medium icon-button--ghost app-header__action-link ${pathname === '/settings' ? 'app-header__action-link--active' : ''}`}
          aria-label={tr('Seaded')}
          title={tr('Seaded')}
          data-focusable="true"
          data-pressable="true"
        >
          <Icon name="settings" size={19} />
        </AppLink>
      </div>
    </header>
  )
}
