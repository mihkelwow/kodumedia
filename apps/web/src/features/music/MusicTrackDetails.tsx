import type { MediaItem } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { Button } from '../../shared/components/Button'
import { Dialog } from '../../shared/components/Dialog'
import { formatDuration } from '../../shared/utils/format'
import { MusicArtwork } from './MusicArtwork'

function formatBytes(bytes: number): string {
  if (!bytes) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) { value /= 1024; index++ }
  return `${value.toFixed(index > 1 ? 1 : 0)} ${units[index]}`
}

function compact(values: Array<string | undefined>): string {
  return values.filter((value): value is string => Boolean(value && value.trim())).join(' · ') || '—'
}

function sourceLabel(source?: string): string {
  switch (source) {
    case 'sidecar': return tr('KoduMedia sidecar')
    case 'embedded': return tr('helifaili tagid')
    case 'musicbrainz': return 'MusicBrainz'
    case 'deezer': return 'Deezer'
    case 'audio-analysis': return tr('audioanalüüs')
    case 'local-inference': return tr('lokaalne hinnang')
    case 'fallback': return tr('varuvalik')
    case 'mixed': return tr('mitu allikat')
    default: return source?.trim() || ''
  }
}

function classificationSource(track: MediaItem, hasGenres: boolean, hasStyles: boolean): string {
  const classification = track.musicClassification
  if (!classification) return ''
  const genreSource = sourceLabel(classification.genreSource || (hasGenres ? classification.source : undefined))
  const styleSource = sourceLabel(classification.styleSource || (hasStyles ? classification.source : undefined))
  if (genreSource && styleSource && genreSource !== styleSource) {
    return tr('Allikad: žanr – {{genre}}; stiil – {{style}}', { genre: genreSource, style: styleSource })
  }
  const source = genreSource || styleSource
  return source ? tr('Allikas: {{source}}', { source }) : ''
}

export function MusicTrackDetails({ track, onClose }: { track: MediaItem; onClose: () => void }) {
  const technical = track.technical
  const genres = track.musicClassification?.genres || track.metadata.genres || []
  const styles = track.musicClassification?.styles || []
  const moods = track.musicClassification?.moods || []
  const quality = compact([
    technical.audioCodec?.toUpperCase(),
    technical.container?.toUpperCase() || track.file.extension.replace('.', '').toUpperCase(),
    technical.audioSampleRate ? `${(technical.audioSampleRate / 1000).toFixed(1)} kHz` : undefined,
    technical.audioBitDepth ? `${technical.audioBitDepth}-bit` : undefined,
    technical.audioChannels ? tr('{{count}} kanalit', { count: technical.audioChannels }) : undefined,
  ])
  const fileFacts = compact([
    technical.durationMs ? formatDuration(technical.durationMs) : undefined,
    formatBytes(track.file.sizeBytes),
    technical.bitrate ? `${Math.round(technical.bitrate / 1000)} kb/s` : undefined,
  ])
  const classification = compact([...genres, ...styles, ...moods])
  const sourceUrl = track.musicClassification?.sourceUrl
  const source = classificationSource(track, genres.length > 0, styles.length > 0)

  return (
    <Dialog
      onClose={onClose}
      layerClassName="modal-layer music-details-layer"
      className="music-track-details"
      ariaLabelledBy="music-track-details-title"
    >
        <Button className="music-track-details__close" compact variant="ghost" icon="close" onClick={onClose} aria-label={tr('Sulge')} />
        <div className="music-track-details__art-frame">
          <MusicArtwork src={track.metadata.posterUrl} alt={track.album || track.title} className="music-track-details__art" />
        </div>
        <div className="music-track-details__copy">
          <span className="eyebrow">{tr('Audiofail')}</span>
          <h2 id="music-track-details-title">{track.title}</h2>
          <p>{track.artist || tr('Tundmatu esitaja')}{track.album ? ` · ${track.album}` : ''}</p>
          <dl>
            <div><dt>{tr('Helikvaliteet')}</dt><dd>{quality}</dd></div>
            <div>
              <dt>{tr('Žanr ja stiil')}</dt>
              <dd>
                {classification === '—' ? tr('Liigitamata') : classification}
                {(source || sourceUrl) ? (
                  <small className="music-track-details__classification-source">
                    {source ? <span>{source}</span> : null}
                    {sourceUrl ? <a href={sourceUrl} target="_blank" rel="noreferrer">{tr('Ava allikaleht')}</a> : null}
                  </small>
                ) : null}
              </dd>
            </div>
            <div><dt>{tr('Algne failinimi')}</dt><dd title={track.file.relativePath}>{track.file.name}</dd></div>
            <div><dt>{tr('Fail ja kestus')}</dt><dd>{fileFacts}</dd></div>
            <div><dt>{tr('Esitustee')}</dt><dd>{tr('Originaalfail või kadudeta PCM-varutee')}</dd></div>
          </dl>
        </div>
    </Dialog>
  )
}
