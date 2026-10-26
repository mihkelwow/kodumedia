import { availableLocales, useTranslation } from '../../i18n/i18n'
import { Field } from '../../shared/components/Field'
import { SelectControl } from '../../shared/components/SelectControl'

export function LanguageSettings() {
  const { locale, setLocale, t } = useTranslation()
  const locales = availableLocales()

  return (
    <section id="settings-interface-language" className="settings-panel settings-panel--language settings-subsection">
      <div className="settings-panel__heading">
        <div>
          <span className="eyebrow">{t('Keel')}</span>
          <h2>{t('Rakenduse keel')}</h2>
          <p>{t('Valik muudab KoduMedia kasutajaliidese keelt selles brauseris. Metaandmete ja subtiitrite keeled on eraldi seaded.')}</p>
        </div>
      </div>
      <div className="settings-form-grid settings-form-grid--language">
        <Field
          id="interface-language"
          className="settings-field"
          label={t('Kasutajaliidese keel')}
          help={t('Keelevalik salvestatakse sellesse brauserisse ja rakendub kohe ilma serveri taaskäivitamiseta.')}
        >
          <SelectControl
            id="interface-language"
            aria-label={t('Kasutajaliidese keel')}
            value={locale}
            onChange={(event) => setLocale(event.target.value)}
          >
            {locales.map((option) => (
              <option key={option.locale} value={option.locale}>
                {option.nativeName}{option.nativeName !== option.name ? ` · ${option.name}` : ''}
              </option>
            ))}
          </SelectControl>
        </Field>
      </div>
    </section>
  )
}
