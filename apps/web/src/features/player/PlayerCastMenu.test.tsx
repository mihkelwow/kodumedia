import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { CastDevice } from '../../api/types'
import type { GoogleCastSnapshot } from './googleCast'
import { PlayerCastMenu } from './PlayerCastMenu'

const television: CastDevice = {
  id: 'samsung-ru7172',
  name: 'Elutoa teler',
  manufacturer: 'Samsung',
  model: 'RU7172',
  protocol: 'dlna',
  canSeek: true,
  canSetVolume: true,
}

const google: GoogleCastSnapshot = {
  supported: true,
  loading: false,
  available: true,
  connected: false,
  playing: false,
  positionMs: 0,
  durationMs: 0,
  volume: 1,
  muted: false,
  basePositionMs: 0,
}

describe('PlayerCastMenu', () => {
  it('presents native, Google Cast and local-network outputs in one picker', () => {
    const onDLNA = vi.fn()
    render(
      <PlayerCastMenu
        open
        onOpenChange={() => undefined}
        devices={[television]}
        discovering={false}
        connecting={false}
        error=""
        systemSupported
        systemAvailable
        systemConnected={false}
        google={google}
        googleActive={false}
        dlnaSession={null}
        activeRemote={null}
        onRefresh={() => undefined}
        onSystemPicker={() => undefined}
        onGoogleCast={() => undefined}
        onDLNA={onDLNA}
        onStop={() => undefined}
      />,
    )

    expect(screen.getByRole('menuitem', { name: /AirPlay \/ süsteemi seadmed/ })).not.toHaveProperty('disabled', true)
    expect(screen.getByRole('menuitem', { name: /Google Cast/ })).not.toHaveProperty('disabled', true)
    const device = screen.getByRole('menuitem', { name: /Elutoa teler/ })
    expect(device.textContent).toContain('Samsung · RU7172 · DLNA / UPnP')

    fireEvent.click(device)
    expect(onDLNA).toHaveBeenCalledWith(television)
  })
})
