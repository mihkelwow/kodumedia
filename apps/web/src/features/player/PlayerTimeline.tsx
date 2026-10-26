import { useCallback, useState } from 'react'
import { tr } from '../../i18n/i18n'

function formatTime(milliseconds: number): string {
  const total = Math.max(0, Math.floor(milliseconds / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
    : `${minutes}:${String(seconds).padStart(2, '0')}`
}

interface PlayerTimelineProps {
  currentSeconds: number
  durationSeconds: number
  bufferedAhead: number
  draftSeconds: number | null
  onDraftChange: (seconds: number | null) => void
  onSeek: (seconds: number) => void
}

export function PlayerTimeline({
  currentSeconds,
  durationSeconds,
  bufferedAhead,
  draftSeconds,
  onDraftChange,
  onSeek,
}: PlayerTimelineProps) {
  const [hover, setHover] = useState<{ seconds: number; percent: number } | null>(null)
  const shownSeconds = draftSeconds ?? currentSeconds
  const progress = durationSeconds > 0 ? Math.min(1, Math.max(0, shownSeconds / durationSeconds)) : 0
  const buffered = durationSeconds > 0
    ? Math.min(1, Math.max(progress, (currentSeconds + bufferedAhead) / durationSeconds))
    : progress

  const updateHover = useCallback((target: HTMLInputElement, clientX: number) => {
    const rect = target.getBoundingClientRect()
    if (!rect.width) return
    const ratio = Math.min(1, Math.max(0, (clientX - rect.left) / rect.width))
    setHover({ seconds: ratio * Math.max(1, durationSeconds), percent: ratio * 100 })
  }, [durationSeconds])

  return (
    <div className="player-controls__timeline-block">
      <div className="player-controls__timeline-meta" aria-hidden="true">
        <span>{formatTime(Math.round(shownSeconds * 1000))}</span>
        <span>-{formatTime(Math.round(Math.max(0, durationSeconds - shownSeconds) * 1000))}</span>
      </div>
      <div className="player-timeline-wrap">
        {hover ? (
          <div className="player-timeline__tooltip" style={{ left: `${hover.percent}%` }} aria-hidden="true">
            {formatTime(Math.round(hover.seconds * 1000))}
          </div>
        ) : null}
        <span className="player-timeline__track" aria-hidden="true">
          <span className="player-timeline__buffered" style={{ transform: `scaleX(${buffered})` }} />
          <span className="player-timeline__progress" style={{ transform: `scaleX(${progress})` }} />
        </span>
        <input
          className="player-timeline"
          type="range"
          min={0}
          max={Math.max(1, durationSeconds)}
          step={0.1}
          value={Math.min(Math.max(0, shownSeconds), Math.max(1, durationSeconds))}
          aria-label={tr('Esituse ajajoon')}
          onChange={(event) => onDraftChange(Number(event.target.value))}
          onPointerMove={(event) => updateHover(event.currentTarget, event.clientX)}
          onPointerEnter={(event) => updateHover(event.currentTarget, event.clientX)}
          onPointerLeave={() => setHover(null)}
          onPointerUp={(event) => onSeek(Number(event.currentTarget.value))}
          onKeyUp={(event) => {
            if (['ArrowLeft', 'ArrowRight', 'Home', 'End', 'PageUp', 'PageDown'].includes(event.key)) {
              onSeek(Number(event.currentTarget.value))
            }
          }}
        />
      </div>
    </div>
  )
}
