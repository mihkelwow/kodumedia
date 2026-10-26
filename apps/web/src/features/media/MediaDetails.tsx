import { useEffect } from 'react'
import type { MediaItem } from '../../api/types'
import { Button } from '../../shared/components/Button'
import { optimizeAmbientImageUrl } from '../../shared/components/AmbientBackdrop'
import { Dialog } from '../../shared/components/Dialog'
import { Icon } from '../../shared/components/Icon'
import { formatBytes, formatDate, formatDuration, formatVotes } from '../../shared/utils/format'
import { preloadPlayback } from '../player/preload'
import { CastCarousel } from './CastCarousel'
import { MediaPlayButton } from './MediaPlayButton'
import { tr } from '../../i18n/i18n'

function displayTitle(item: MediaItem): string {
  return item.type === 'episode' && item.seriesTitle ? `${item.seriesTitle} — ${item.title}` : item.title
}

function resolutionLabel(item: MediaItem): string {
  const height = item.technical.height ?? 0
  if (height >= 2160) return '4K'
  if (height >= 1440) return 'QHD'
  if (height >= 1080) return 'Full HD'
  if (height >= 720) return 'HD'
  return ''
}

export function MediaDetails({ item, onClose, onPlay }: { item: MediaItem; onClose: () => void; onPlay: (item: MediaItem) => void }) {
  useEffect(() => { preloadPlayback(item.id) }, [item.id])

  const episodeCode = item.type === 'episode' && item.season && item.episode
    ? `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}`
    : ''
  const title = displayTitle(item)
  const backdrop = optimizeAmbientImageUrl(item.metadata.backdropUrl || item.metadata.posterUrl || '')
  const resolution = resolutionLabel(item)
  const fullPath = item.file.path || item.file.relativePath

  return (
    <Dialog
      onClose={onClose}
      layerClassName="modal-layer details-layer"
      className="details details--cinematic"
      ariaLabelledBy="details-title"
    >
        <div className="details__background" style={backdrop ? { backgroundImage: `url(${backdrop})` } : undefined} aria-hidden="true" />
        <div className="details__background-wash" aria-hidden="true" />
        <Button className="details__close" compact variant="ghost" icon="close" onClick={onClose} aria-label={tr('Sulge')} />

        <div className="details__content">
          <header className="details__title-block">
            <span className="eyebrow">{item.type === 'episode' ? [episodeCode, item.year].filter(Boolean).join(' · ') : item.type === 'movie' ? tr('Film') : tr('Kohalik meedia')}</span>
            {item.metadata.logoUrl ? <img className="details__title-logo" src={item.metadata.logoUrl} alt={title} referrerPolicy="no-referrer" /> : <h2 id="details-title">{title}</h2>}
            {item.metadata.logoUrl ? <h2 id="details-title" className="sr-only">{title}</h2> : null}
          </header>

          <div className="details__overview-grid">
            <div className="details__actions-column">
              <div className="details__actions">
                <MediaPlayButton onClick={() => onPlay(item)}>{tr('Esita')}</MediaPlayButton>
              </div>
              <div className="details__round-actions" aria-label={tr('Lisatoimingud')}>
                {item.externalIds.imdb ? (
                  <a className="round-tool" href={`https://www.imdb.com/title/${item.externalIds.imdb}/`} target="_blank" rel="noreferrer" aria-label={tr('Ava IMDb')} data-focusable="true">
                    <Icon name="star" size={19} />
                  </a>
                ) : null}
                <button type="button" className="round-tool" aria-label={tr('Näita tehnilist infot')} onClick={() => document.getElementById(`technical-${item.id}`)?.scrollIntoView({ behavior: 'auto' })} data-focusable="true">
                  <Icon name="info" size={19} />
                </button>
              </div>
            </div>

            <div className="details__description-column">
              {item.metadata.tagline ? <p className="details__tagline">{item.metadata.tagline}</p> : null}
              <p className="details__summary">{item.metadata.summary || tr('Kirjeldust pole veel laaditud. Kui TMDb on seadistatud, lisab server selle taustal automaatselt.')}</p>
            </div>

            <div className="details__metadata-column">
              <div className="details__primary-facts">
                {item.technical.durationMs ? <strong>{formatDuration(item.technical.durationMs)}</strong> : null}
                {item.year ? <strong>{item.year}</strong> : null}
                {item.metadata.imdbRating ? <strong className="details__imdb"><span>IMDb</span>{item.metadata.imdbRating.toFixed(1)}</strong> : null}
              </div>
              {item.metadata.genres?.length ? <p className="details__genres">{item.metadata.genres.join('  |  ')}</p> : null}
              <div className="details__quality-badges">
                {resolution ? <span>{resolution}</span> : null}
                {item.technical.hdrFormat ? <span>{item.technical.hdrFormat}</span> : null}
                {item.technical.audioChannels && item.technical.audioChannels >= 6 ? <span>{item.technical.audioChannels > 6 ? '7.1' : '5.1'}</span> : null}
              </div>
              {item.metadata.imdbVotes ? <small>{tr('{{count}} IMDb hinnangut', { count: formatVotes(item.metadata.imdbVotes) })}</small> : null}
            </div>
          </div>

          {item.metadata.cast?.length ? (
            <section className="details__cast-section">
              <h3>{tr('Näitlejad ja tegijad')}</h3>
              <CastCarousel people={item.metadata.cast} />
            </section>
          ) : null}

          <section className="details__technical-band" id={`technical-${item.id}`}>
            <div aria-label={tr('Algne meediafail')}>
              <h3><Icon name="info" size={24} /> {tr('Meediainfo')}</h3>
              <dl>
                <div className="details__file-row"><dt>{tr('Algne failinimi')}</dt><dd title={item.file.name}><strong>{item.file.name}</strong></dd></div>
                <div className="details__file-row"><dt>{tr('Asukoht')}</dt><dd title={fullPath}>{fullPath}</dd></div>
                <div><dt>{tr('Suurus')}</dt><dd>{formatBytes(item.file.sizeBytes) || '—'}</dd></div>
                <div><dt>{tr('Konteiner')}</dt><dd>{item.technical.container?.toUpperCase() || '—'}</dd></div>
              </dl>
            </div>
            <div>
              <h3><Icon name="tv" size={24} /> {tr('Video')}</h3>
              <dl>
                <div><dt>{tr('Codec')}</dt><dd>{item.technical.videoCodec?.toUpperCase() || '—'}</dd></div>
                <div><dt>{tr('Resolutsioon')}</dt><dd>{item.technical.width && item.technical.height ? `${item.technical.width} × ${item.technical.height}` : '—'}</dd></div>
                <div><dt>{tr('Bitisügavus')}</dt><dd>{item.technical.bitDepth ? `${item.technical.bitDepth}-bit` : '—'}</dd></div>
                <div><dt>{tr('Värviruum')}</dt><dd>{item.technical.colorSpace || '—'}</dd></div>
              </dl>
            </div>
            <div>
              <h3><Icon name="volume" size={24} /> {tr('Heli')}</h3>
              <dl>
                <div><dt>{tr('Codec')}</dt><dd>{item.technical.audioCodec?.toUpperCase() || '—'}</dd></div>
                <div><dt>{tr('Kanalid')}</dt><dd>{item.technical.audioChannels || '—'}</dd></div>
                <div><dt>{tr('Bitikiirus')}</dt><dd>{item.technical.bitrate ? `${Math.round(item.technical.bitrate / 1000)} kbps` : '—'}</dd></div>
              </dl>
            </div>
            <div>
              <h3><Icon name="captions" size={24} /> {tr('Metaandmed')}</h3>
              <dl>
                <div><dt>{tr('Väljalase')}</dt><dd>{item.releaseDate ? formatDate(item.releaseDate) : item.year || '—'}</dd></div>
                <div><dt>TMDb</dt><dd>{item.externalIds.tmdb || '—'}</dd></div>
                <div><dt>IMDb</dt><dd>{item.externalIds.imdb || '—'}</dd></div>
              </dl>
            </div>
          </section>
        </div>
    </Dialog>
  )
}
