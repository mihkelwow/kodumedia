import type { HDRMode, PlaybackSession } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { Icon } from '../../shared/components/Icon'
import { profileLabel } from './playerLabels'

interface PlayerPlaybackStatusProps {
  session: PlaybackSession
  diagnosticsOpen: boolean
  qualityStatusTitle: string
  qualityStatusDetails: string
  notice: string
  hasHDRSource: boolean
  hdrFallbackActive: boolean
  selectedHDRMode: HDRMode
  onToggleDiagnostics: () => void
  onHDRModeChange: (mode: HDRMode) => void
}

export function PlayerPlaybackStatus({
  session,
  diagnosticsOpen,
  qualityStatusTitle,
  qualityStatusDetails,
  notice,
  hasHDRSource,
  hdrFallbackActive,
  selectedHDRMode,
  onToggleDiagnostics,
  onHDRModeChange,
}: PlayerPlaybackStatusProps) {
  return (
    <div className="player-status">
      <button
        type="button"
        className="player-status__summary"
        onClick={onToggleDiagnostics}
        aria-expanded={diagnosticsOpen}
        title={tr('Ava esituse tehniline info')}
      >
        <span className={`playback-mode playback-mode--${session.plan.mode}`}>{session.plan.mode}</span>
        <span className="playback-mode playback-mode--quality">{profileLabel(session.plan.qualityProfile)}</span>
        {session.plan.preserveHdr ? <span className="playback-mode playback-mode--hdr-live">HDR</span> : null}
        {session.plan.toneMap ? (
          <span className={hdrFallbackActive ? 'playback-mode playback-mode--hdr-fallback' : 'playback-mode playback-mode--hdr'}>
            {hdrFallbackActive ? tr('SDR fallback') : 'HDR→SDR'}
          </span>
        ) : null}
        <span className="player-status__copy">
          <strong>{qualityStatusTitle}</strong>
          <small>{notice || qualityStatusDetails}</small>
        </span>
        <Icon name="info" size={15} />
      </button>
      {hasHDRSource ? (
        <div
          className={`player-hdr-switch ${hdrFallbackActive ? 'player-hdr-switch--fallback' : ''}`}
          role="group"
          aria-label={tr('HDR esitusrežiim')}
          title={hdrFallbackActive ? tr('HDR on eelistusena valitud, kuid aktiivne voog on pärast ebaõnnestunud HDR-katset SDR.') : undefined}
        >
          {(['auto', 'hdr', 'sdr'] as HDRMode[]).map((mode) => (
            <button
              key={mode}
              type="button"
              className={selectedHDRMode === mode ? 'player-hdr-switch__button player-hdr-switch__button--active' : 'player-hdr-switch__button'}
              onClick={() => onHDRModeChange(mode)}
              aria-pressed={selectedHDRMode === mode}
            >
              {mode === 'auto' ? tr('Auto') : mode.toUpperCase()}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
}
