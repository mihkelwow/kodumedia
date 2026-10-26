import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../../api/client'
import type { MediaItem, SeriesDetails as SeriesDetailsPayload } from '../../api/types'
import { optimizeAmbientImageUrl } from '../../shared/components/AmbientBackdrop'
import { Button } from '../../shared/components/Button'
import { Dialog } from '../../shared/components/Dialog'
import { Icon } from '../../shared/components/Icon'
import { Spinner } from '../../shared/components/Spinner'
import { formatDate, formatVotes } from '../../shared/utils/format'
import { preloadPlayback } from '../player/preload'
import { CastCarousel } from './CastCarousel'
import { EpisodeCarousel, episodeCode } from './EpisodeCarousel'
import { MediaPlayButton } from './MediaPlayButton'
import { tr, trCount } from '../../i18n/i18n'

interface SeriesDetailsProps {
  item: MediaItem
  onClose: () => void
  onOpenEpisode: (episode: MediaItem) => void
  onPlayEpisode?: (episode: MediaItem) => void
}

function seasonCountLabel(value: number): string {
  return trCount('{{count}} hooaeg', '{{count}} hooaega', value)
}

function episodeCountLabel(value: number): string {
  return trCount('{{count}} osa', '{{count}} osa (mitmus)', value)
}

function serverEpisodeCountLabel(value: number): string {
  return trCount('{{count}} osa serveris', '{{count}} osa serveris (mitmus)', value)
}

function playButtonLabel(target?: MediaItem): string {
  if (target?.watch && !target.watch.completed && target.watch.progress >= 0.01 && target.watch.progress <= 0.95) return tr('Jätka')
  return tr('Esita')
}


export function SeriesDetails({ item: initialItem, onClose, onOpenEpisode, onPlayEpisode = onOpenEpisode }: SeriesDetailsProps) {
  const [details, setDetails] = useState<SeriesDetailsPayload | null>(null)
  const [activeSeason, setActiveSeason] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const result = await api.series.get(initialItem.id)
      setDetails(result)
      setActiveSeason((current) => {
        if (current !== null && result.seasons.some((season) => season.number === current)) return current
        if (result.playTarget?.season !== undefined && result.seasons.some((season) => season.number === result.playTarget?.season)) {
          return result.playTarget.season ?? null
        }
        return result.seasons[0]?.number ?? null
      })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : tr('Seriaali osi ei saanud laadida.'))
    } finally {
      setLoading(false)
    }
  }, [initialItem.id])

  useEffect(() => { void load() }, [load])
  useEffect(() => {
    if (details?.playTarget?.id) preloadPlayback(details.playTarget.id)
  }, [details?.playTarget?.id])

  const series = details?.series ?? initialItem
  const selectedSeason = useMemo(
    () => details?.seasons.find((season) => season.number === activeSeason) ?? details?.seasons[0],
    [activeSeason, details?.seasons],
  )
  const backdrop = optimizeAmbientImageUrl(series.metadata.backdropUrl || series.metadata.posterUrl || '')
  const availability = details?.availability
  const latestReleased = availability?.latestReleasedEpisode

  return (
    <Dialog
      onClose={onClose}
      layerClassName="modal-layer details-layer"
      className="details details--cinematic series-details"
      ariaLabelledBy="series-details-title"
    >
        <div className="details__background" style={backdrop ? { backgroundImage: `url(${backdrop})` } : undefined} aria-hidden="true" />
        <div className="details__background-wash" aria-hidden="true" />
        <Button className="details__close" compact variant="ghost" icon="close" onClick={onClose} aria-label={tr('Sulge')} />

        <div className="details__content">
          <header className="details__title-block">
            <span className="eyebrow">{tr('Seriaal')}{series.year ? ` · ${series.year}` : ''}</span>
            {series.metadata.logoUrl ? <img className="details__title-logo" src={series.metadata.logoUrl} alt={series.title} referrerPolicy="no-referrer" /> : <h2 id="series-details-title">{series.title}</h2>}
            {series.metadata.logoUrl ? <h2 id="series-details-title" className="sr-only">{series.title}</h2> : null}
          </header>

          <div className="details__overview-grid details__overview-grid--series">
            <div className="details__actions-column">
              {details?.playTarget ? (
                <div className="series-play-action">
                  <MediaPlayButton onClick={() => onPlayEpisode(details.playTarget as MediaItem)}>
                    {playButtonLabel(details.playTarget)}
                  </MediaPlayButton>
                  <small>{episodeCode(details.playTarget)}</small>
                </div>
              ) : null}
              <div className="details__round-actions">
                {series.externalIds.imdb ? (
                  <a className="round-tool" href={`https://www.imdb.com/title/${series.externalIds.imdb}/`} target="_blank" rel="noreferrer" aria-label={tr('Ava IMDb')} data-focusable="true">
                    <Icon name="star" size={19} />
                  </a>
                ) : null}
                <button type="button" className="round-tool" onClick={() => document.getElementById(`seasons-${series.id}`)?.scrollIntoView({ behavior: 'auto' })} aria-label={tr('Ava hooajad')} data-focusable="true">
                  <Icon name="tv" size={19} />
                </button>
              </div>
            </div>
            <div className="details__description-column">
              {series.metadata.tagline ? <p className="details__tagline">{series.metadata.tagline}</p> : null}
              <p className="details__summary">{series.metadata.summary || tr('Kirjeldust pole veel laaditud. Kui TMDb on seadistatud, lisab server selle taustal automaatselt.')}</p>
            </div>
            <div className="details__metadata-column">
              <div className="details__primary-facts">
                {series.year ? <strong>{series.year}</strong> : null}
                {series.metadata.imdbRating ? <strong className="details__imdb"><span>IMDb</span>{series.metadata.imdbRating.toFixed(1)}</strong> : null}
              </div>
              {availability?.newEpisodeAvailable && latestReleased ? (
                <div className="series-release-callout" role="status" aria-label={tr('Uuem osa on ilmunud')}>
                  <span className="series-release-callout__icon" aria-hidden="true">
                    <Icon name="sparkles" size={18} />
                  </span>
                  <div className="series-release-callout__copy">
                    <small>{tr('Uuem osa on ilmunud')}</small>
                    <strong>{`S${String(latestReleased.season).padStart(2, '0')}E${String(latestReleased.episode).padStart(2, '0')}`}{latestReleased.title ? ` · ${latestReleased.title}` : ''}</strong>
                    {latestReleased.releaseDate ? <span>{formatDate(latestReleased.releaseDate)}</span> : null}
                  </div>
                </div>
              ) : null}
              {availability ? (
                <div className="series-availability" aria-label={tr('Seriaali saadavus')}>
                  <span><b>{tr('Väljas')}</b>{seasonCountLabel(availability.releasedSeasonCount)} · {episodeCountLabel(availability.releasedEpisodeCount)}</span>
                  <span><b>{tr('Vaata kohe')}</b>{seasonCountLabel(availability.availableSeasonCount)} · {episodeCountLabel(availability.availableEpisodeCount)}</span>
                </div>
              ) : null}
              {series.metadata.genres?.length ? <p className="details__genres">{series.metadata.genres.join('  |  ')}</p> : null}
              {series.metadata.imdbVotes ? <small>{tr('{{count}} IMDb hinnangut', { count: formatVotes(series.metadata.imdbVotes) })}</small> : null}
            </div>
          </div>

          <section className="season-browser season-browser--cinematic" id={`seasons-${series.id}`} aria-labelledby="season-browser-title">
            <div className="season-browser__heading">
              <div>
                <span className="eyebrow">{tr('Vaata kohe')}</span>
                <h3 id="season-browser-title">{tr('Hooajad ja osad')}</h3>
              </div>
              {details ? <span>{seasonCountLabel(details.seasons.length)} · {serverEpisodeCountLabel(series.episodeCount || 0)}</span> : null}
            </div>

            {error ? <p className="inline-error"><Icon name="alert" size={17} />{error}</p> : null}
            {loading ? <div className="series-details__state"><Spinner label={tr('Laen hooaegu')} /><span>{tr('Laen seriaali osi…')}</span></div> : null}
            {!loading && !error && details?.seasons.length === 0 ? (
              <div className="series-details__state"><Icon name="tv" size={28} /><span>{tr('Selle seriaali alla pole veel ühtegi episoodi rühmitatud.')}</span></div>
            ) : null}

            {!loading && details?.seasons.length ? (
              <>
                {details.seasons.length > 1 ? (
                  <div className="season-tabs-shell">
                    <div className="season-tabs" role="tablist" aria-label={tr('Hooajad')}>
                      {details.seasons.map((season) => (
                        <button
                          type="button"
                          role="tab"
                          aria-selected={season.number === selectedSeason?.number}
                          className={`season-tab ${season.number === selectedSeason?.number ? 'season-tab--active' : ''}`}
                          key={season.number}
                          onClick={() => setActiveSeason(season.number)}
                          data-focusable="true"
                        >
                          <span>{season.title}</span>
                        </button>
                      ))}
                    </div>
                    {selectedSeason ? <span className="season-tabs__meta" aria-live="polite">{episodeCountLabel(selectedSeason.episodeCount)}</span> : null}
                  </div>
                ) : selectedSeason ? (
                  <div className="season-tabs-single">
                    <strong>{selectedSeason.title}</strong>
                    <span>{episodeCountLabel(selectedSeason.episodeCount)}</span>
                  </div>
                ) : null}

                <div role="tabpanel" aria-label={selectedSeason?.title}>
                  {selectedSeason?.episodes.length ? (
                    <EpisodeCarousel episodes={selectedSeason.episodes} label={selectedSeason.title} onOpenEpisode={onOpenEpisode} />
                  ) : (
                    <div className="series-details__state"><Icon name="tv" size={28} /><span>{tr('Selles hooajas pole veel ühtegi episoodi.')}</span></div>
                  )}
                </div>
              </>
            ) : null}
          </section>

          {series.metadata.cast?.length ? (
            <section className="details__cast-section">
              <h3>{tr('Näitlejad ja tegijad')}</h3>
              <CastCarousel people={series.metadata.cast} />
            </section>
          ) : null}
        </div>
    </Dialog>
  )
}
