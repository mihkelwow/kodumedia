import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { PlayerTimeline } from './PlayerTimeline'

describe('PlayerTimeline', () => {
  it('renders playback and buffer progress with transforms rather than layout width', () => {
    render(
      <PlayerTimeline
        currentSeconds={25}
        durationSeconds={100}
        bufferedAhead={25}
        draftSeconds={null}
        onDraftChange={vi.fn()}
        onSeek={vi.fn()}
      />,
    )

    const timeline = screen.getByRole('slider', { name: 'Esituse ajajoon' })
    const root = timeline.closest('.player-timeline-wrap') as HTMLElement
    const progress = root.querySelector('.player-timeline__progress') as HTMLElement
    const buffered = root.querySelector('.player-timeline__buffered') as HTMLElement

    expect(progress.style.transform).toBe('scaleX(0.25)')
    expect(buffered.style.transform).toBe('scaleX(0.5)')
    expect(progress.style.width).toBe('')
    expect(buffered.style.width).toBe('')
  })
})
