import type { CSSProperties, ReactNode } from 'react'
import { tr } from '../../i18n/i18n'
import { IconButton } from '../../shared/components/IconButton'
import { PlayerTimeline } from './PlayerTimeline'

interface PlayerControlsProps {
  visible: boolean
  playing: boolean
  currentSeconds: number
  durationSeconds: number
  bufferedAhead: number
  seekDraft: number | null
  muted: boolean
  volume: number
  trackMenus: ReactNode
  pictureInPictureAvailable: boolean
  diagnostics?: ReactNode
  status?: ReactNode
  onSeekDraftChange: (seconds: number | null) => void
  onSeek: (seconds: number) => void
  onTogglePlayback: () => void
  onToggleMuted: () => void
  onVolumeChange: (volume: number) => void
  onPictureInPicture: () => void
  onFullscreen: () => void
}

/** Player transport and tools shell. Playback session logic remains in the controller hook/component. */
export function PlayerControls({
  visible,
  playing,
  currentSeconds,
  durationSeconds,
  bufferedAhead,
  seekDraft,
  muted,
  volume,
  trackMenus,
  pictureInPictureAvailable,
  diagnostics,
  status,
  onSeekDraftChange,
  onSeek,
  onTogglePlayback,
  onToggleMuted,
  onVolumeChange,
  onPictureInPicture,
  onFullscreen,
}: PlayerControlsProps) {
  const hidden = !visible && playing
  const volumeStyle = {
    '--player-volume': `${Math.round((muted ? 0 : volume) * 100)}%`,
  } as CSSProperties

  return (
    <div
      className="player-controls"
      aria-hidden={hidden}
      inert={hidden || undefined}
    >
      <div className="player-controls__main">
        <div className="player-controls__transport">
          <IconButton icon="rewind" label={tr('10 sekundit tagasi')} className="player-control-button" onClick={() => onSeek(currentSeconds - 10)} />
          <IconButton
            icon={playing ? 'pause' : 'play'}
            iconSize={24}
            label={playing ? tr('Paus') : tr('Esita')}
            variant="primary"
            className="player-control-button player-control-button--primary"
            onClick={onTogglePlayback}
          />
          <IconButton icon="forward" label={tr('10 sekundit edasi')} className="player-control-button" onClick={() => onSeek(currentSeconds + 10)} />
        </div>

        <PlayerTimeline
          currentSeconds={currentSeconds}
          durationSeconds={durationSeconds}
          bufferedAhead={bufferedAhead}
          draftSeconds={seekDraft}
          onDraftChange={onSeekDraftChange}
          onSeek={onSeek}
        />

        <div className="player-controls__tools">
          <IconButton
            icon={muted || volume === 0 ? 'mute' : 'volume'}
            label={muted ? tr('Lülita heli sisse') : tr('Vaigista')}
            className="player-control-button"
            onClick={onToggleMuted}
          />
          <input
            className="player-volume"
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={muted ? 0 : volume}
            style={volumeStyle}
            aria-label={tr('Helitugevus')}
            onChange={(event) => onVolumeChange(Number(event.target.value))}
          />
          {trackMenus}
          {pictureInPictureAvailable ? (
            <IconButton icon="pip" label={tr('Pilt-pildis')} className="player-control-button" onClick={onPictureInPicture} />
          ) : null}
          <IconButton icon="fullscreen" label={tr('Täisekraan')} className="player-control-button" onClick={onFullscreen} />
        </div>
      </div>
      {diagnostics}
      {status}
    </div>
  )
}
