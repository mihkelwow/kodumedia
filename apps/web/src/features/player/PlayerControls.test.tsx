import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { PlayerControls } from './PlayerControls'

const common = {
  currentSeconds: 10,
  durationSeconds: 100,
  bufferedAhead: 20,
  seekDraft: null,
  muted: false,
  volume: 0.8,
  trackMenus: null,
  pictureInPictureAvailable: false,
  onSeekDraftChange: vi.fn(),
  onSeek: vi.fn(),
  onTogglePlayback: vi.fn(),
  onToggleMuted: vi.fn(),
  onVolumeChange: vi.fn(),
  onPictureInPicture: vi.fn(),
  onFullscreen: vi.fn(),
}

describe('PlayerControls', () => {
  it('removes hidden controls from the keyboard focus order while video is playing', () => {
    const { container } = render(<PlayerControls {...common} visible={false} playing />)
    const controls = container.querySelector('.player-controls') as HTMLElement
    expect(controls.getAttribute('aria-hidden')).toBe('true')
    expect(controls.hasAttribute('inert')).toBe(true)
  })

  it('keeps controls interactive while paused', () => {
    render(<PlayerControls {...common} visible={false} playing={false} />)
    expect(screen.getByRole('button', { name: 'Esita' })).toBeTruthy()
    expect(screen.getByRole('slider', { name: 'Esituse ajajoon' })).toBeTruthy()
  })
})
