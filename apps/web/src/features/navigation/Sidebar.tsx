import { tr } from '../../i18n/i18n'
import type { Library, LibraryKind } from '../../api/types'
import { AppLink } from '../../shared/components/AppLink'
import { Icon, type IconName } from '../../shared/components/Icon'

function libraryIcon(kind: LibraryKind): IconName {
  switch (kind) {
    case 'movies': return 'film'
    case 'series': return 'tv'
    case 'music': return 'music'
    default: return 'folder'
  }
}

interface NavItemProps {
  path: string
  label: string
  icon: IconName
  active: boolean
  onNavigate: (path: string) => void
}

function NavItem({ path, label, icon, active, onNavigate }: NavItemProps) {
  return (
    <AppLink
      href={path}
      onNavigate={onNavigate}
      className={`sidebar__item ${active ? 'sidebar__item--active' : ''}`}
      aria-current={active ? 'page' : undefined}
      data-focusable="true"
    >
      <Icon name={icon} size={20} />
      <span>{label}</span>
    </AppLink>
  )
}

export function Sidebar({ libraries, pathname, onNavigate }: { libraries: Library[]; pathname: string; onNavigate: (path: string) => void }) {
  const system = libraries.filter((library) => library.system)
  const custom = libraries.filter((library) => !library.system)
  return (
    <aside className="sidebar" aria-label={tr('Põhinavigatsioon')}>
      <AppLink className="brand" href="/" onNavigate={onNavigate} data-focusable="true" aria-label={tr('Kodu avaleht')}>
        <img className="brand__logo" src="/kodu-media-logo.png" alt="Kodu" />
      </AppLink>

      <nav className="sidebar__nav">
        <div className="sidebar__group">
          <span className="sidebar__label">{tr('Sirvi')}</span>
          <NavItem path="/" label={tr('Avaleht')} icon="home" active={pathname === '/'} onNavigate={onNavigate} />
          {system.map((library) => {
            const path = `/library/${encodeURIComponent(library.id)}`
            return <NavItem key={library.id} path={path} label={library.name} icon={libraryIcon(library.kind)} active={pathname === path} onNavigate={onNavigate} />
          })}
        </div>

        {custom.length > 0 ? (
          <div className="sidebar__group">
            <span className="sidebar__label">{tr('Minu kogud')}</span>
            {custom.map((library) => {
              const path = `/library/${encodeURIComponent(library.id)}`
              return <NavItem key={library.id} path={path} label={library.name} icon={libraryIcon(library.kind)} active={pathname === path} onNavigate={onNavigate} />
            })}
          </div>
        ) : null}
      </nav>

      <div className="sidebar__footer">
        <NavItem path="/settings" label={tr('Seaded')} icon="settings" active={pathname === '/settings'} onNavigate={onNavigate} />
      </div>
    </aside>
  )
}
