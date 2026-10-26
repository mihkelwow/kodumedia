import { memo } from 'react'
import type { MediaItem } from '../../api/types'
import { Icon } from '../../shared/components/Icon'
import { tr, trCount } from '../../i18n/i18n'

export type MediaCardVariant = 'poster' | 'landscape' | 'search'

function subtitle(item: MediaItem): string {
  if (item.type === 'episode') {
    const episode = item.season && item.episode ? `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}` : ''
    return [item.seriesTitle, episode].filter(Boolean).join(' · ')
  }
  if (item.type === 'series') {
    if (item.watch?.season && item.watch?.episode) {
      return `${tr('Jätka')} · S${String(item.watch.season).padStart(2, '0')}E${String(item.watch.episode).padStart(2, '0')}`
    }
    const seasons = item.seasonCount ? trCount('{{count}} hooaeg', '{{count}} hooaega', item.seasonCount) : ''
    const episodes = item.episodeCount ? trCount('{{count}} osa', '{{count}} osa', item.episodeCount) : ''
    return [item.year, seasons, episodes].filter(Boolean).join(' · ')
  }
  if (item.type === 'music') return [item.artist, item.album].filter(Boolean).join(' · ')
  return item.year ? String(item.year) : ''
}

function visualUrl(item: MediaItem, variant: MediaCardVariant): string | undefined {
  if (variant === 'landscape') return item.metadata.backdropUrl || item.metadata.posterUrl
  return item.metadata.posterUrl || item.metadata.backdropUrl
}

interface MediaCardProps {
  item: MediaItem
  onOpen: (item: MediaItem) => void
  variant?: MediaCardVariant
  onPreview?: (item: MediaItem) => void
  onPreviewImmediate?: (item: MediaItem) => void
  onPreviewCancel?: () => void
}

export const MediaCard = memo(function MediaCard({ item, onOpen, variant = 'poster', onPreview, onPreviewImmediate, onPreviewCancel }: MediaCardProps) {
  const label = item.type === 'episode' && item.seriesTitle ? `${item.seriesTitle}: ${item.title}` : item.title
  const image = visualUrl(item, variant)
  const cardSubtitle = subtitle(item) || tr('Kohalik meedia')
  return (
    <button
      className={`media-card media-card--${variant}`}
      onClick={() => onOpen(item)}
      onMouseEnter={() => onPreview?.(item)}
      onMouseLeave={onPreviewCancel}
      onFocus={() => (onPreviewImmediate ?? onPreview)?.(item)}
      data-focusable="true"
      aria-label={tr('Ava {{title}}', { title: label })}
    >
      <span className="media-card__visual">
        {image ? (
          <img src={image} alt="" loading="lazy" decoding="async" referrerPolicy="no-referrer" draggable={false} />
        ) : (
          <span className="media-card__placeholder" aria-hidden="true">
            <span>{label.slice(0, 1).toUpperCase()}</span>
          </span>
        )}
        <span className="media-card__shade" />
        {item.metadata.imdbRating ? (
          <span className="rating-badge"><b>IMDb</b>{item.metadata.imdbRating.toFixed(1)}</span>
        ) : null}
        {variant === 'landscape' ? (
          <span className="media-card__landscape-copy">
            <strong>{label}</strong>
            <small>{cardSubtitle}</small>
          </span>
        ) : null}
        {item.watch && item.watch.progress > 0 ? (
          <span className="media-card__progress" aria-label={tr('Vaadatud {{percent}}%', { percent: Math.round(item.watch.progress * 100) })}>
            <span style={{ transform: `scaleX(${Math.min(1, Math.max(0, item.watch.progress))})` }} />
          </span>
        ) : null}
        <span className="media-card__play"><Icon name={item.type === 'series' ? 'chevronRight' : 'play'} size={22} /></span>
      </span>
      {variant !== 'landscape' ? (
        <span className="media-card__copy">
          <strong title={label}>{label}</strong>
          <small>{cardSubtitle}</small>
        </span>
      ) : null}
    </button>
  )
})
