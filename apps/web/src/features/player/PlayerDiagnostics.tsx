import type { MediaItem, PlaybackRuntimeStats, PlaybackSession } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { decoderLabel, encoderLabel, formatBitrate, profileLabel, resolutionLabel } from './playerLabels'

interface PlayerDiagnosticsProps {
  session: PlaybackSession
  item: MediaItem
  videoPathLabel: string
  audioPathLabel?: string
  bufferedAhead: number
  desiredBufferSeconds: number
  runtimeStats: PlaybackRuntimeStats | null
  stallCount: number
  stallDurationMs: number
  notice: string
}

export function PlayerDiagnostics({
  session,
  item,
  videoPathLabel,
  audioPathLabel,
  bufferedAhead,
  desiredBufferSeconds,
  runtimeStats,
  stallCount,
  stallDurationMs,
  notice,
}: PlayerDiagnosticsProps) {
  return (
    <div className="player-diagnostics" role="region" aria-label={tr('Esituse diagnostika')}>
      <div><span>{tr('Profiil')}</span><strong>{profileLabel(session.plan.qualityProfile)}</strong></div>
      <div><span>{tr('Videotee')}</span><strong>{videoPathLabel}</strong></div>
      <div><span>{tr('Väljund')}</span><strong>{resolutionLabel(session.plan, item)} · {session.plan.videoCodec?.toUpperCase() || '—'}</strong></div>
      <div><span>{tr('Heli')}</span><strong>{audioPathLabel || '—'}</strong></div>
      <div><span>{tr('Encoder')}</span><strong>{encoderLabel(session.plan)}</strong></div>
      <div><span>{tr('Dekodeerimine')}</span><strong>{decoderLabel(session.plan)}</strong></div>
      <div><span>{tr('Puhver ees')}</span><strong>{tr('{{current}} s / eesmärk {{target}} s', { current: bufferedAhead.toFixed(1), target: desiredBufferSeconds })}</strong></div>
      <div><span>{tr('Encoder speed')}</span><strong>{runtimeStats?.averageEncoderSpeed ? `${runtimeStats.averageEncoderSpeed.toFixed(2)}×` : '—'}</strong></div>
      <div><span>{tr('Väljundbitikiirus')}</span><strong>{formatBitrate(runtimeStats?.estimatedBitrate || session.plan.maxVideoBitrate)}</strong></div>
      <div><span>VBV</span><strong>{formatBitrate(session.plan.maxVideoBitrate)} / {formatBitrate(session.plan.rateControlBuffer)}</strong></div>
      <div><span>{tr('Puhverdamised')}</span><strong>{stallCount} · {(stallDurationMs / 1000).toFixed(1)} s</strong></div>
      <div><span>{tr('Cache')}</span><strong>{session.plan.cacheHit ? tr('Valmis optimeeritud voog') : session.plan.cachePolicy === 'off' ? tr('Väljas') : tr('Kirjutamine / vajaduspõhine')}</strong></div>
      <p>{notice}</p>
    </div>
  )
}
