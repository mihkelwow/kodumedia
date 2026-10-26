import { useTheme, type Theme } from '../theme/ThemeProvider'
import { Icon } from '../../shared/components/Icon'
import { tr, type TranslationKey } from '../../i18n/i18n'

const options: Array<{ value: Theme; title: TranslationKey; description: TranslationKey; icon: 'moon' | 'sun' }> = [
  { value: 'dark', title: 'Tume', description: 'Kinolik ja silmadele mahe', icon: 'moon' },
  { value: 'light', title: 'Hele', description: 'Selge ja päevavalguses loetav', icon: 'sun' },
]

export function ThemeSettings() {
  const { theme, setTheme } = useTheme()
  return (
    <section id="settings-appearance" className="settings-panel settings-panel--appearance settings-subsection">
      <div className="settings-panel__heading">
        <div>
          <span className="eyebrow">{tr('Välimus')}</span>
          <h2>{tr('Vali KoduMedia ilme')}</h2>
          <p>{tr('Mõlemad teemad kasutavad sama CVI-d. Valik salvestatakse sellesse brauserisse ega muutu serveri taaskäivitamisel.')}</p>
        </div>
      </div>
      <div className="theme-options" role="radiogroup" aria-label={tr('Värviteema')}>
        {options.map((option) => (
          <button
            key={option.value}
            className={`theme-option ${theme === option.value ? 'theme-option--active' : ''}`}
            onClick={() => setTheme(option.value)}
            role="radio"
            aria-checked={theme === option.value}
            data-focusable="true"
          >
            <span className="theme-option__preview" data-preview-theme={option.value}>
              <span /><span /><span />
            </span>
            <span className="theme-option__copy">
              <span className="theme-option__title"><Icon name={option.icon} size={18} />{tr(option.title)}</span>
              <small>{tr(option.description)}</small>
            </span>
            {theme === option.value ? <span className="theme-option__check"><Icon name="check" size={16} /></span> : null}
          </button>
        ))}
      </div>
    </section>
  )
}
