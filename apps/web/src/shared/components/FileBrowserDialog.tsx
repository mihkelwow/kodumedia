import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../../api/client'
import type { DirectoryListing, FileSystemEntry } from '../../api/types'
import { formatBytes } from '../utils/format'
import { Button } from './Button'
import { Dialog } from './Dialog'
import { IconButton } from './IconButton'
import { tr } from '../../i18n/i18n'
import { Icon } from './Icon'
import { Spinner } from './Spinner'

interface FileBrowserDialogProps {
  title: string
  description?: string
  mode: 'directory' | 'file'
  initialPath?: string
  extensions?: string[]
  onClose: () => void
  onSelect: (path: string) => void
}

const emptyListing: DirectoryListing = { roots: [], entries: [] }

export function FileBrowserDialog({
  title,
  description,
  mode,
  initialPath = '',
  extensions = [],
  onClose,
  onSelect,
}: FileBrowserDialogProps) {
  const [listing, setListing] = useState<DirectoryListing>(emptyListing)
  const [selectedFile, setSelectedFile] = useState<FileSystemEntry | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [requestedPath, setRequestedPath] = useState(initialPath)
  const requestNumber = useRef(0)
  const activeRequest = useRef<AbortController | null>(null)
  const extensionKey = extensions.join(',')
  const normalizedExtensions = useMemo(() => extensionKey ? extensionKey.split(',') : [], [extensionKey])

  const browse = useCallback(async (path: string) => {
    activeRequest.current?.abort()
    const controller = new AbortController()
    activeRequest.current = controller
    const currentRequest = ++requestNumber.current
    setLoading(true)
    setError('')
    setRequestedPath(path)
    setSelectedFile(null)
    try {
      const response = await api.filesystem.browse(path, {
        includeFiles: mode === 'file',
        extensions: normalizedExtensions,
        signal: controller.signal,
      })
      if (currentRequest !== requestNumber.current) return
      setListing({
        ...response,
        roots: Array.isArray(response.roots) ? response.roots : [],
        entries: Array.isArray(response.entries) ? response.entries : [],
      })
    } catch (caught) {
      if (controller.signal.aborted || currentRequest !== requestNumber.current) return
      setError(caught instanceof Error ? caught.message : tr('Kausta sisu ei saanud avada.'))
    } finally {
      if (currentRequest === requestNumber.current) setLoading(false)
    }
  }, [mode, normalizedExtensions])

  useEffect(() => {
    void browse(initialPath)
    return () => activeRequest.current?.abort()
  }, [browse, initialPath])

  const entries = listing.currentPath ? listing.entries : listing.roots
  const selection = mode === 'directory' ? listing.currentPath : selectedFile?.path
  const selectionLabel = mode === 'directory' ? tr('Vali see kaust') : tr('Vali fail')
  const locationLabel = listing.currentPath || tr('Selle Windowsi kettad')

  const activateEntry = (entry: FileSystemEntry) => {
    if (entry.directory) {
      void browse(entry.path)
      return
    }
    setSelectedFile(entry)
  }

  return (
    <Dialog
      onClose={onClose}
      layerClassName="file-browser-layer"
      className="file-browser"
      ariaLabelledBy="file-browser-title"
      motion="standard"
    >
        <header className="file-browser__header">
          <div>
            <span className="eyebrow">{tr('Serveri failisüsteem')}</span>
            <h2 id="file-browser-title">{title}</h2>
            {description ? <p>{description}</p> : null}
          </div>
          <IconButton icon="close" size="small" onClick={onClose} label={tr('Sulge kaustasirvija')} />
        </header>

        <div className="file-browser__location">
          <Button compact variant="ghost" icon="chevronRight" className="file-browser__up" disabled={!listing.parentPath} onClick={() => void browse(listing.parentPath || '')} aria-label={tr('Üks kaust üles')} />
          <Button compact variant="ghost" icon="server" onClick={() => void browse('')} aria-label={tr('Näita Windowsi kettaid')}>{tr('Kettad')}</Button>
          <Icon name="folder" size={18} />
          <code title={locationLabel}>{locationLabel}</code>
        </div>

        <div className="file-browser__content">
          {loading ? <div className="file-browser__state"><Spinner label={tr('Laen kausta')} /></div> : null}
          {error ? (
            <div className="file-browser__state file-browser__state--error">
              <Icon name="alert" size={26} />
              <p>{error}</p>
              <div className="file-browser__recovery">
                <Button variant="secondary" icon="refresh" onClick={() => void browse(requestedPath)}>{tr('Proovi uuesti')}</Button>
                <Button variant="ghost" icon="server" onClick={() => void browse('')}>{tr('Tagasi ketaste juurde')}</Button>
              </div>
            </div>
          ) : null}
          {!loading && !error && listing.truncated ? (
            <div className="file-browser__notice"><Icon name="info" size={16} />{tr('Kaustas on üle 1000 kirje. Kiiruse hoidmiseks kuvame esimesed 1000.')}</div>
          ) : null}
          {!loading && !error && entries.length === 0 ? (
            <div className="file-browser__state"><Icon name="folder" size={28} /><p>{tr('See kaust on tühi või sobivaid faile ei leitud.')}</p></div>
          ) : null}
          {!loading && !error && entries.length > 0 ? (
            <div className="file-browser__list" role="listbox" aria-label={tr('Kaustad ja failid')}>
              {entries.map((entry) => {
                const selected = selectedFile?.path === entry.path
                return (
                  <button
                    type="button"
                    key={entry.path}
                    className={`file-browser__entry ${selected ? 'file-browser__entry--selected' : ''}`}
                    onClick={() => activateEntry(entry)}
                    onDoubleClick={() => { if (!entry.directory && mode === 'file') onSelect(entry.path) }}
                    data-focusable="true"
                    role="option"
                    aria-selected={selected}
                  >
                    <span className="file-browser__entry-icon"><Icon name={entry.directory ? 'folder' : 'film'} size={21} /></span>
                    <span className="file-browser__entry-copy"><strong>{entry.name}</strong><small>{entry.directory ? tr('Kaust') : formatBytes(entry.sizeBytes || 0)}</small></span>
                    {entry.directory ? <Icon name="chevronRight" size={18} /> : selected ? <Icon name="check" size={18} /> : null}
                  </button>
                )
              })}
            </div>
          ) : null}
        </div>

        <footer className="file-browser__footer">
          <div className="file-browser__selection">
            <span>{tr('Valik')}</span>
            <code title={selection || ''}>{selection || (mode === 'directory' ? tr('Ava esmalt sobiv kaust.') : tr('Vali loendist fail.'))}</code>
          </div>
          <div className="file-browser__footer-actions">
            <Button variant="ghost" onClick={onClose}>{tr('Tühista')}</Button>
            <Button variant="primary" icon="check" disabled={!selection} onClick={() => selection && onSelect(selection)}>{selectionLabel}</Button>
          </div>
        </footer>
    </Dialog>
  )
}
