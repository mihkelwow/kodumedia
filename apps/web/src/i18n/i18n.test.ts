import { afterEach, describe, expect, it } from 'vitest'
import { availableLocales, catalogKeys, resetLocaleForTests, setLocale, tr, trExternal } from './i18n'

const placeholders = (value: string) => [...value.matchAll(/\{\{\s*([\w.-]+)\s*\}\}/g)].map((match) => match[1]).sort()

describe('i18n catalogs', () => {
  afterEach(() => {
    window.localStorage.clear()
    resetLocaleForTests()
  })

  it('loads complete Estonian and English catalogs with matching message keys', () => {
    const locales = availableLocales()
    expect(locales.map((locale) => locale.locale)).toEqual(expect.arrayContaining(['et-EE', 'en-US']))
    expect(locales.find((locale) => locale.locale === 'en-US')?.completeness).toBe(100)
    expect(catalogKeys('en-US')).toEqual(catalogKeys('et-EE'))
    expect(catalogKeys('et-EE').length).toBeGreaterThan(500)
  })

  it('uses English for a browser with no saved language choice', () => {
    window.localStorage.clear()
    resetLocaleForTests()
    expect(tr('Muusika')).toBe('Music')
    expect(document.documentElement.lang).toBe('en-US')
  })

  it('keeps every placeholder intact and every translated value non-empty', () => {
    for (const key of catalogKeys('et-EE')) {
      resetLocaleForTests('en-US')
      const translated = tr(key)
      expect(translated.trim(), key).not.toBe('')
      expect(placeholders(translated), key).toEqual(placeholders(key))
    }
  })

  it('switches locale, persists the choice and interpolates values', () => {
    resetLocaleForTests('et-EE')
    setLocale('en-US')
    expect(tr('Rakenduse keel')).toBe('Application language')
    expect(tr('Test {{value}}', { value: 42 })).toBe('Test 42')
    expect(window.localStorage.getItem('kodumedia:ui-language')).toBe('en-US')
    expect(document.documentElement.lang).toBe('en-US')
    expect(document.documentElement.dir).toBe('ltr')
  })

  it('translates dynamic messages received from the server', () => {
    resetLocaleForTests('en-US')
    expect(trExternal('Sama žanr: Drama, Crime')).toBe('Same genre: Drama, Crime')
    expect(trExternal('Valitud NVENC hardware encoder ei käivitunud.')).toBe('The selected NVENC hardware encoder did not start.')
    expect(trExternal('invalid input: faili SHA-256 ei ühti manifestiga: kodumedia.exe')).toBe('Invalid input: The file SHA-256 does not match the manifest: kodumedia.exe')
    expect(trExternal('Kvaliteedimootor loob kliendile sobiva kõrge kvaliteediga HLS-voo. VMAF 94.2 põhjal kasutab KoduMedia CRF 20.')).toBe('The quality engine is creating a high-quality HLS stream suitable for the client. Based on VMAF 94.2, KoduMedia is using CRF 20.')
    expect(trExternal('Režissöör')).toBe('Director')
  })
})
