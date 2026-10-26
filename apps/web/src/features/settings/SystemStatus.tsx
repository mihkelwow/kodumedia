import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../../api/client'
import type { ComponentStatus, Job, Library, SystemConfiguration, SystemSettingsUpdate, UpdateStatus } from '../../api/types'
import { Button } from '../../shared/components/Button'
import { Disclosure } from '../../shared/components/Disclosure'
import { Field } from '../../shared/components/Field'
import { FileBrowserDialog } from '../../shared/components/FileBrowserDialog'
import { Icon, type IconName } from '../../shared/components/Icon'
import { SelectControl } from '../../shared/components/SelectControl'
import { Spinner } from '../../shared/components/Spinner'
import { ProgressBar } from '../../shared/components/ProgressBar'
import { Switch } from '../../shared/components/Switch'
import { formatBytes } from '../../shared/utils/format'
import { openNativeFilePicker } from '../../shared/utils/filePicker'
import { getLocale, tr, trExternal } from '../../i18n/i18n'
import { SERIES_THEME_SETTINGS_CHANGED_EVENT } from '../themes/events'
import { SettingsFeedback } from './SettingsFeedback'
import { SettingsSaveBar } from './SettingsSaveBar'
import { SettingsGroup, SettingsScope } from './SettingsGroup'
import { LibrarySettings } from './LibrarySettings'
import { LanguageSettings } from './LanguageSettings'
import { ThemeSettings } from './ThemeSettings'
import { StorageSettings } from './StorageSettings'

const defaultIMDbSource = 'https://datasets.imdbws.com/title.ratings.tsv.gz'

interface FormState {
  ffmpegPath: string
  ffprobePath: string
  tmdbToken: string
  clearTmdbToken: boolean
  language: string
  imdbRatingsSource: string
  musicMetadataEnabled: boolean
  shazamEnabled: boolean
  fingerprintEnabled: boolean
  catalogFallbackEnabled: boolean
  audioAnalysisEnabled: boolean
  generatedArtworkEnabled: boolean
  acoustIdApiKey: string
  seriesThemePlayback: SystemConfiguration['settings']['themeMusic']['seriesPlayback']
  clearAcoustIdApiKey: boolean
  preferredSubtitleLanguages: string
  openSubtitlesApiKey: string
  clearOpenSubtitlesApiKey: boolean
  openSubtitlesUsername: string
  clearOpenSubtitlesUsername: boolean
  openSubtitlesPassword: string
  clearOpenSubtitlesPassword: boolean
  qualityProfile: SystemConfiguration['settings']['quality']['profile']
  preferredVideoCodec: SystemConfiguration['settings']['quality']['preferredCodec']
  hardwarePreference: SystemConfiguration['settings']['quality']['hardwarePreference']
  cachePolicy: SystemConfiguration['settings']['quality']['cachePolicy']
  cacheMaxGb: number
  cacheFreeReserveGb: number
  cacheFreeReservePercent: number
  gracePeriodSeconds: number
  vmafEnabled: boolean
  vmafTarget: number
  vmafSampleSeconds: number
}

const emptyForm: FormState = {
  ffmpegPath: 'ffmpeg',
  ffprobePath: 'ffprobe',
  tmdbToken: '',
  clearTmdbToken: false,
  language: 'en-US',
  imdbRatingsSource: defaultIMDbSource,
  musicMetadataEnabled: true,
  shazamEnabled: true,
  fingerprintEnabled: true,
  catalogFallbackEnabled: true,
  audioAnalysisEnabled: true,
  generatedArtworkEnabled: true,
  acoustIdApiKey: '',
  seriesThemePlayback: 'always',
  clearAcoustIdApiKey: false,
  preferredSubtitleLanguages: 'en',
  openSubtitlesApiKey: '',
  clearOpenSubtitlesApiKey: false,
  openSubtitlesUsername: '',
  clearOpenSubtitlesUsername: false,
  openSubtitlesPassword: '',
  clearOpenSubtitlesPassword: false,
  qualityProfile: 'original',
  preferredVideoCodec: 'auto',
  hardwarePreference: 'auto',
  cachePolicy: 'watched',
  cacheMaxGb: 50,
  cacheFreeReserveGb: 10,
  cacheFreeReservePercent: 10,
  gracePeriodSeconds: 15,
  vmafEnabled: true,
  vmafTarget: 95,
  vmafSampleSeconds: 12,
}

type PickerField = 'ffmpegPath' | 'ffprobePath' | 'imdbRatingsSource'

interface PickerState {
  field: PickerField
  title: string
  extensions: string[]
}

function containingDirectory(path: string): string {
  const trimmed = path.trim()
  if (/^https?:\/\//i.test(trimmed)) return ''
  const index = Math.max(trimmed.lastIndexOf('\\'), trimmed.lastIndexOf('/'))
  return index > 0 ? trimmed.slice(0, index) : ''
}

function toForm(configuration: SystemConfiguration): FormState {
  const subtitles = configuration.settings.subtitles
  return {
    ffmpegPath: configuration.settings.ffmpegPath,
    ffprobePath: configuration.settings.ffprobePath,
    tmdbToken: '',
    clearTmdbToken: false,
    language: configuration.settings.metadata.language,
    imdbRatingsSource: configuration.settings.metadata.imdbRatingsSource || defaultIMDbSource,
    musicMetadataEnabled: configuration.settings.music?.metadataEnabled ?? true,
    shazamEnabled: configuration.settings.music?.shazamEnabled ?? true,
    fingerprintEnabled: configuration.settings.music?.fingerprintEnabled ?? true,
    catalogFallbackEnabled: configuration.settings.music?.catalogFallbackEnabled ?? true,
    audioAnalysisEnabled: configuration.settings.music?.audioAnalysisEnabled ?? true,
    generatedArtworkEnabled: configuration.settings.music?.generatedArtworkEnabled ?? true,
    acoustIdApiKey: '',
    seriesThemePlayback: configuration.settings.themeMusic?.seriesPlayback ?? 'always',
    clearAcoustIdApiKey: false,
    preferredSubtitleLanguages: (subtitles?.preferredLanguages?.length ? subtitles.preferredLanguages : ['et', 'en']).join(', '),
    openSubtitlesApiKey: '',
    clearOpenSubtitlesApiKey: false,
    openSubtitlesUsername: '',
    clearOpenSubtitlesUsername: false,
    openSubtitlesPassword: '',
    clearOpenSubtitlesPassword: false,
    qualityProfile: configuration.settings.quality.profile,
    preferredVideoCodec: configuration.settings.quality.preferredCodec,
    hardwarePreference: configuration.settings.quality.hardwarePreference,
    cachePolicy: configuration.settings.quality.cachePolicy,
    cacheMaxGb: configuration.settings.quality.cacheMaxGb,
    cacheFreeReserveGb: configuration.settings.quality.cacheFreeReserveGb ?? 10,
    cacheFreeReservePercent: configuration.settings.quality.cacheFreeReservePercent ?? 10,
    gracePeriodSeconds: configuration.settings.quality.gracePeriodSeconds,
    vmafEnabled: configuration.settings.quality.vmafEnabled,
    vmafTarget: configuration.settings.quality.vmafTarget,
    vmafSampleSeconds: configuration.settings.quality.vmafSampleSeconds,
  }
}

function toUpdate(form: FormState): SystemSettingsUpdate {
  const token = form.tmdbToken.trim()
  const acoustIdApiKey = form.acoustIdApiKey.trim()
  const apiKey = form.openSubtitlesApiKey.trim()
  const username = form.openSubtitlesUsername.trim()
  const password = form.openSubtitlesPassword
  const preferredLanguages = Array.from(new Set(
    form.preferredSubtitleLanguages
      .split(/[\s,;]+/)
      .map((value) => value.trim().toLowerCase())
      .filter(Boolean),
  ))
  return {
    ffmpegPath: form.ffmpegPath.trim(),
    ffprobePath: form.ffprobePath.trim(),
    metadata: {
      ...(token ? { tmdbToken: token } : {}),
      ...(form.clearTmdbToken ? { clearTmdbToken: true } : {}),
      language: form.language.trim(),
      imdbRatingsSource: form.imdbRatingsSource.trim() || defaultIMDbSource,
    },
    music: {
      metadataEnabled: form.musicMetadataEnabled,
      shazamEnabled: form.shazamEnabled,
      fingerprintEnabled: form.fingerprintEnabled,
      catalogFallbackEnabled: form.catalogFallbackEnabled,
      audioAnalysisEnabled: form.audioAnalysisEnabled,
      generatedArtworkEnabled: form.generatedArtworkEnabled,
      ...(acoustIdApiKey ? { acoustIdApiKey } : {}),
      ...(form.clearAcoustIdApiKey ? { clearAcoustIdApiKey: true } : {}),
    },
    themeMusic: {
      seriesPlayback: form.seriesThemePlayback,
    },
    subtitles: {
      preferredLanguages: preferredLanguages.length ? preferredLanguages : ['et', 'en'],
      ...(apiKey ? { openSubtitlesApiKey: apiKey } : {}),
      ...(form.clearOpenSubtitlesApiKey ? { clearOpenSubtitlesApiKey: true } : {}),
      ...(username ? { openSubtitlesUsername: username } : {}),
      ...(form.clearOpenSubtitlesUsername ? { clearOpenSubtitlesUsername: true } : {}),
      ...(password ? { openSubtitlesPassword: password } : {}),
      ...(form.clearOpenSubtitlesPassword ? { clearOpenSubtitlesPassword: true } : {}),
    },
    quality: {
      profile: form.qualityProfile,
      preferredCodec: form.preferredVideoCodec,
      hardwarePreference: form.hardwarePreference,
      cachePolicy: form.cachePolicy,
      cacheMaxGb: Number(form.cacheMaxGb),
      cacheFreeReserveGb: Number(form.cacheFreeReserveGb),
      cacheFreeReservePercent: Number(form.cacheFreeReservePercent),
      gracePeriodSeconds: Number(form.gracePeriodSeconds),
      vmafEnabled: form.vmafEnabled,
      vmafTarget: Number(form.vmafTarget),
      vmafSampleSeconds: Number(form.vmafSampleSeconds),
    },
  }
}

function formatTimestamp(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(getLocale(), { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

function StatusPill({ status, optional = false }: { status: ComponentStatus; optional?: boolean }) {
  const state = status.refreshing ? 'optional' : status.available ? 'ok' : optional ? 'optional' : 'warning'
  return (
    <span className={`status-pill status-pill--${state}`}>
      <Icon name={status.refreshing ? 'refresh' : status.available ? 'check' : optional ? 'info' : 'alert'} size={15} />
      {status.refreshing ? tr('Värskendab') : status.available ? tr('Töötab') : optional ? tr('Valikuline') : tr('Vajab seadistamist')}
    </span>
  )
}

function StatusCard({ icon, title, description, status, optional = false }: { icon: IconName; title: string; description: string; status: ComponentStatus; optional?: boolean }) {
  const facts = [
    status.sizeBytes ? formatBytes(status.sizeBytes) : '',
    status.updatedAt ? tr('Uuendatud {{time}}', { time: formatTimestamp(status.updatedAt) }) : '',
    status.lastCheckedAt ? tr('Kontrollitud {{time}}', { time: formatTimestamp(status.lastCheckedAt) }) : '',
  ].filter(Boolean)
  return (
    <div className={`status-card status-card--detailed ${status.available ? 'status-card--ok' : 'status-card--attention'}`}>
      <span className="status-card__icon"><Icon name={icon} /></span>
      <div className="status-card__copy">
        <strong>{title}</strong>
        <small>{description}</small>
        <p>{trExternal(status.message) || (status.available ? tr('Komponent on saadaval.') : tr('Komponent pole saadaval.'))}</p>
        {status.version ? <code title={status.version}>{status.version}</code> : null}
        {facts.length ? <small className="status-card__facts">{facts.join(' · ')}</small> : null}
      </div>
      <StatusPill status={status} optional={optional} />
    </div>
  )
}

function isActive(job: Job): boolean {
  return job.status === 'queued' || job.status === 'running'
}

function isLocalUpdateClient(): boolean {
  if (typeof window === 'undefined') return false
  const host = window.location.hostname.toLowerCase()
  return host === 'localhost' || host === '127.0.0.1' || host === '::1' || host === '[::1]'
}

function updateStateLabel(status: UpdateStatus | null): string {
  if (!status) return tr('Olek pole laaditud')
  switch (status.state) {
    case 'preparing': return tr('Kontrollib paketti')
    case 'installing': return tr('Paigaldab')
    case 'restarting': return tr('Taaskäivitab')
    case 'succeeded': return tr('Paigaldatud')
    case 'failed': return tr('Ebaõnnestus')
    default: return tr('Valmis')
  }
}

interface SystemStatusProps {
  libraries?: Library[]
  onLibrariesChanged?: () => Promise<void>
}

export function SystemStatus({ libraries = [], onLibrariesChanged = async () => {} }: SystemStatusProps = {}) {
  const [configuration, setConfiguration] = useState<SystemConfiguration | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [savedForm, setSavedForm] = useState<FormState>(emptyForm)
  const [initialSetupOpen, setInitialSetupOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<'save' | 'test' | 'imdb' | 'music' | 'scan' | ''>('')
  const [picking, setPicking] = useState<PickerField | ''>('')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [picker, setPicker] = useState<PickerState | null>(null)
  const [imdbJob, setIMDbJob] = useState<Job | null>(null)
  const [musicJob, setMusicJob] = useState<Job | null>(null)
  const [softwareUpdate, setSoftwareUpdate] = useState<UpdateStatus | null>(null)
  const [updateBusy, setUpdateBusy] = useState(false)
  const updateInput = useRef<HTMLInputElement>(null)
  const mounted = useRef(true)
  const setupDecisionMade = useRef(false)
  const localUpdateClient = isLocalUpdateClient()

  useEffect(() => () => { mounted.current = false }, [])

  const load = useCallback(async (replaceForm = true) => {
    setError('')
    try {
      const [current, update] = await Promise.all([
        api.system.configuration(),
        localUpdateClient ? api.system.updateStatus() : Promise.resolve(null),
      ])
      if (!mounted.current) return
      setConfiguration(current)
      setSoftwareUpdate(update)
      if (replaceForm) {
        const nextForm = toForm(current)
        setForm(nextForm)
        setSavedForm(nextForm)
      }
    } catch (caught) {
      if (mounted.current) setError(caught instanceof Error ? caught.message : tr('Süsteemi seadeid ei saanud laadida.'))
    } finally {
      if (mounted.current) setLoading(false)
    }
  }, [localUpdateClient])

  useEffect(() => { void load() }, [load])

  const dirty = useMemo(
    () => configuration !== null && JSON.stringify(toUpdate(form)) !== JSON.stringify(toUpdate(savedForm)),
    [configuration, form, savedForm],
  )

  const setupComplete = useMemo(() => Boolean(
    configuration
      && configuration.status.ffmpeg.available
      && configuration.status.ffprobe.available
      && libraries.some((library) => library.sources.length > 0),
  ), [configuration, libraries])

  useEffect(() => {
    if (!configuration || loading || setupDecisionMade.current) return
    setupDecisionMade.current = true
    setInitialSetupOpen(!setupComplete)
  }, [configuration, loading, setupComplete])

  const dismissFeedback = useCallback(() => {
    setError('')
    setMessage('')
  }, [])

  useEffect(() => {
    if (!dirty) return
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  const resetChanges = () => {
    setForm(savedForm)
    setError('')
    setMessage(tr('Salvestamata muudatused tühistati.'))
  }

  const change = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((current) => ({ ...current, [key]: value }))
    setMessage('')
  }

  const run = async (mode: 'save' | 'test') => {
    setBusy(mode)
    setError('')
    setMessage('')
    try {
      const result = mode === 'save'
        ? await api.system.update(toUpdate(form))
        : await api.system.test(toUpdate(form))
      setConfiguration(result)
      if (mode === 'save') {
        const nextForm = toForm(result)
        setForm(nextForm)
        setSavedForm(nextForm)
        window.dispatchEvent(new CustomEvent(SERIES_THEME_SETTINGS_CHANGED_EVENT))
        setMessage(tr('Seaded salvestati. Puuduvad filmi- ja seriaalipildid ning kirjeldused lisatakse taustal automaatselt.'))
      } else {
        setMessage(tr('Kontroll lõpetati. Olek vastab vormil olevatele väärtustele; neid pole veel salvestatud.'))
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Seadete kontroll ebaõnnestus.'))
    } finally {
      setBusy('')
    }
  }

  const scanAllLibraries = async () => {
    const targets = libraries.filter((library) => library.sources.length > 0)
    if (targets.length === 0) {
      setMessage('')
      setError(tr('Skannimiseks pole veel ühtegi kaustaga meediakogu.'))
      return
    }

    setBusy('scan')
    setError('')
    setMessage('')
    try {
      const results = await Promise.allSettled(targets.map((library) => api.libraries.scan(library.id)))
      const failed = results.filter((result) => result.status === 'rejected')
      if (failed.length > 0) throw new Error(tr('Mõne meediakogu skannimist ei saanud käivitada.'))
      setMessage(tr('Kõigi meediakogude skannimine käivitati.'))
      try {
        await onLibrariesChanged()
      } catch {
        // Scans are already queued; a list refresh failure must not report them as failed.
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Skannimist ei saanud käivitada.'))
    } finally {
      setBusy('')
    }
  }

  const chooseFile = async (field: PickerField, title: string, extensions: string[]) => {
    setPicking(field)
    setError('')
    try {
      const result = await openNativeFilePicker({ mode: 'file', title, initialPath: form[field], extensions })
      if (result.kind === 'selected') {
        change(field, result.path)
      } else if (result.kind === 'unavailable') {
        setPicker({ field, title, extensions })
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Windowsi failivalijat ei saanud avada.'))
    } finally {
      setPicking('')
    }
  }

  const waitForJob = async (initial: Job, onUpdate: (job: Job) => void): Promise<Job> => {
    let current = initial
    onUpdate(current)
    while (isActive(current)) {
      await new Promise((resolve) => window.setTimeout(resolve, 700))
      current = await api.jobs.get(current.id)
      if (!mounted.current) return current
      onUpdate(current)
    }
    return current
  }

  const refreshIMDb = async () => {
    setBusy('imdb')
    setError('')
    setMessage('')
    try {
      const saved = await api.system.update(toUpdate(form))
      setConfiguration(saved)
      const nextForm = toForm(saved)
      setForm(nextForm)
      setSavedForm(nextForm)
      const completed = await waitForJob(await api.system.refreshIMDb(), setIMDbJob)
      if (completed.status === 'failed') throw new Error(trExternal(completed.error) || tr('IMDb andmestiku värskendamine ebaõnnestus.'))
      setMessage(tr('IMDb reitinguandmestik ja kataloogis olevad skoorid on ajakohased.'))
      await load(true)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('IMDb andmestiku värskendamine ebaõnnestus.'))
    } finally {
      setBusy('')
    }
  }


  const enrichMusic = async () => {
    setBusy('music')
    setError('')
    setMessage('')
    try {
      const saved = await api.system.update(toUpdate(form))
      setConfiguration(saved)
      const nextForm = toForm(saved)
      setForm(nextForm)
      setSavedForm(nextForm)
      const completed = await waitForJob(await api.music.enrich(), setMusicJob)
      if (completed.status === 'failed') throw new Error(trExternal(completed.error) || tr('Muusika rikastamine ebaõnnestus.'))
      setMessage(tr('Muusikateek on uuesti töödeldud. Puuduvad stiilid ja artwork on lisatud või asendatud neutraalse fallback-kaardiga.'))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Muusika rikastamine ebaõnnestus.'))
    } finally {
      setBusy('')
    }
  }

  const waitForUpdatedServer = async (targetVersion: string) => {
    const deadline = Date.now() + 120_000
    while (Date.now() < deadline && mounted.current) {
      await new Promise((resolve) => window.setTimeout(resolve, 1000))
      try {
        const health = await api.system.health()
        if (health.status === 'ok' && health.version === targetVersion) {
          const refreshed = new URL(window.location.href)
          refreshed.searchParams.set('_updated', String(Date.now()))
          window.location.replace(refreshed.toString())
          return
        }
      } catch {
        // A connection failure is expected while the server process restarts.
      }
    }
    throw new Error(tr('Server ei käivitunud kahe minuti jooksul uue versiooniga. Vaata data\\updates\\last-update.log faili.'))
  }

  const installRelease = async (file: File) => {
    if (!localUpdateClient || updateBusy) return
    const accepted = window.confirm(tr('Paigaldada uuenduspakett {{file}}? Server taaskäivitub automaatselt.', { file: file.name }))
    if (!accepted) return
    setUpdateBusy(true)
    setError('')
    setMessage('')
    try {
      const status = await api.system.installRelease(file)
      if (!mounted.current) return
      setSoftwareUpdate(status)
      if (!status.targetVersion) throw new Error(tr('Uuenduspaketi sihtversiooni ei saadud tuvastada.'))
      setMessage(tr('Versioon {{version}} on kontrollitud. Server taaskäivitub; hoia see leht avatuna.', { version: status.targetVersion }))
      await waitForUpdatedServer(status.targetVersion)
    } catch (caught) {
      if (!mounted.current) return
      setError(caught instanceof Error ? caught.message : tr('KoduMedia uuendamine ebaõnnestus.'))
      try {
        setSoftwareUpdate(await api.system.updateStatus())
      } catch {
        // The server may still be restarting; the persisted status remains available after recovery.
      }
    } finally {
      if (mounted.current) setUpdateBusy(false)
      if (updateInput.current) updateInput.current.value = ''
    }
  }

  return (
    <div className={`settings-sections${dirty ? ' settings-sections--dirty' : ''}`}>
      <SettingsFeedback error={error} message={message} onDismiss={dismissFeedback} />
      <section id="settings-overview" className="settings-group settings-group--overview">
        <div className="settings-group__hero">
                <div>
                  <h2>{tr('KoduMedia töövalmidus')}</h2>
                  <p>{tr('Siin näed kohe, kas esitamiseks, analüüsiks ja metaandmeteks vajalikud teenused töötavad. Muudatused jõustuvad ilma serveri taaskäivituseta.')}</p>
                  <SettingsScope scope={tr('Serveriülene')} behavior={tr('Tööriistateede muudatused vajavad salvestamist.')} />
                  <button
                    type="button"
                    className={`initial-setup-toggle${setupComplete ? ' initial-setup-toggle--complete' : ''}`}
                    aria-expanded={initialSetupOpen}
                    aria-controls="initial-setup-guide"
                    data-focusable="true"
                    onClick={() => setInitialSetupOpen((current) => !current)}
                  >
                    <span className="initial-setup-toggle__icon"><Icon name={setupComplete ? 'check' : 'sparkles'} size={18} /></span>
                    <span className="initial-setup-toggle__copy">
                      <strong>{tr('Esmane seadistus')}</strong>
                      <small>{setupComplete ? tr('Põhiseadistus on valmis.') : tr('Ühekordne seadistus võtab tavaliselt alla 10 minuti.')}</small>
                    </span>
                    <Icon className="initial-setup-toggle__chevron" name="chevronDown" size={18} />
                  </button>
                </div>
                {configuration ? <span className="version-badge">v{configuration.version}</span> : null}
              </div>
        <div className="settings-group__body">
          {initialSetupOpen ? (
                  <div id="initial-setup-guide" className="initial-setup-guide">
                    <header className="initial-setup-guide__intro">
                      <span className="initial-setup-guide__intro-icon"><Icon name="check" size={18} /></span>
                      <p>{tr('KoduMedia tuleb seadistada ainult üks kord ja see võtab tavaliselt alla 10 minuti. Kõik vajalikud API-võtmed on tasuta.')}</p>
                    </header>

                    <ol className="initial-setup-list">
                      <li className="initial-setup-step">
                        <span className="initial-setup-step__number" aria-hidden="true">1</span>
                        <div className="initial-setup-step__copy">
                          <strong>{tr('Lisa tasuta TMDb juurdepääsutunnus')}</strong>
                          <p>{tr('TMDb lisab filmidele ja seriaalidele kirjeldused, posterid, logod, näitlejad ning treilerid.')}</p>
                        </div>
                        <a className="initial-setup-step__action" data-focusable="true" href="#settings-metadata-provider">{tr('Seadista TMDb')}<Icon name="chevronRight" size={15} /></a>
                      </li>
                      <li className="initial-setup-step">
                        <span className="initial-setup-step__number" aria-hidden="true">2</span>
                        <div className="initial-setup-step__copy">
                          <strong>{tr('Seadista tasuta OpenSubtitles ja vali subtiitrikeeled')}</strong>
                        </div>
                        <a className="initial-setup-step__action" data-focusable="true" href="#settings-subtitles-provider">{tr('Seadista subtiitrid')}<Icon name="chevronRight" size={15} /></a>
                      </li>
                      <li className="initial-setup-step">
                        <span className="initial-setup-step__number" aria-hidden="true">3</span>
                        <div className="initial-setup-step__copy">
                          <strong>{tr('Lisa filmide, seriaalide ja muusika kaustad')}</strong>
                        </div>
                        <a className="initial-setup-step__action" data-focusable="true" href="#settings-libraries">{tr('Lisa meediakaustad')}<Icon name="chevronRight" size={15} /></a>
                      </li>
                      <li className="initial-setup-step initial-setup-step--actions">
                        <span className="initial-setup-step__number" aria-hidden="true">4</span>
                        <div className="initial-setup-step__copy">
                          <strong>{tr('Salvesta seaded ja skanni meediakogud')}</strong>
                        </div>
                        <span className="initial-setup-step__primary-actions">
                          <Button
                            type="button"
                            variant="primary"
                            icon="check"
                            className="initial-setup-primary-action"
                            disabled={Boolean(busy) || updateBusy}
                            onClick={() => void run('save')}
                          >
                            {busy === 'save' ? tr('Salvestan…') : tr('Salvesta seaded')}
                          </Button>
                          <Button
                            type="button"
                            variant="secondary"
                            icon="refresh"
                            className="initial-setup-primary-action"
                            disabled={Boolean(busy) || updateBusy || libraries.every((library) => library.sources.length === 0)}
                            onClick={() => void scanAllLibraries()}
                          >
                            {busy === 'scan' ? tr('Skannin kõiki kogusid…') : tr('Skanni kõik meediakogud')}
                          </Button>
                        </span>
                      </li>
                    </ol>

                    <div className="initial-setup-guide__music-note">
                      <span><Icon name="music" size={18} /></span>
                      <p>{tr('Muusika tuvastamine töötab tasuta ilma võtmeta. Ka valikuline AcoustID võti on tasuta.')}</p>
                      <a data-focusable="true" href="#settings-music-metadata">{tr('Muusika metaandmed')}<Icon name="chevronRight" size={15} /></a>
                    </div>

                    <div className="initial-setup-guide__tray-note">
                      <span><Icon name="server" size={18} /></span>
                      <p>{tr('Serveriarvuti KoduMedia system tray menüüst saad samuti skannida kõik kogud ja lülitada sisse automaatse käivitumise Windowsi sisselogimisel.')}</p>
                    </div>
                  </div>
                ) : null}
          {loading ? <div className="settings-loading"><Spinner label={tr('Kontrollin serverit')} />{tr('Kontrollin komponente…')}</div> : null}

                {configuration ? (
                  <div className="status-grid status-grid--components">
                    <StatusCard icon="film" title="FFmpeg" description={tr('Remux ja transkodeerimine')} status={configuration.status.ffmpeg} />
                    <StatusCard icon="sliders" title="FFprobe" description={tr("Failide, radade ja codec'ite analüüs")} status={configuration.status.ffprobe} />
                    <StatusCard icon="star" title="TMDb" description={tr('Ametlikud pealkirjad, kirjeldused ja pildid')} status={configuration.status.metadata.tmdb} />
                    <StatusCard icon="info" title={tr('IMDb reitingud')} description={tr('Serveri kohalik title.ratings vahemälu')} status={configuration.status.metadata.imdb} />
                  </div>
                ) : null}
          {!loading ? (
            <div className="settings-subsection-list">
              <section id="settings-tools" className="settings-form-section settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="sliders" size={20} /></span>
                            <span><strong>{tr('Meedia tööriistad')}</strong><small>{tr('FFmpeg ja FFprobe')}</small></span>
                          </header>
                          <Disclosure
                            title={tr('Täpsemad tööriistateed')}
                            description={tr('Muuda ainult siis, kui kasutad enda kontrollitud FFmpegi buildi.')}
                            icon="sliders"
                          >
                          <p className="field-help">
                            {tr("KoduMedia sisaldab Windows x64 FFmpegi ja FFprobe binaare ning kasutab neid vaikimisi. Muuda neid teid ainult siis, kui soovid kasutada enda kontrollitud build'i.")}
                            {' '}<a href="https://ffmpeg.org/download.html" target="_blank" rel="noreferrer">{tr('Ava FFmpegi ametlik allalaadimisleht')}</a>.
                          </p>
                          <label className="settings-field" htmlFor="system-ffmpeg-path">
                            <span>{tr('FFmpeg käivitusfail')}</span>
                            <div className="path-control">
                              <input id="system-ffmpeg-path" aria-label={tr('FFmpeg käivitusfail')} value={form.ffmpegPath} onChange={(event) => change('ffmpegPath', event.target.value)} placeholder={'C:\\ffmpeg\\bin\\ffmpeg.exe'} autoComplete="off" data-focusable="true" />
                              <Button type="button" variant="secondary" icon="folder" disabled={picking === 'ffmpegPath'} onClick={() => void chooseFile('ffmpegPath', tr('Vali ffmpeg.exe'), ['.exe'])}>{picking === 'ffmpegPath' ? tr('Avan…') : tr('Sirvi')}</Button>
                            </div>
                            <small>{tr('Vaikimisi:')} <code>tools\ffmpeg\bin\ffmpeg.exe</code>. {tr("Välise build'i kasutamiseks vali selle")} <code>ffmpeg.exe</code>.</small>
                          </label>
                          <label className="settings-field" htmlFor="system-ffprobe-path">
                            <span>{tr('FFprobe käivitusfail')}</span>
                            <div className="path-control">
                              <input id="system-ffprobe-path" aria-label={tr('FFprobe käivitusfail')} value={form.ffprobePath} onChange={(event) => change('ffprobePath', event.target.value)} placeholder={'C:\\ffmpeg\\bin\\ffprobe.exe'} autoComplete="off" data-focusable="true" />
                              <Button type="button" variant="secondary" icon="folder" disabled={picking === 'ffprobePath'} onClick={() => void chooseFile('ffprobePath', tr('Vali ffprobe.exe'), ['.exe'])}>{picking === 'ffprobePath' ? tr('Avan…') : tr('Sirvi')}</Button>
                            </div>
                            <small>{tr('Vaikimisi:')} <code>tools\ffmpeg\bin\ffprobe.exe</code>. {tr("Hoia FFmpeg ja FFprobe samast build'ist.")}</small>
                          </label>
                          </Disclosure>
                        </section>
            </div>
          ) : null}
          <div className="security-note">
                  <Icon name="info" size={19} />
                  <div><strong>{tr('Praegune versioon on mõeldud usaldatud koduvõrku.')}</strong><p>{tr('Kasutajakontosid ja autentimist veel pole. Ära suuna porti 8096 internetti ega paiguta serverit avaliku pöördproksi taha.')}</p></div>
                </div>
        </div>
      </section>

      {!loading ? (
        <>
          <SettingsGroup
            id="settings-metadata"
            title={tr('Meediakogud ja metaandmed')}
            description={tr('Lisa kaustad ning määra filmide, seriaalide ja muusika usaldusväärsed metaandmete allikad.')}
            icon="folder"
            scope={tr('Serveriülene')}
            behavior={tr('Eelistused vajavad salvestamist. Meediakogude toimingud rakenduvad kohe.')}
          >
            <div className="settings-subsection-list">
              <LibrarySettings libraries={libraries} onChanged={onLibrariesChanged} />
              <section id="settings-metadata-provider" className="settings-form-section settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="sparkles" size={20} /></span>
                            <span><strong>{tr('Filmide ja seriaalide metaandmed')}</strong><small>{tr('TMDb nimed, kirjeldused ja artwork')}</small></span>
                          </header>
                          <div className="instruction-card">
                            <Icon name="info" size={20} />
                            <div>
                              <strong>{tr('TMDb seadistamine')}</strong>
                              <ol>
                                <li>{tr('Loo TMDb konto ja ava konto seadetes API.')}</li>
                                <li>{tr('Taotle API ligipääs ja kopeeri pikk API Read Access Token ehk Bearer token, mitte lühike v3 API Key.')}</li>
                                <li>{tr('Kleebi token, salvesta ning KoduMedia täidab puuduvad nimed, kirjeldused ja pildid taustal automaatselt.')}</li>
                              </ol>
                              <a href="https://www.themoviedb.org/settings/api" target="_blank" rel="noreferrer">{tr('Ava TMDb API seaded')}</a>
                            </div>
                          </div>
                          <label className="settings-field" htmlFor="system-tmdb-token">
                            <span>TMDb API Read Access Token</span>
                            <input
                              id="system-tmdb-token"
                              aria-label="TMDb API Read Access Token"
                              type="password"
                              value={form.tmdbToken}
                              onChange={(event) => change('tmdbToken', event.target.value)}
                              placeholder={configuration?.settings.metadata.tmdbTokenConfigured ? tr('Salvestatud {{hint}}. Uue tokeni jaoks kirjuta siia', { hint: configuration.settings.metadata.tmdbTokenHint || '' }) : tr('Kleebi API Read Access Token')}
                              autoComplete="new-password"
                              data-focusable="true"
                            />
                            <small>{tr('Salvestatud tokenit ei saadeta brauserisse tagasi. Tühi väli jätab olemasoleva tokeni alles.')}</small>
                          </label>
                          {configuration?.settings.metadata.tmdbTokenConfigured ? (
                            <Switch
                              danger
                              checked={form.clearTmdbToken}
                              onChange={(event) => change('clearTmdbToken', event.target.checked)}
                              label={tr('Eemalda salvestamisel olemasolev TMDb token')}
                            />
                          ) : null}
                          <label className="settings-field settings-field--short" htmlFor="system-metadata-language">
                            <span>{tr('Metaandmete keel')}</span>
                            <input id="system-metadata-language" aria-label={tr('Metaandmete keel')} value={form.language} onChange={(event) => change('language', event.target.value)} placeholder="en-US" data-focusable="true" />
                            <small>{tr('Kasuta kuju')} <code>en-US</code>, <code>et-EE</code> {tr('või')} <code>en</code>. {tr('Keele muutmisel värskendatakse olemasolevad kirjeldused taustal.')}</small>
                          </label>
                          <p className="field-help"><strong>{tr('Automaatne töö:')}</strong> {tr('Eraldi „Laadi metaandmed” nuppu failide juures enam pole. Skannimine ja seadete salvestamine käivitavad puuduva info laadimise serveris.')}</p>
                          <p className="tmdb-attribution">This product uses the TMDB API but is not endorsed or certified by TMDB.</p>
                        </section>
              <section id="settings-imdb" className="settings-form-section settings-form-section--compact settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="star" size={20} /></span>
                            <span><strong>{tr('IMDb reitingud')}</strong><small>{tr('Kohalikult vahemällu salvestatud hinnangud')}</small></span>
                          </header>
                          <p className="field-help">
                            {tr('Vaikimisi kasutatakse IMDb ametlikku aadressi. Server teeb käivitamisel tingimusliku värskenduse ja hoiab pakitud faili kohalikus vahemälus; brauser ei laadi ega töötle suurt andmestikku.')}
                            {' '}<a href="https://developer.imdb.com/non-commercial-datasets/" target="_blank" rel="noreferrer">{tr('Ava IMDb andmestike juhend')}</a>.
                          </p>
                          <label className="settings-field" htmlFor="system-imdb-ratings-source">
                            <span>{tr('title.ratings.tsv.gz URL või kohalik fail')}</span>
                            <div className="path-control">
                              <input id="system-imdb-ratings-source" aria-label={tr('IMDb reitinguandmestiku URL või fail')} value={form.imdbRatingsSource} onChange={(event) => change('imdbRatingsSource', event.target.value)} placeholder={defaultIMDbSource} autoComplete="off" data-focusable="true" />
                              <Button type="button" variant="secondary" icon="folder" disabled={picking === 'imdbRatingsSource'} onClick={() => void chooseFile('imdbRatingsSource', tr('Vali IMDb reitinguandmestik'), ['.tsv', '.gz'])}>{picking === 'imdbRatingsSource' ? tr('Avan…') : tr('Kohalik fail')}</Button>
                            </div>
                            <small>{tr('Soovituslik väärtus on')} <code>{defaultIMDbSource}</code>. {tr('Kohalik .tsv või .tsv.gz sobib võrguühenduseta kasutuseks.')}</small>
                          </label>

                          {imdbJob && isActive(imdbJob) ? (
                            <div className="scan-status scan-status--running" aria-live="polite">
                              <ProgressBar value={Math.max(3, imdbJob.progress)} label={tr('IMDb värskendamise edenemine')} />
                              <span><Icon name="refresh" size={15} />{trExternal(imdbJob.message) || tr('Värskendan IMDb andmestikku…')}</span>
                            </div>
                          ) : null}
                          <div className="system-settings-inline-action">
                            <div><strong>{tr('Värskenda IMDb andmestikku')}</strong><small>{tr('Kontrollib URL-i, laadib faili ainult muutuse korral ja uuendab olemasolevad skoorid.')}</small></div>
                            <Button type="button" variant="secondary" icon="refresh" disabled={Boolean(busy) || updateBusy} onClick={() => void refreshIMDb()}>{busy === 'imdb' ? tr('Värskendan…') : tr('Värskenda nüüd')}</Button>
                          </div>
                        </section>
              <section id="settings-music-metadata" className="settings-form-section settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="music" size={20} /></span>
                            <span><strong>{tr('Muusika metaandmed')}</strong><small>{tr('Mitme allika vaste, lokaalne stiilianalüüs ja artwork')}</small></span>
                          </header>
                          <div className="instruction-card">
                            <Icon name="info" size={20} />
                            <div>
                              <strong>{tr('Maksimaalne katvus ilma püsitasuta')}</strong>
                              <p>{tr('KoduMedia proovib kõigepealt tasuta Shazami audio-tuvastust ning kasutab seejärel MusicBrainzi, Cover Art Archive’i, AcoustID-d ja Deezerit. Leitud info, fingerprint, analüüsi tulemus ja artwork salvestatakse kohalikult, et sama lugu ei peaks korduvalt töötlema.')}</p>
                              <a href="https://acoustid.org/new-application" target="_blank" rel="noreferrer">{tr('Loo tasuta AcoustID rakenduse võti')}</a>
                            </div>
                          </div>
                          <Switch
                            checked={form.musicMetadataEnabled}
                            onChange={(event) => change('musicMetadataEnabled', event.target.checked)}
                            label={tr('Otsi puuduvaid muusika metaandmeid ja artworki')}
                          />
                          <Switch
                            checked={form.shazamEnabled}
                            onChange={(event) => change('shazamEnabled', event.target.checked)}
                            disabled={!form.musicMetadataEnabled}
                            label={tr('Kasuta esimese valikuna tasuta Shazami audio-tuvastust')}
                          />
                          <Switch
                            checked={form.fingerprintEnabled}
                            onChange={(event) => change('fingerprintEnabled', event.target.checked)}
                            disabled={!form.musicMetadataEnabled}
                            label={tr('Kasuta Shazami järel AcoustID audio fingerprinti, kui tasuta API võti on lisatud')}
                          />
                          <Switch
                            checked={form.catalogFallbackEnabled}
                            onChange={(event) => change('catalogFallbackEnabled', event.target.checked)}
                            disabled={!form.musicMetadataEnabled}
                            label={tr('Kasuta MusicBrainzi järel laiemat kataloogivastet stiili ja artworki leidmiseks')}
                          />
                          <Switch
                            checked={form.audioAnalysisEnabled}
                            onChange={(event) => change('audioAnalysisEnabled', event.target.checked)}
                            disabled={!form.musicMetadataEnabled}
                            label={tr('Määra puuduva stiili jaoks helist lokaalne hinnang')}
                          />
                          <Switch
                            checked={form.generatedArtworkEnabled}
                            onChange={(event) => change('generatedArtworkEnabled', event.target.checked)}
                            disabled={!form.musicMetadataEnabled}
                            label={tr('Näita puuduva ametliku pildi asemel neutraalset loo fallback-kaarti')}
                          />
                          <label className="settings-field" htmlFor="system-acoustid-api-key">
                            <span>AcoustID API key <small>({tr('valikuline')})</small></span>
                            <input
                              id="system-acoustid-api-key"
                              type="password"
                              value={form.acoustIdApiKey}
                              onChange={(event) => change('acoustIdApiKey', event.target.value)}
                              placeholder={configuration?.settings.music?.acoustIdApiKeyConfigured ? tr('Salvestatud {{hint}}. Uue võtme jaoks kirjuta siia', { hint: configuration.settings.music.acoustIdApiKeyHint || '' }) : tr('Kleebi tasuta AcoustID API võti')}
                              autoComplete="new-password"
                              data-focusable="true"
                            />
                            <small>{tr('Ilma võtmeta töötab MusicBrainzi pealkirja, esitaja, albumi ja kestuse põhine vaste. Võti lisab helipõhise tuvastuse; fpcalc laaditakse Windowsis alla ainult vajadusel ja kontrollitakse SHA-256 summaga.')}</small>
                          </label>
                          {configuration?.settings.music?.acoustIdApiKeyConfigured ? (
                            <Switch
                              danger
                              checked={form.clearAcoustIdApiKey}
                              onChange={(event) => change('clearAcoustIdApiKey', event.target.checked)}
                              label={tr('Eemalda salvestamisel AcoustID API võti')}
                            />
                          ) : null}
                          <p className="field-help">{tr('Shazami vaste on audio fingerprinti põhine ja asendab madala kindlusega automaatse audioanalüüsi tulemuse. Käsitsi määratud sidecar ja helifaili tagid jäävad alles. Shazami liides on mitteametlik ning võib teenuse poolel muutuda; tõrke korral jätkab KoduMedia automaatselt järgmiste tasuta allikatega.')}</p>
                          {musicJob && isActive(musicJob) ? (
                            <div className="scan-status scan-status--running" aria-live="polite">
                              <ProgressBar value={Math.max(3, musicJob.progress)} label={tr('Muusika rikastamise edenemine')} />
                              <span><Icon name="refresh" size={15} />{trExternal(musicJob.message) || tr('Töötlen muusikateeki…')}</span>
                            </div>
                          ) : null}
                          <div className="system-settings-inline-action">
                            <div>
                              <strong>{tr('Töötle kogu muusikateek uuesti')}</strong>
                              <small>{tr('Kontrollib kõik lood esmalt Shazami audio-tuvastusega ning seejärel ülejäänud kataloogi-, stiili- ja artworki fallback-ahelaga. Olemasolevad käsitsi määratud väärtused jäävad alles.')}</small>
                            </div>
                            <Button
                              type="button"
                              variant="secondary"
                              icon="refresh"
                              disabled={Boolean(busy) || updateBusy || Boolean(musicJob && isActive(musicJob))}
                              onClick={() => void enrichMusic()}
                            >
                              {busy === 'music' ? tr('Töötlen…') : tr('Töötle nüüd')}
                            </Button>
                          </div>
                        </section>
            </div>
          </SettingsGroup>

          <SettingsGroup
            id="settings-quality"
            title={tr('Taasesitus ja heli')}
            description={tr('Määra videokvaliteet, transkodeerimise vahemälu ja seriaalide tunnusmuusika käitumine.')}
            icon="play"
            scope={tr('Serveriülene')}
            behavior={tr('Muudatused vajavad salvestamist.')}
          >
            <div className="settings-subsection-list">
              <section id="settings-video-quality" className="settings-form-section settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="sparkles" size={20} /></span>
                            <span><strong>{tr('Pildikvaliteet')}</strong><small>{tr('Originaalvideo, kvaliteetne transkodeerimine ja vajaduspõhine cache')}</small></span>
                          </header>
                          <div className="instruction-card">
                            <Icon name="info" size={20} />
                            <div>
                              <strong>{tr('Originaalkvaliteet on alati esimene valik')}</strong>
                              <p>{tr("KoduMedia proovib esmalt Direct Play'd ja videoraja muutmata remux'i. Optimeeritud segmendid luuakse ainult siis, kui videot tuleb päriselt transkodeerida; kogu meediakogu öist eelkodeerimist ei tehta.")}</p>
                            </div>
                          </div>
                          <div className="settings-field-row">
                            <label className="settings-field" htmlFor="system-quality-profile">
                              <span>{tr('Vaikimisi kvaliteediprofiil')}</span>
                              <SelectControl id="system-quality-profile" value={form.qualityProfile} onChange={(event) => change('qualityProfile', event.target.value as FormState['qualityProfile'])}>
                                <option value="original">{tr('Originaal - säilita lähteresolutsioon')}</option>
                                <option value="maximum">{tr('Maksimaalne - parim võimalik transkodeering')}</option>
                                <option value="balanced">{tr('Tasakaalus - kuni 1080p')}</option>
                                <option value="data-saver">{tr('Andmesäästlik - kuni 720p')}</option>
                              </SelectControl>
                              <small>{tr('Originaal ja Maksimaalne ei vähenda 4K videot automaatselt 1080p peale.')}</small>
                            </label>
                            <label className="settings-field" htmlFor="system-quality-codec">
                              <span>{tr('Eelistatud väljundcodec')}</span>
                              <SelectControl id="system-quality-codec" value={form.preferredVideoCodec} onChange={(event) => change('preferredVideoCodec', event.target.value as FormState['preferredVideoCodec'])}>
                                <option value="auto">{tr('Automaatne')}</option>
                                <option value="h264">{tr('H.264 - kõige laiem ühilduvus')}</option>
                                <option value="hevc">{tr('HEVC / H.265 - 10-bit ja HDR')}</option>
                                <option value="av1">{tr('AV1 - sobiva riistvara korral')}</option>
                              </SelectControl>
                              <small>{tr('AV1 valitakse automaatselt ainult siis, kui serveril on sobiv riistvaraline encoder ja klient suudab seda sujuvalt esitada.')}</small>
                            </label>
                          </div>
                          <Disclosure
                            title={tr('Täpsemad taasesituse seaded')}
                            description={tr('Encoder, vahemälu ja VMAF analüüs. Vaikeseaded sobivad enamikele serveritele.')}
                            icon="sliders"
                          >
                          <div className="settings-field-row">
                            <label className="settings-field" htmlFor="system-quality-hardware">
                              <span>{tr("Encoder'i eelistus")}</span>
                              <SelectControl id="system-quality-hardware" value={form.hardwarePreference} onChange={(event) => change('hardwarePreference', event.target.value as FormState['hardwarePreference'])}>
                                <option value="auto">{tr('Automaatne')}</option>
                                <option value="software">{tr('Tarkvaraline')}</option>
                                <option value="qsv">Intel Quick Sync</option>
                                <option value="nvenc">NVIDIA NVENC</option>
                                <option value="amf">AMD AMF</option>
                              </SelectControl>
                              <small>{tr("Kui valitud hardware encoder ei käivitu või ei püsi reaalajas, kasutab KoduMedia kontrollitud fallback'i.")}</small>
                            </label>
                            <label className="settings-field" htmlFor="system-quality-cache-policy">
                              <span>{tr('Optimeeritud segmentide säilitamine')}</span>
                              <SelectControl id="system-quality-cache-policy" value={form.cachePolicy} onChange={(event) => change('cachePolicy', event.target.value as FormState['cachePolicy'])}>
                                <option value="off">{tr('Ära säilita')}</option>
                                <option value="watched">{tr('Säilita vaatamise ajal loodud segmendid')}</option>
                                <option value="complete">{tr('Säilita ainult lõpuni valminud vood')}</option>
                                <option value="finish">{tr('Lõpeta juba alustatud optimeeritud voog')}</option>
                              </SelectControl>
                              <small>{tr("Cache tekib ainult päris playback'i käigus; eraldi taustal kogu meediakogu ei transkodeerita.")}</small>
                            </label>
                          </div>
                          <div className="settings-field-row">
                            <label className="settings-field settings-field--short" htmlFor="system-quality-cache-reserve-gb">
                              <span>{tr('Vaba ruumi reserv (GB)')}</span>
                              <input id="system-quality-cache-reserve-gb" type="number" min={1} max={1024} step={1} value={form.cacheFreeReserveGb} onChange={(event) => change('cacheFreeReserveGb', Number(event.target.value))} data-focusable="true" />
                              <small>{tr('KoduMedia eemaldab vana vahemälu enne, kui kettale jääb sellest vähem vaba ruumi.')}</small>
                            </label>
                            <label className="settings-field settings-field--short" htmlFor="system-quality-cache-reserve-percent">
                              <span>{tr('Vaba ruumi reserv (%)')}</span>
                              <input id="system-quality-cache-reserve-percent" type="number" min={1} max={50} step={1} value={form.cacheFreeReservePercent} onChange={(event) => change('cacheFreeReservePercent', Number(event.target.value))} data-focusable="true" />
                              <small>{tr('GB ja protsendi väärtusest kasutatakse alati rangemat piiri.')}</small>
                            </label>
                          </div>
                          <div className="settings-field-row">
                            <label className="settings-field settings-field--short" htmlFor="system-quality-cache-size">
                              <span>{tr("Cache'i maksimaalne maht (GB)")}</span>
                              <input id="system-quality-cache-size" type="number" min={1} max={4096} step={1} value={form.cacheMaxGb} onChange={(event) => change('cacheMaxGb', Number(event.target.value))} data-focusable="true" />
                              <small>{tr('Piiri ületamisel eemaldatakse vanimad mitteaktiivsed optimeeritud vood.')}</small>
                            </label>
                            <label className="settings-field settings-field--short" htmlFor="system-quality-grace">
                              <span>{tr('Sulgemisjärgne grace-periood (s)')}</span>
                              <input id="system-quality-grace" type="number" min={0} max={300} step={1} value={form.gracePeriodSeconds} onChange={(event) => change('gracePeriodSeconds', Number(event.target.value))} data-focusable="true" />
                              <small>{tr('Kui klient sulgeb playeri, võib sama encoder lühidalt oodata võimalikku taasühendust.')}</small>
                            </label>
                          </div>
                          <Switch
                            checked={form.vmafEnabled}
                            onChange={(event) => change('vmafEnabled', event.target.checked)}
                            label={tr('Kasuta vajaduspõhist VMAF kvaliteedianalüüsi järgmiste esituste profiili täpsustamiseks')}
                          />
                          {form.vmafEnabled ? (
                            <div className="settings-field-row">
                              <label className="settings-field settings-field--short" htmlFor="system-quality-vmaf-target">
                                <span>{tr('VMAF sihttase')}</span>
                                <input id="system-quality-vmaf-target" type="number" min={80} max={99} step={1} value={form.vmafTarget} onChange={(event) => change('vmafTarget', Number(event.target.value))} data-focusable="true" />
                                <small>{tr('Kõrgem väärtus eelistab rohkem detaile ja suuremat bitikiirust.')}</small>
                              </label>
                              <label className="settings-field settings-field--short" htmlFor="system-quality-vmaf-sample">
                                <span>{tr('Prooviala pikkus (s)')}</span>
                                <input id="system-quality-vmaf-sample" type="number" min={4} max={60} step={1} value={form.vmafSampleSeconds} onChange={(event) => change('vmafSampleSeconds', Number(event.target.value))} data-focusable="true" />
                                <small>{tr("Analüüs töötab ainult juba valminud SDR-segmentidel ega blokeeri playback'i.")}</small>
                              </label>
                            </div>
                          ) : null}
                          </Disclosure>
                        </section>
              <section id="settings-theme-music" className="settings-form-section settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="music" size={20} /></span>
                            <span><strong>{tr('Seriaalide tunnusmuusika')}</strong><small>{tr('ThemerrDB-st hangitud ja serverisse vahemällu salvestatud tunnusmeloodiad')}</small></span>
                          </header>
                          <Field
                            id="system-series-theme-playback"
                            className="settings-field"
                            label={tr('Esita seriaalide tunnusmuusikat')}
                            help={tr('Tunnusmuusika hangitakse ThemerrDB viite kaudu üks kord, salvestatakse serveri vahemällu ja esitatakse 15% helitugevusega. Sisse- ja väljahajumine kestavad 2 sekundit.')}
                          >
                            <SelectControl
                              id="system-series-theme-playback"
                              value={form.seriesThemePlayback}
                              onChange={(event) => change('seriesThemePlayback', event.target.value as FormState['seriesThemePlayback'])}
                            >
                              <option value="always">{tr('Alati')}</option>
                              <option value="first-open">{tr('Ainult esimesel avamisel')}</option>
                              <option value="off">{tr('Tunnusmuusika väljas')}</option>
                            </SelectControl>
                          </Field>
                          <p className="field-help"><strong>{tr('Esitamise käitumine:')}</strong> {tr('Heli jätkub sama seriaali üldvaate, hooaegade ja episoodide vahel. Episoodi käivitamisel või seriaali vaadetest lahkumisel hajub heli välja ning automaatselt ei kordu.')}</p>
                          <p className="field-help"><strong>{tr('Allikas:')}</strong> {tr('KoduMedia kasutab ainult ThemerrDB andmebaasis määratud YouTube’i tunnusmeloodiat. Seriaalil peab olema TMDb ID ning serveril internetiühendus esmakordseks laadimiseks.')}</p>
                        </section>
            </div>
          </SettingsGroup>

          <SettingsGroup
            id="settings-storage"
            title={tr('Kettakasutus')}
            description={tr('Vaata vahemälude, uuenduste ja logide tegelikku mahtu ning vabasta turvaliselt ruumi.')}
            icon="server"
            scope={tr('Serveriarvutis')}
            behavior={tr('Hooldustoimingud rakenduvad kohe.')}
          >
            <StorageSettings />
          </SettingsGroup>

          <SettingsGroup
            id="settings-subtitles"
            title={tr('Subtiitrid')}
            description={tr('Vali eelistatud keeled ning seadista OpenSubtitles otsing ja allalaadimine.')}
            icon="captions"
            scope={tr('Serveriülene')}
            behavior={tr('Muudatused vajavad salvestamist.')}
          >
            <div className="settings-subsection-list">
              <section id="settings-subtitles-provider" className="settings-form-section settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="captions" size={20} /></span>
                            <span><strong>{tr('Subtiitrid ja OpenSubtitles')}</strong><small>{tr('Keeleeelistused ja allalaadimine')}</small></span>
                          </header>
                          <p className="field-help">
                            {tr('KoduMedia leiab automaatselt meediafaili kõrval olevad ning faili sisse pakitud subtiitrid. Tekstisubtiitrid teisendatakse serveris UTF-8 WebVTT vormingusse; pildisubtiitrid põletatakse vajaduse korral transkodeeritud videopilti.')}
                          </p>
                          <label className="settings-field settings-field--short" htmlFor="system-subtitle-languages">
                            <span>{tr('Eelistatud subtiitrite keeled')}</span>
                            <input
                              id="system-subtitle-languages"
                              aria-label={tr('Eelistatud subtiitrite keeled')}
                              value={form.preferredSubtitleLanguages}
                              onChange={(event) => change('preferredSubtitleLanguages', event.target.value)}
                              placeholder="et, en, fi"
                              autoComplete="off"
                              data-focusable="true"
                            />
                            <small>{tr('Sisesta ISO keelekoodid komaga eraldatult, näiteks')} <code>en, et, fi</code>. {tr('Järjekord määrab eelistuse.')}</small>
                          </label>
                          <div className="instruction-card">
                            <Icon name="captions" size={20} />
                            <div>
                              <strong>{tr('OpenSubtitles.com seadistamine')}</strong>
                              <ol>
                                <li>{tr('Loo OpenSubtitles.com konto ja Consumer API võti.')}</li>
                                <li>{tr('Sisesta API võti ning konto kasutajanimi ja parool.')}</li>
                                <li>{tr('Playeri subtiitrimenüüst saad seejärel vasteid otsida ja serverisse alla laadida.')}</li>
                              </ol>
                              <a href="https://www.opensubtitles.com/consumers" target="_blank" rel="noreferrer">{tr('Ava OpenSubtitles API tarbijate leht')}</a>
                            </div>
                          </div>
                          <label className="settings-field" htmlFor="system-opensubtitles-api-key">
                            <span>{tr('OpenSubtitles Consumer API võti')}</span>
                            <input
                              id="system-opensubtitles-api-key"
                              aria-label={tr('OpenSubtitles Consumer API võti')}
                              type="password"
                              value={form.openSubtitlesApiKey}
                              onChange={(event) => change('openSubtitlesApiKey', event.target.value)}
                              placeholder={configuration?.settings.subtitles?.openSubtitlesApiKeyConfigured ? tr('Salvestatud {{hint}}. Uue võtme jaoks kirjuta siia', { hint: configuration.settings.subtitles.openSubtitlesApiKeyHint || '' }) : tr('Kleebi Consumer API võti')}
                              autoComplete="new-password"
                              data-focusable="true"
                            />
                            <small>{tr('Tühi väli jätab olemasoleva võtme alles. Salvestatud saladust ei saadeta brauserisse tagasi.')}</small>
                          </label>
                          {configuration?.settings.subtitles?.openSubtitlesApiKeyConfigured ? (
                            <Switch
                              danger
                              checked={form.clearOpenSubtitlesApiKey}
                              onChange={(event) => change('clearOpenSubtitlesApiKey', event.target.checked)}
                              label={tr('Eemalda salvestamisel OpenSubtitles API võti')}
                            />
                          ) : null}
                          <div className="settings-field-row">
                            <label className="settings-field" htmlFor="system-opensubtitles-username">
                              <span>{tr('OpenSubtitles kasutajanimi')}</span>
                              <input
                                id="system-opensubtitles-username"
                                aria-label={tr('OpenSubtitles kasutajanimi')}
                                value={form.openSubtitlesUsername}
                                onChange={(event) => change('openSubtitlesUsername', event.target.value)}
                                placeholder={configuration?.settings.subtitles?.openSubtitlesUsernameConfigured ? tr('Salvestatud {{hint}}', { hint: configuration.settings.subtitles.openSubtitlesUsernameHint || '' }) : tr('Kasutajanimi')}
                                autoComplete="username"
                                data-focusable="true"
                              />
                            </label>
                            <label className="settings-field" htmlFor="system-opensubtitles-password">
                              <span>{tr('OpenSubtitles parool')}</span>
                              <input
                                id="system-opensubtitles-password"
                                aria-label={tr('OpenSubtitles parool')}
                                type="password"
                                value={form.openSubtitlesPassword}
                                onChange={(event) => change('openSubtitlesPassword', event.target.value)}
                                placeholder={configuration?.settings.subtitles?.openSubtitlesPasswordConfigured ? tr('Parool on salvestatud. Uue parooli jaoks kirjuta siia') : tr('Parool')}
                                autoComplete="new-password"
                                data-focusable="true"
                              />
                            </label>
                          </div>
                          {configuration?.settings.subtitles?.openSubtitlesUsernameConfigured || configuration?.settings.subtitles?.openSubtitlesPasswordConfigured ? (
                            <div className="check-field-group">
                              {configuration.settings.subtitles.openSubtitlesUsernameConfigured ? (
                                <Switch
                                  danger
                                  checked={form.clearOpenSubtitlesUsername}
                                  onChange={(event) => change('clearOpenSubtitlesUsername', event.target.checked)}
                                  label={tr('Eemalda salvestamisel kasutajanimi')}
                                />
                              ) : null}
                              {configuration.settings.subtitles.openSubtitlesPasswordConfigured ? (
                                <Switch
                                  danger
                                  checked={form.clearOpenSubtitlesPassword}
                                  onChange={(event) => change('clearOpenSubtitlesPassword', event.target.checked)}
                                  label={tr('Eemalda salvestamisel parool')}
                                />
                              ) : null}
                            </div>
                          ) : null}
                          <p className="field-help"><strong>{tr('Privaatsus:')}</strong> {tr('Otsing ja allalaadimine saadavad OpenSubtitles teenusele filmi või episoodi otsinguandmed ning võimaluse korral OpenSubtitles failiräsi. Kohalikke faile tervikuna teenusele ei laadita.')}</p>
                        </section>
            </div>
          </SettingsGroup>

          <SettingsGroup
            id="settings-language"
            title={tr('Välimus ja keel')}
            description={tr('Muuda selles brauseris kasutatavat keelt ja KoduMedia värviteemat.')}
            icon="sun"
            scope={tr('Ainult selles brauseris')}
            behavior={tr('Rakendub kohe.')}
          >
            <div className="settings-subsection-list">
              <LanguageSettings />
              <ThemeSettings />
            </div>
          </SettingsGroup>

          <SettingsGroup
            id="settings-update"
            title={tr('Uuendused ja diagnostika')}
            description={tr('Paigalda kontrollitud Windowsi väljalase ning vaata uuendamise tehnilist olekut.')}
            icon="download"
            scope={tr('Serveriarvutis')}
            behavior={tr('Uuenduse paigaldamine käivitub kohe.')}
          >
            <div className="settings-subsection-list">
              <section id="settings-update-release" className="settings-form-section settings-form-section--update settings-subsection">
                          <header className="settings-form-section__heading">
                            <span className="settings-form-section__icon"><Icon name="download" size={20} /></span>
                            <span><strong>{tr('KoduMedia uuendamine')}</strong><small>{tr('Kontrollitud Windows x64 väljalasked')}</small></span>
                          </header>
                          <p className="field-help">
                            {tr('Vali ametlik Windows x64 KoduMedia väljalaske ZIP. Server kontrollib manifesti, kõigi failide SHA-256 väärtusi ja versiooni, teeb rakendusfailidest varukoopia ning taastab eelmise versiooni, kui uus server ei käivitu.')} <code>data</code> {tr('kausta ja')} <code>.env</code> {tr('faili ei muudeta.')}
                          </p>
                          <div className={`software-update-card software-update-card--${softwareUpdate?.state || 'idle'}`}>
                            <span className="software-update-card__icon"><Icon name={softwareUpdate?.state === 'failed' ? 'alert' : softwareUpdate?.state === 'succeeded' ? 'check' : 'refresh'} size={20} /></span>
                            <div className="software-update-card__copy">
                              <strong>{updateStateLabel(softwareUpdate)}</strong>
                              <small>{tr('Praegune versioon:')} {softwareUpdate?.currentVersion || configuration?.version || '-'}{softwareUpdate?.targetVersion ? ` · ${tr('Sihtversioon:')} ${softwareUpdate.targetVersion}` : ''}</small>
                              {softwareUpdate?.message ? <p>{trExternal(softwareUpdate.message)}</p> : null}
                              {softwareUpdate?.error ? <p className="software-update-card__error">{trExternal(softwareUpdate.error)}</p> : null}
                              {softwareUpdate?.rollbackApplied ? <small>{tr('Eelmine versioon taastati automaatselt.')}</small> : null}
                            </div>
                            {localUpdateClient ? (
                              <>
                                <input
                                  ref={updateInput}
                                  className="software-update-input"
                                  type="file"
                                  accept=".zip,application/zip"
                                  aria-label={tr('KoduMedia uuenduspakett')}
                                  onChange={(event) => {
                                    const file = event.target.files?.[0]
                                    if (file) void installRelease(file)
                                  }}
                                />
                                <Button type="button" variant="secondary" icon="folder" disabled={updateBusy || Boolean(busy)} onClick={() => updateInput.current?.click()}>
                                  {updateBusy ? tr('Uuendan…') : tr('Vali uuenduse ZIP')}
                                </Button>
                              </>
                            ) : (
                              <span className="status-pill status-pill--optional"><Icon name="info" size={15} />{tr('Ainult localhost')}</span>
                            )}
                          </div>
                          <p className="field-help"><strong>{tr('Turvapiirang:')}</strong> {tr('Uuendamist saab käivitada ainult serveriarvuti brauserist aadressil')} <code>http://localhost:8096</code>. {tr('Esimene üleminek versioonilt 0.2.1 versioonile 0.2.2 tuleb teha käsitsi; alates 0.2.2-st saab järgmised väljalasked paigaldada siit.')}</p>
                        </section>
            </div>
          </SettingsGroup>

          <SettingsSaveBar
            dirty={dirty}
            busy={Boolean(busy) || updateBusy}
            saving={busy === 'save'}
            testing={busy === 'test'}
            onReset={resetChanges}
            onTest={() => void run('test')}
            onSave={() => void run('save')}
          />
        </>
      ) : null}

      {picker ? (
              <FileBrowserDialog
                title={picker.title}
                description={tr('Windows Explorerit saab avada ainult serveriarvuti localhost-brauserist. See on kerge serveripoolne varuvalija.')}
                mode="file"
                initialPath={containingDirectory(form[picker.field])}
                extensions={picker.extensions}
                onClose={() => setPicker(null)}
                onSelect={(path) => {
                  change(picker.field, path)
                  setPicker(null)
                }}
              />
            ) : null}
    </div>
  )
}
