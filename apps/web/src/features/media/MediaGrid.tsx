import type { MediaItem } from '../../api/types'
import { MediaCard } from './MediaCard'

export function MediaGrid({ items, onOpen }: { items: MediaItem[]; onOpen: (item: MediaItem) => void }) {
  return <div className="media-grid">{items.map((item) => <MediaCard key={item.id} item={item} onOpen={onOpen} />)}</div>
}
