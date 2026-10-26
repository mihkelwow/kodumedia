import { useSyncExternalStore } from 'react'
import sourceCatalog from './locales/et-EE.json'
import englishCatalog from './locales/en-US.json'

export interface LocaleMetadata {
  locale: string
  name: string
  nativeName: string
  direction?: 'ltr' | 'rtl'
  authors?: string[]
}

interface LocaleFile {
  _meta: LocaleMetadata
  messages: Record<string, string>
}

export interface LocaleOption extends LocaleMetadata {
  completeness: number
}

export type TranslationKey = keyof typeof sourceCatalog.messages
export type TranslationValues = Record<string, string | number | boolean | null | undefined>

const STORAGE_KEY = 'kodumedia:ui-language'
const SOURCE_LOCALE = 'et-EE'
const DEFAULT_LOCALE = 'en-US'
const modules: Record<string, LocaleFile> = {
  './locales/et-EE.json': sourceCatalog as LocaleFile,
  './locales/en-US.json': englishCatalog as LocaleFile,
}
const catalogs = new Map<string, LocaleFile>()

for (const [path, localeFile] of Object.entries(modules)) {
  const locale = localeFile?._meta?.locale?.trim()
  if (!locale || !localeFile.messages) {
    console.warn(`KoduMedia ignored an invalid translation catalog: ${path}`)
    continue
  }
  catalogs.set(locale, localeFile)
}

if (!catalogs.has(SOURCE_LOCALE)) {
  throw new Error(`KoduMedia source translation catalog ${SOURCE_LOCALE} is missing.`)
}
if (!catalogs.has(DEFAULT_LOCALE)) {
  throw new Error(`KoduMedia default translation catalog ${DEFAULT_LOCALE} is missing.`)
}

const fallbackMessages = catalogs.get(DEFAULT_LOCALE)?.messages ?? {}
const listeners = new Set<() => void>()

function normalizeLocale(value?: string | null): string | undefined {
  if (!value) return undefined
  const normalized = value.replace('_', '-').trim()
  if (catalogs.has(normalized)) return normalized
  const language = normalized.split('-')[0]?.toLowerCase()
  return [...catalogs.keys()].find((locale) => locale.toLowerCase().split('-')[0] === language)
}

function readStoredLocale(): string | undefined {
  if (typeof window === 'undefined') return undefined
  try {
    return normalizeLocale(window.localStorage.getItem(STORAGE_KEY))
  } catch {
    return undefined
  }
}

function storeLocale(locale: string): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // Language selection still works for the current tab when storage is blocked.
  }
}

function preferredLocale(): string {
  // A fresh browser starts in English. An explicit browser-local choice always
  // wins, so existing users keep the interface language they selected.
  return readStoredLocale() ?? DEFAULT_LOCALE
}

let currentLocale = preferredLocale()

function applyDocumentLocale(locale: string): void {
  if (typeof document === 'undefined') return
  const metadata = catalogs.get(locale)?._meta
  document.documentElement.lang = locale
  document.documentElement.dir = metadata?.direction ?? 'ltr'
}

applyDocumentLocale(currentLocale)

function interpolate(message: string, values?: TranslationValues): string {
  if (!values) return message
  return message.replace(/\{\{\s*([\w.-]+)\s*\}\}/g, (match, key: string) => {
    const value = values[key]
    return value === undefined || value === null ? match : String(value)
  })
}

export function tr(message: TranslationKey, values?: TranslationValues): string {
  const translated = catalogs.get(currentLocale)?.messages[message] ?? fallbackMessages[message] ?? message
  return interpolate(translated, values)
}

export function trExternal(message?: string | null): string {
  if (!message) return ''
  const exact = catalogs.get(currentLocale)?.messages[message] ?? fallbackMessages[message]
  if (exact) return exact

  const sentences = message.split(/(?<=\.)\s+/)
  if (sentences.length > 1) {
    const translated = sentences.map((sentence) => trExternal(sentence))
    if (translated.some((sentence, index) => sentence !== sentences[index])) return translated.join(' ')
  }

  const progress = message.match(/^Analüüsitud (\d+) \/ (\d+)$/)
  if (progress) return tr('Analüüsitud {{current}} / {{total}}', { current: progress[1], total: progress[2] })

  const executable = message.match(/^(FFmpeg|FFprobe) (tee pole seadistatud|kontroll aegus|töötab)\.$/)
  if (executable) {
    const key = executable[2] === 'tee pole seadistatud'
      ? '{{name}} tee pole seadistatud.'
      : executable[2] === 'kontroll aegus'
        ? '{{name}} kontroll aegus.'
        : '{{name}} töötab.'
    return tr(key, { name: executable[1] })
  }

  const executableFailure = message.match(/^(FFmpeg|FFprobe) ei käivitunud: (.+)$/)
  if (executableFailure) return tr('{{name}} ei käivitunud: {{details}}', { name: executableFailure[1], details: executableFailure[2] })

  const tmdbFailure = message.match(/^TMDb ühendus ebaõnnestus: (.+)$/)
  if (tmdbFailure) return tr('TMDb ühendus ebaõnnestus: {{details}}', { details: tmdbFailure[1] })
  const tmdbHTTP = message.match(/^TMDb vastas HTTP (\d+)\.$/)
  if (tmdbHTTP) return tr('TMDb vastas HTTP {{status}}.', { status: tmdbHTTP[1] })

  const sameGenre = message.match(/^Sama žanr: (.+)$/)
  if (sameGenre) return tr('Sama žanr: {{genres}}', { genres: sameGenre[1] })

  const selectedEncoder = message.match(/^Valitud (.+) hardware encoder ei käivitunud\.$/)
  if (selectedEncoder) return tr('Valitud {{encoder}} hardware encoder ei käivitunud.', { encoder: selectedEncoder[1] })
  const decoderFallback = message.match(/^Hardware decoder (.+) ebaõnnestus; sama hardware encoder proovitakse tarkvaralise dekodeerimisega\.$/)
  if (decoderFallback) return tr('Hardware decoder {{decoder}} ebaõnnestus; sama hardware encoder proovitakse tarkvaralise dekodeerimisega.', { decoder: decoderFallback[1] })
  const nextEncoder = message.match(/^(.+) ei käivitunud selle faili jaoks; KoduMedia proovib järgmist encoder'it\.$/)
  if (nextEncoder) return tr("{{encoder}} ei käivitunud selle faili jaoks; KoduMedia proovib järgmist encoder'it.", { encoder: nextEncoder[1] })
  const encoderStartupSpeed = message.match(/^Encoder'i käivituskiirus ([\d.]+)x jäi alla ([\d.]+)x turvavaru\.$/)
  if (encoderStartupSpeed) return tr("Encoder'i käivituskiirus {{speed}}x jäi alla {{minimum}}x turvavaru.", { speed: encoderStartupSpeed[1], minimum: encoderStartupSpeed[2] })
  const encoderAverageSpeed = message.match(/^Encoder'i keskmine kiirus ([\d.]+)x jäi alla ([\d.]+)x turvavaru\.$/)
  if (encoderAverageSpeed) return tr("Encoder'i keskmine kiirus {{speed}}x jäi alla {{minimum}}x turvavaru.", { speed: encoderAverageSpeed[1], minimum: encoderAverageSpeed[2] })
  const vmafAdjustment = message.match(/^VMAF ([\d.]+) põhjal kasutab KoduMedia CRF (\d+)\.$/)
  if (vmafAdjustment) return tr('VMAF {{score}} põhjal kasutab KoduMedia CRF {{crf}}.', { score: vmafAdjustment[1], crf: vmafAdjustment[2] })

  const subtitleLanguage = message.match(/^subtiitrite keel "?([^" ]+)"? peab olema 2–3 täheline kood$/)
  if (subtitleLanguage) return tr('subtiitrite keel {{language}} peab olema 2–3 täheline kood', { language: subtitleLanguage[1] })

  const pathMessages: Array<[RegExp, TranslationKey]> = [
    [/^ZIP-failitee väljub paketist: (.+)$/, 'ZIP-failitee väljub paketist: {{path}}'],
    [/^sümbollink ei ole paketis lubatud: (.+)$/, 'sümbollink ei ole paketis lubatud: {{path}}'],
    [/^fail asub paketijuurest väljaspool: (.+)$/, 'fail asub paketijuurest väljaspool: {{path}}'],
    [/^faili SHA-256 ei ühti manifestiga: (.+)$/, 'faili SHA-256 ei ühti manifestiga: {{path}}'],
    [/^faili lahtipakitud suurus ei ühti manifestiga: (.+)$/, 'faili lahtipakitud suurus ei ühti manifestiga: {{path}}'],
    [/^faili suurus ei ühti manifestiga: (.+)$/, 'faili suurus ei ühti manifestiga: {{path}}'],
    [/^vigane SHA-256 väärtus failil (.+)$/, 'vigane SHA-256 väärtus failil {{path}}'],
  ]
  for (const [pattern, key] of pathMessages) {
    const match = message.match(pattern)
    if (match) return tr(key, { path: match[1] })
  }

  const detailMessages: Array<[RegExp, TranslationKey]> = [
    [/^OpenSubtitles otsing ebaõnnestus: (.+)$/, 'OpenSubtitles otsing ebaõnnestus: {{details}}'],
    [/^OpenSubtitles allalaadimispäring ebaõnnestus: (.+)$/, 'OpenSubtitles allalaadimispäring ebaõnnestus: {{details}}'],
    [/^OpenSubtitles sisselogimine ebaõnnestus: (.+)$/, 'OpenSubtitles sisselogimine ebaõnnestus: {{details}}'],
    [/^subtiitrifaili allalaadimine ebaõnnestus: (.+)$/, 'subtiitrifaili allalaadimine ebaõnnestus: {{details}}'],
  ]
  for (const [pattern, key] of detailMessages) {
    const match = message.match(pattern)
    if (match) return tr(key, { details: match[1] })
  }

  const openSubtitlesCredentials = message.match(/^(.+): kontrolli API-võtit ja kasutajakontot \((.+)\)$/)
  if (openSubtitlesCredentials) return tr('{{operation}}: kontrolli API-võtit ja kasutajakontot ({{status}})', { operation: openSubtitlesCredentials[1], status: openSubtitlesCredentials[2] })
  const openSubtitlesRateLimit = message.match(/^(.+): päringulimiit on täis$/)
  if (openSubtitlesRateLimit) return tr('{{operation}}: päringulimiit on täis', { operation: openSubtitlesRateLimit[1] })

  const prefixes: Array<[string, TranslationKey]> = [
    ['invalid input: ', 'Vigane sisend: {{details}}'],
    ['unavailable: ', 'Teenus pole saadaval: {{details}}'],
    ['unsupported: ', 'Toetamata toiming: {{details}}'],
    ['forbidden: ', 'Toiming pole lubatud: {{details}}'],
    ['conflict: ', 'Toimingu konflikt: {{details}}'],
  ]
  for (const [prefix, key] of prefixes) {
    if (message.startsWith(prefix)) return tr(key, { details: trExternal(message.slice(prefix.length)) })
  }
  return message
}

export function trCount(one: TranslationKey, other: TranslationKey, count: number, values?: TranslationValues): string {
  const locale = getLocale()
  const form = new Intl.PluralRules(locale).select(count)
  return tr(form === 'one' ? one : other, { ...values, count })
}

export function getLocale(): string {
  return currentLocale
}

function updateLocale(locale: string, persist: boolean): void {
  const supported = normalizeLocale(locale)
  if (!supported) return
  if (persist) storeLocale(supported)
  if (supported === currentLocale) return
  currentLocale = supported
  applyDocumentLocale(supported)
  listeners.forEach((listener) => listener())
}

export function setLocale(locale: string): void {
  updateLocale(locale, true)
}

if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event) => {
    if (event.key !== STORAGE_KEY || !event.newValue) return
    updateLocale(event.newValue, false)
  })
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function useLocale(): string {
  return useSyncExternalStore(subscribe, getLocale, () => DEFAULT_LOCALE)
}

export function useTranslation(): { locale: string; t: typeof tr; setLocale: typeof setLocale } {
  const locale = useLocale()
  return { locale, t: tr, setLocale }
}

export function availableLocales(): LocaleOption[] {
  const total = Math.max(1, Object.keys(catalogs.get(SOURCE_LOCALE)?.messages ?? {}).length)
  return [...catalogs.values()]
    .map((catalog) => ({
      ...catalog._meta,
      direction: catalog._meta.direction ?? 'ltr',
      completeness: Math.round((Object.keys(catalog.messages).length / total) * 100),
    }))
    .sort((left, right) => left.nativeName.localeCompare(right.nativeName, left.locale))
}

export function catalogKeys(locale: string): string[] {
  return Object.keys(catalogs.get(locale)?.messages ?? {}).sort()
}

export function resetLocaleForTests(locale = DEFAULT_LOCALE): void {
  currentLocale = normalizeLocale(locale) ?? DEFAULT_LOCALE
  applyDocumentLocale(currentLocale)
  listeners.forEach((listener) => listener())
}
