import { useEffect, useState } from 'react'
import type { Library } from '../api/types'
import { SystemStatus } from '../features/settings/SystemStatus'
import { useTranslation, type TranslationKey } from '../i18n/i18n'
import { Icon, type IconName } from '../shared/components/Icon'
import { SelectControl } from '../shared/components/SelectControl'

interface SettingsNavigationItem {
  href: string
  label: TranslationKey
  description: TranslationKey
  icon: IconName
}

const navigationGroups: Array<{ label: TranslationKey; items: SettingsNavigationItem[] }> = [
  { label: 'Server', items: [
    { href: '#settings-overview', label: 'Server ja töövalmidus', description: 'Olek, tööriistad ja turvalisus', icon: 'server' },
  ] },
  { label: 'Meedia', items: [
    { href: '#settings-metadata', label: 'Meediakogud ja metaandmed', description: 'Kaustad, TMDb, IMDb ja muusika', icon: 'folder' },
    { href: '#settings-quality', label: 'Taasesitus ja heli', description: 'Kvaliteet, vahemälu ja tunnusmuusika', icon: 'play' },
    { href: '#settings-storage', label: 'Kettakasutus', description: 'Vahemälud, uuendused ja vaba ruum', icon: 'server' },
    { href: '#settings-subtitles', label: 'Subtiitrid', description: 'Keeled ja OpenSubtitles', icon: 'captions' },
  ] },
  { label: 'Kasutajaliides', items: [
    { href: '#settings-language', label: 'Välimus ja keel', description: 'Kasutajaliidese keel ja teema', icon: 'sun' },
  ] },
  { label: 'Hooldus', items: [
    { href: '#settings-update', label: 'Uuendused ja diagnostika', description: 'Väljalasked ja tehniline olek', icon: 'download' },
  ] },
]

const navigation = navigationGroups.flatMap((group) => group.items)

function initialSection(): string {
  const hash = window.location.hash
  return navigation.some((item) => item.href === hash) ? hash : navigation[0]?.href || '#settings-overview'
}

export function SettingsPage({ libraries, onLibrariesChanged }: { libraries: Library[]; onLibrariesChanged: () => Promise<void> }) {
  const { t } = useTranslation()
  const [activeSection, setActiveSection] = useState(initialSection)

  useEffect(() => {
    const sections = navigation
      .map((item) => document.querySelector<HTMLElement>(item.href))
      .filter((section): section is HTMLElement => Boolean(section))
    if (!sections.length || typeof IntersectionObserver === 'undefined') return

    const observer = new IntersectionObserver((entries) => {
      const visible = entries
        .filter((entry) => entry.isIntersecting)
        .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
      if (visible[0]?.target.id) setActiveSection(`#${visible[0].target.id}`)
    }, { rootMargin: '-18% 0px -68% 0px', threshold: [0, 0.1, 0.35] })

    sections.forEach((section) => observer.observe(section))
    return () => observer.disconnect()
  }, [])

  return (
    <div className="page settings-page cinematic-page">
      <header className="settings-page__header">
        <h1>{t('Seaded')}</h1>
        <p>{t('Halda serverit, meediakogusid ja vaatamiskogemust ühest selgest kohast.')}</p>
      </header>

      <div className="settings-layout">
        <aside className="settings-navigation" aria-label={t('Seadete jaotised')}>
          <div className="settings-navigation__intro">
            <strong>{t('Seadete jaotised')}</strong>
            <small>{t('Vali teema või liigu otse vajaliku seadistuseni.')}</small>
          </div>
          <nav className="settings-navigation__desktop">
            {navigation.map((item) => {
              const active = activeSection === item.href
              return (
                <a
                  key={item.href}
                  href={item.href}
                  className={`settings-navigation__link${active ? ' settings-navigation__link--active' : ''}`}
                  aria-current={active ? 'location' : undefined}
                  onClick={() => setActiveSection(item.href)}
                >
                  <span className="settings-navigation__icon"><Icon name={item.icon} size={17} /></span>
                  <span className="settings-navigation__link-copy">
                    <strong>{t(item.label)}</strong>
                    <small>{t(item.description)}</small>
                  </span>
                  <Icon className="settings-navigation__chevron" name="chevronRight" size={15} />
                </a>
              )
            })}
          </nav>
          <label className="settings-navigation__mobile" htmlFor="settings-section-select">
            <span className="settings-navigation__mobile-label">{t('Seadete jaotis')}</span>
            <SelectControl
              controlClassName="settings-navigation__mobile-control"
              id="settings-section-select"
              value={activeSection}
              aria-label={t('Seadete jaotis')}
              onChange={(event) => {
                const href = event.target.value
                setActiveSection(href)
                window.location.hash = href
              }}
            >
              {navigationGroups.map((group) => (
                <optgroup key={group.label} label={t(group.label)}>
                  {group.items.map((item) => <option key={item.href} value={item.href}>{t(item.label)}</option>)}
                </optgroup>
              ))}
            </SelectControl>
          </label>
        </aside>

        <div className="settings-content">
          <SystemStatus libraries={libraries} onLibrariesChanged={onLibrariesChanged} />
        </div>
      </div>
    </div>
  )
}
