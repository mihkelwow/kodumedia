import { memo } from 'react'
import type { MediaItem } from '../../api/types'
import { MediaCard, type MediaCardVariant } from './MediaCard'

interface MediaRowProps {
  title: string
  description?: string
  items: MediaItem[]
  onOpen: (item: MediaItem) => void
  variant?: MediaCardVariant
  onPreview?: (item: MediaItem) => void
  onPreviewImmediate?: (item: MediaItem) => void
  onPreviewCancel?: () => void
}

export const MediaRow = memo(function MediaRow({ title, description, items, onOpen, variant = 'poster', onPreview, onPreviewImmediate, onPreviewCancel }: MediaRowProps) {
  if (items.length === 0) return null
  return (
    <section className={`media-row media-row--${variant}`}>
      <header className="section-heading">
        <div><h2>{title}</h2>{description ? <p>{description}</p> : null}</div>
      </header>
      <div className="media-row__scroller">
        {items.map((item) => (
          <MediaCard
            key={item.id}
            item={item}
            onOpen={onOpen}
            variant={variant}
            onPreview={onPreview}
            onPreviewImmediate={onPreviewImmediate}
            onPreviewCancel={onPreviewCancel}
          />
        ))}
      </div>
    </section>
  )
})
