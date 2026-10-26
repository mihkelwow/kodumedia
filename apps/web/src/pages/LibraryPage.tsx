import { useEffect, useState } from 'react'
import type { Library, MediaItem, MediaPage, MediaQuery } from '../api/types'
import { api } from '../api/client'
import { MediaGrid } from '../features/media/MediaGrid'
import { EmptyState } from '../shared/components/EmptyState'
import { Icon } from '../shared/components/Icon'
import { Spinner } from '../shared/components/Spinner'
import { Button } from '../shared/components/Button'
import { useDebouncedValue } from '../shared/hooks/useDebouncedValue'
import { tr, trCount } from '../i18n/i18n'

const initialPage: MediaPage = { items: [], total: 0, limit: 200, offset: 0 }

export function LibraryPage({ library, onOpen, onNavigate }: { library: Library; onOpen: (item: MediaItem) => void; onNavigate: (path: string) => void }) {
  const [search, setSearch] = useState('')
  const [contentType, setContentType] = useState<'all' | 'movie' | 'series' | 'music'>('all')
  const [sort, setSort] = useState<NonNullable<MediaQuery['sort']>>('addedAt')
  const [order, setOrder] = useState<NonNullable<MediaQuery['order']>>('desc')
  const [genre, setGenre] = useState('')
  const [yearFrom, setYearFrom] = useState('')
  const [yearTo, setYearTo] = useState('')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [page, setPage] = useState(initialPage)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const debouncedSearch = useDebouncedValue(search)

  useEffect(() => {
    setSearch('')
    setContentType('all')
    setGenre('')
    setYearFrom('')
    setYearTo('')
    setSort('addedAt')
    setOrder('desc')
    setFiltersOpen(false)
  }, [library.id])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    api.media.list({
      libraryId: library.id,
      type: contentType === 'all' ? ['movie', 'series', 'music'] : contentType,
      search: debouncedSearch,
      sort,
      order,
      genre,
      yearFrom: Number(yearFrom) || undefined,
      yearTo: Number(yearTo) || undefined,
      limit: 200,
    })
      .then((response) => { if (!cancelled) setPage(response) })
      .catch((caught) => { if (!cancelled) setError(caught instanceof Error ? caught.message : tr('Kogu laadimine ebaõnnestus.')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [contentType, debouncedSearch, genre, library.id, order, sort, yearFrom, yearTo])

  const resetFilters = () => {
    setSearch('')
    setContentType('all')
    setGenre('')
    setYearFrom('')
    setYearTo('')
    setSort('addedAt')
    setOrder('desc')
  }

  const hasFilters = Boolean(search || contentType !== 'all' || genre || yearFrom || yearTo || sort !== 'addedAt' || order !== 'desc')
  return (
    <div className="page library-page cinematic-page">
      <header className="library-page__header">
        <div>
          <span className="eyebrow">{tr('Meediakogu')}</span>
          <h1>{library.name}</h1>
          <p>{trCount('{{count}} kirje', '{{count}} kirjet', page.total)}</p>
        </div>
        <div className="library-page__tools">
          <label className="library-quick-search">
            <Icon name="search" size={18} />
            <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder={tr('Otsi selles kogus')} data-focusable="true" />
          </label>
          <button type="button" className={`round-tool ${filtersOpen ? 'round-tool--active' : ''}`} onClick={() => setFiltersOpen((open) => !open)} aria-label={tr('Filtrid')} aria-expanded={filtersOpen} data-focusable="true">
            <Icon name="sliders" size={19} />
          </button>
        </div>
      </header>

      {filtersOpen ? (
        <div className="library-filter-strip">
          <label><span>{tr('Sisutüüp')}</span>
            <select value={contentType} onChange={(event) => setContentType(event.target.value as 'all' | 'movie' | 'series' | 'music')} data-focusable="true">
              <option value="all">{tr('Kõik teosed')}</option>
              <option value="movie">{tr('Filmid')}</option>
              <option value="series">{tr('Seriaalid')}</option>
              <option value="music">{tr('Muusika')}</option>
            </select>
          </label>
          <label><span>{tr('Sordi')}</span>
            <select value={sort} onChange={(event) => setSort(event.target.value as NonNullable<MediaQuery['sort']>)} data-focusable="true">
              <option value="addedAt">{tr('Viimati lisatud')}</option>
              <option value="releaseDate">{tr('Väljalaskekuupäev')}</option>
              <option value="title">{tr('Pealkiri')}</option>
              <option value="rating">{tr('IMDb skoor')}</option>
              <option value="year">{tr('Aasta')}</option>
              <option value="duration">{tr('Kestus')}</option>
            </select>
          </label>
          <label><span>{tr('Järjekord')}</span>
            <select value={order} onChange={(event) => setOrder(event.target.value as 'asc' | 'desc')} data-focusable="true">
              <option value="desc">{tr('Kahanev')}</option>
              <option value="asc">{tr('Kasvav')}</option>
            </select>
          </label>
          <label><span>{tr('Žanr')}</span><input value={genre} onChange={(event) => setGenre(event.target.value)} placeholder={tr('nt Draama')} data-focusable="true" /></label>
          <label><span>{tr('Aastast')}</span><input inputMode="numeric" value={yearFrom} onChange={(event) => setYearFrom(event.target.value.replace(/\D/g, '').slice(0, 4))} placeholder="1900" data-focusable="true" /></label>
          <label><span>{tr('Aastani')}</span><input inputMode="numeric" value={yearTo} onChange={(event) => setYearTo(event.target.value.replace(/\D/g, '').slice(0, 4))} placeholder="2026" data-focusable="true" /></label>
          {hasFilters ? <Button compact variant="ghost" icon="refresh" onClick={resetFilters}>{tr('Lähtesta')}</Button> : null}
        </div>
      ) : null}

      {loading ? <div className="page-loader"><Spinner label={tr('Laen kogu')} /></div> : null}
      {error ? <div className="notice notice--error"><Icon name="alert" />{error}</div> : null}
      {!loading && !error && page.items.length > 0 ? <MediaGrid items={page.items} onOpen={onOpen} /> : null}
      {!loading && !error && page.items.length === 0 ? (
        <EmptyState
          icon={library.sources.length === 0 ? 'folder' : 'search'}
          title={library.sources.length === 0 ? tr('Sellel kogul pole veel meediakausta') : tr('Sobivaid kirjeid ei leitud')}
          description={library.sources.length === 0 ? tr('Lisa seadetes serveri kaust ja käivita skannimine.') : tr('Muuda otsingut või lähtesta filtrid.')}
          action={library.sources.length === 0
            ? <Button variant="primary" icon="plus" onClick={() => onNavigate('/settings')}>{tr('Ava seaded')}</Button>
            : <Button variant="secondary" icon="refresh" onClick={resetFilters}>{tr('Lähtesta filtrid')}</Button>}
        />
      ) : null}
    </div>
  )
}
