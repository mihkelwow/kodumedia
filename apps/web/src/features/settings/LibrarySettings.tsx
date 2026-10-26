import { useCallback, useState, type FormEvent } from 'react'
import type { Job, Library, LibraryKind } from '../../api/types'
import { api } from '../../api/client'
import { Button } from '../../shared/components/Button'
import { FileBrowserDialog } from '../../shared/components/FileBrowserDialog'
import { Icon, type IconName } from '../../shared/components/Icon'
import { SelectControl } from '../../shared/components/SelectControl'
import { ProgressBar } from '../../shared/components/ProgressBar'
import { openNativeFilePicker } from '../../shared/utils/filePicker'
import { useLibraryScan } from './useLibraryScan'
import { tr, trCount, trExternal, type TranslationKey } from '../../i18n/i18n'

const kindOptions: Array<{ value: LibraryKind; label: TranslationKey }> = [
  { value: 'movies', label: 'Filmid' },
  { value: 'series', label: 'Seriaalid' },
  { value: 'music', label: 'Muusika' },
  { value: 'mixed', label: 'Segakogu' },
]

function kindIcon(kind: LibraryKind): IconName {
  if (kind === 'movies') return 'film'
  if (kind === 'series') return 'tv'
  if (kind === 'music') return 'music'
  return 'folder'
}

function JobProgress({ job }: { job?: Job }) {
  if (!job) return null
  const active = job.status === 'queued' || job.status === 'running'
  return (
    <div className={`scan-status scan-status--${job.status}`} aria-live="polite">
      <ProgressBar value={Math.max(3, job.progress)} label={tr('Skannimise edenemine')} />
      <span>
        <Icon name={job.status === 'failed' ? 'alert' : job.status === 'succeeded' ? 'check' : 'refresh'} size={15} />
        {job.status === 'failed' ? trExternal(job.error) || tr('Skannimine ebaõnnestus') : trExternal(job.message) || (active ? tr('Skannin…') : tr('Valmis'))}
      </span>
    </div>
  )
}

export function LibrarySettings({ libraries, onChanged }: { libraries: Library[]; onChanged: () => Promise<void> }) {
  const [name, setName] = useState('')
  const [kind, setKind] = useState<LibraryKind>('mixed')
  const [busyKey, setBusyKey] = useState('')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [pickerLibrary, setPickerLibrary] = useState<Library | null>(null)
  const [pickerBusy, setPickerBusy] = useState('')
  const onScanCompleted = useCallback(() => onChanged(), [onChanged])
  const { jobs, startScan } = useLibraryScan(onScanCompleted)

  const runMutation = async (key: string, action: () => Promise<unknown>, success: string) => {
    setBusyKey(key)
    setError('')
    setMessage('')
    try {
      await action()
      await onChanged()
      setMessage(success)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Toiming ebaõnnestus.'))
      throw caught
    } finally {
      setBusyKey('')
    }
  }

  const createLibrary = async (event: FormEvent) => {
    event.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    try {
      await runMutation('create', () => api.libraries.create(trimmed, kind), tr('Kogu „{{name}}” loodi.', { name: trimmed }))
      setName('')
      setKind('mixed')
    } catch {
      // The shared mutation handler already rendered the error.
    }
  }

  const addSource = async (library: Library, path: string) => {
    setPickerLibrary(null)
    try {
      await runMutation(`source:${library.id}`, () => api.libraries.addSource(library.id, path), tr('Kaust lisati kogule „{{name}}”. Skannimine käivitub kohe.', { name: library.name }))
      await startScan(library)
    } catch {
      // The shared mutation handler already rendered the error.
    }
  }


  const browseSource = async (library: Library) => {
    setPickerBusy(library.id)
    setError('')
    try {
      const result = await openNativeFilePicker({
        mode: 'directory',
        title: tr('Vali kaust kogule „{{name}}”', { name: library.name }),
        initialPath: library.sources[0]?.path || '',
      })
      if (result.kind === 'selected') {
        await addSource(library, result.path)
      } else if (result.kind === 'unavailable') {
        setPickerLibrary(library)
      }
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Windowsi kaustavalijat ei saanud avada.'))
    } finally {
      setPickerBusy('')
    }
  }

  const removeSource = async (library: Library, sourceId: string) => {
    try {
      await runMutation(`remove-source:${sourceId}`, () => api.libraries.removeSource(library.id, sourceId), tr('Meediakaust eemaldati kogust.'))
    } catch {
      // The shared mutation handler already rendered the error.
    }
  }

  const removeLibrary = async (library: Library) => {
    if (!window.confirm(tr('Kas kustutada kogu „{{name}}”? Meediafaile kettalt ei kustutata.', { name: library.name }))) return
    try {
      await runMutation(`delete:${library.id}`, () => api.libraries.remove(library.id), tr('Kogu „{{name}}” kustutati.', { name: library.name }))
    } catch {
      // The shared mutation handler already rendered the error.
    }
  }

  const scanLibrary = async (library: Library) => {
    setError('')
    setMessage('')
    try {
      await startScan(library)
      setMessage(tr('Kogu „{{name}}” skannimine käivitati.', { name: library.name }))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Skannimist ei saanud käivitada.'))
    }
  }

  return (
    <section id="settings-libraries" className="settings-panel settings-panel--libraries settings-panel--flat settings-subsection">
      <div className="settings-panel__heading">
        <div>
          <span className="eyebrow">{tr('Sisu')}</span>
          <h2>{tr('Meediakogud ja kaustad')}</h2>
          <p>{tr('Seo KoduMedia Windowsi kaustade või võrguketastega. Skannimisel tuvastatakse filmid, seriaalid, hooajad, episoodid ja tehniline info.')}</p>
        </div>
      </div>

      {error ? <div className="notice notice--error"><Icon name="alert" size={18} />{error}</div> : null}
      {message ? <div className="notice notice--success"><Icon name="check" size={18} />{message}</div> : null}

      <div className="library-settings-list">
        {libraries.map((library) => {
          const job = jobs[library.id]
          const scanning = job?.status === 'queued' || job?.status === 'running'
          const kindLabel = tr(kindOptions.find((option) => option.value === library.kind)?.label || 'Meedia')
          const sourceCount = trCount('{{count}} kaust', '{{count}} kausta', library.sources.length)
          return (
            <article className="library-settings-item" key={library.id}>
              <header className="library-settings-item__header">
                <span className="library-settings-item__icon"><Icon name={kindIcon(library.kind)} size={22} /></span>
                <div className="library-settings-item__copy">
                  <div className="library-settings-item__meta">
                    <span className="library-kind-badge">{kindLabel}</span>
                    <span>{library.system ? tr('Põhikogu') : tr('Kohandatud kogu')}</span>
                  </div>
                  <h3>{library.name}</h3>
                  <p>{sourceCount} · {tr('Meediafailid jäävad alati algsesse asukohta.')}</p>
                </div>
                <div className="library-settings-item__actions">
                  <Button compact variant="secondary" icon="refresh" disabled={scanning || library.sources.length === 0} onClick={() => void scanLibrary(library)}>
                    {scanning ? tr('Skannin…') : tr('Skanni')}
                  </Button>
                  {!library.system ? <Button compact variant="ghost" icon="trash" disabled={busyKey === `delete:${library.id}`} onClick={() => void removeLibrary(library)} aria-label={tr('Kustuta {{name}}', { name: library.name })} /> : null}
                </div>
              </header>

              <div className="library-settings-item__body">
                <JobProgress job={job} />

                <div className="library-source-group">
                  <div className="library-source-group__heading">
                    <div>
                      <strong>{tr('Seotud kaustad')}</strong>
                      <small>{tr('KoduMedia jälgib ja skannib ainult allolevaid asukohti.')}</small>
                    </div>
                    <span>{sourceCount}</span>
                  </div>

                  {library.sources.length > 0 ? (
                    <div className="library-source-list">
                      {library.sources.map((source) => (
                        <div className="library-source-row" key={source.id}>
                          <span className="library-source-row__icon"><Icon name="folder" size={17} /></span>
                          <div className="library-source-row__copy">
                            <strong>{tr('Meediakaust')}</strong>
                            <code title={source.path}>{source.path}</code>
                          </div>
                          <Button compact variant="ghost" icon="close" disabled={busyKey === `remove-source:${source.id}`} onClick={() => void removeSource(library, source.id)} aria-label={tr('Eemalda kaust {{path}}', { path: source.path })} />
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="library-source-empty">
                      <span><Icon name="folder" size={19} /></span>
                      <div><strong>{tr('Kaustu pole veel lisatud')}</strong><p><span>{tr('Selle koguga pole veel ühtegi kausta seotud.')}</span> {tr('Vali serveri kaust, et KoduMedia saaks selle sisu leida.')}</p></div>
                    </div>
                  )}
                </div>

                <div className="library-add-source">
                  <div><strong>{tr('Lisa uus meediakaust')}</strong><small>{tr('Kasuta Windowsi kaustavalijat; aadressi pole vaja käsitsi kirjutada.')}</small></div>
                  <Button type="button" variant="secondary" icon="folder" disabled={busyKey === `source:${library.id}` || pickerBusy === library.id} onClick={() => void browseSource(library)}>{pickerBusy === library.id ? tr('Avan Windows Explorerit…') : tr('Sirvi kaustu')}</Button>
                </div>
              </div>
            </article>
          )
        })}
      </div>

      <form className="create-library create-library--standalone" onSubmit={(event) => void createLibrary(event)}>
        <div className="create-library__intro">
          <span className="create-library__icon"><Icon name="plus" size={21} /></span>
          <div>
            <span className="eyebrow">{tr('Uus kogu')}</span>
            <h3>{tr('Loo oma kogu')}</h3>
            <p>{tr('Näiteks Kontserdid, Koduvideod või Lastele. Segakogus tuvastatakse video tüüp failinime ja kaustastruktuuri järgi.')}</p>
          </div>
        </div>
        <div className="create-library__controls">
          <label className="create-library__field">
            <span>{tr('Nimi')}</span>
            <input value={name} onChange={(event) => setName(event.target.value)} placeholder={tr('Minu kogu')} data-focusable="true" />
          </label>
          <label className="create-library__field">
            <span>{tr('Sisutüüp')}</span>
            <SelectControl value={kind} onChange={(event) => setKind(event.target.value as LibraryKind)}>
              {kindOptions.map((option) => <option key={option.value} value={option.value}>{tr(option.label)}</option>)}
            </SelectControl>
          </label>
          <Button type="submit" variant="primary" icon="plus" disabled={!name.trim() || busyKey === 'create'}>{tr('Loo kogu')}</Button>
        </div>
      </form>

      {pickerLibrary ? (
        <FileBrowserDialog
          title={tr('Vali kaust kogule „{{name}}”', { name: pickerLibrary.name })}
          description={tr('Windowsi native valija pole selles brauseris saadaval. Kasuta serveripoolset kiiret varuvalijat.')}
          mode="directory"
          initialPath={pickerLibrary.sources[0]?.path || ''}
          onClose={() => setPickerLibrary(null)}
          onSelect={(path) => void addSource(pickerLibrary, path)}
        />
      ) : null}
    </section>
  )
}
