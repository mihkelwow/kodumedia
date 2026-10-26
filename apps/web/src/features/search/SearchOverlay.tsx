import { useEffect, useMemo, useRef, useState } from 'react'
import type { MediaItem, MediaType } from '../../api/types'
import { api } from '../../api/client'
import { MediaCard } from '../media/MediaCard'
import { tr, type TranslationKey } from '../../i18n/i18n'
import { Dialog } from '../../shared/components/Dialog'
import { Icon } from '../../shared/components/Icon'
import { IconButton } from '../../shared/components/IconButton'
import { Spinner } from '../../shared/components/Spinner'
import { useDebouncedValue } from '../../shared/hooks/useDebouncedValue'

interface SearchOverlayProps {
  onClose: () => void
  onOpen: (item: MediaItem) => void
}

const groups: Array<{ type: MediaType; title: TranslationKey }> = [
  { type: 'movie', title: 'Filmid' },
  { type: 'series', title: 'Seriaalid' },
  { type: 'episode', title: 'Episoodid' },
  { type: 'music', title: 'Muusika' },
]

export function SearchOverlay({ onClose, onOpen }: SearchOverlayProps) {
  const [query, setQuery] = useState('')
  const [items, setItems] = useState<MediaItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const debouncedQuery = useDebouncedValue(query, 180)

  useEffect(() => {
    const value = debouncedQuery.trim()
    if (value.length < 2) {
      setItems([])
      setLoading(false)
      setError('')
      return
    }

    const controller = new AbortController()
    setLoading(true)
    setError('')
    api.media.list({
      type: ['movie', 'series', 'episode', 'music'],
      search: value,
      sort: 'title',
      order: 'asc',
      limit: 80,
    }, controller.signal)
      .then((response) => setItems(response.items))
      .catch((caught) => {
        if (controller.signal.aborted) return
        setError(caught instanceof Error ? caught.message : tr('Otsing ebaõnnestus.'))
      })
      .finally(() => { if (!controller.signal.aborted) setLoading(false) })

    return () => controller.abort()
  }, [debouncedQuery])

  const grouped = useMemo(() => groups.map((group) => ({
    ...group,
    items: items.filter((item) => item.type === group.type),
  })).filter((group) => group.items.length > 0), [items])

  const openItem = (item: MediaItem) => {
    onClose()
    onOpen(item)
  }

  return (
    <Dialog
      onClose={onClose}
      ariaLabel={tr('Otsi meediast')}
      layerClassName="search-overlay"
      className="search-overlay__surface"
      initialFocusRef={inputRef}
      motion="instant"
    >
      <div className="search-overlay__ambient" aria-hidden="true" />
      <header className="search-overlay__header">
        <label className="global-search-field">
          <Icon name="search" size={24} />
          <input
            ref={inputRef}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={tr('Otsi filmi, seriaali, episoodi, näitlejat või žanri')}
            aria-label={tr('Otsing')}
            autoComplete="off"
          />
          {loading ? <Spinner label={tr('Otsin')} /> : null}
          {query ? (
            <IconButton
              className="global-search-field__clear"
              icon="close"
              size="small"
              onClick={() => setQuery('')}
              label={tr('Tühjenda otsing')}
            />
          ) : null}
        </label>
        <IconButton className="search-overlay__close" icon="close" variant="surface" onClick={onClose} label={tr('Sulge otsing')} />
      </header>

      <div className="search-overlay__content">
        {error ? <div className="notice notice--error"><Icon name="alert" size={18} />{error}</div> : null}
        {!query.trim() ? (
          <div className="search-overlay__hint">
            <Icon name="search" size={34} />
            <h2>{tr('Leia kiiresti kõik oma kogust')}</h2>
            <p>{tr('Otsing leiab vasteid ametliku pealkirja, näitleja, tegelase, žanri, albumi ja failinime järgi.')}</p>
          </div>
        ) : null}
        {query.trim().length === 1 ? <p className="search-overlay__minimum">{tr('Sisesta vähemalt kaks tähemärki.')}</p> : null}
        {!loading && query.trim().length >= 2 && !error && items.length === 0 ? (
          <div className="search-overlay__hint search-overlay__hint--compact">
            <h2>{tr('Vasteid ei leitud')}</h2>
            <p>{tr('Proovi ametlikku pealkirja, näitleja nime või mõnda lühemat märksõna.')}</p>
          </div>
        ) : null}
        {grouped.map((group) => (
          <section className="search-results-group" key={group.type}>
            <header><h2>{tr(group.title)}</h2><span>{group.items.length}</span></header>
            <div className="search-results-grid">
              {group.items.map((item) => <MediaCard key={item.id} item={item} onOpen={openItem} variant="search" />)}
            </div>
          </section>
        ))}
      </div>
    </Dialog>
  )
}
