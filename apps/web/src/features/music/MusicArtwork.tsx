import { useState } from 'react'
import { Icon } from '../../shared/components/Icon'

export function MusicArtwork({ src, alt = '', className = '' }: { src?: string; alt?: string; className?: string }) {
  const [failed, setFailed] = useState(false)
  if (!src || failed) {
    return <span className={`music-artwork music-artwork--fallback ${className}`} aria-label={alt}><Icon name="music" size={32} /></span>
  }
  return <img className={`music-artwork ${className}`} src={src} alt={alt} loading="lazy" decoding="async" onError={() => setFailed(true)} />
}
