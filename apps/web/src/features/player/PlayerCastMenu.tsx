import type { CastDevice, CastSession } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { Button } from '../../shared/components/Button'
import { Icon } from '../../shared/components/Icon'
import { IconButton } from '../../shared/components/IconButton'
import { Menu, MenuItem } from '../../shared/components/Menu'
import { Popover } from '../../shared/components/Popover'
import { Spinner } from '../../shared/components/Spinner'
import type { GoogleCastSnapshot } from './googleCast'
import type { ActiveRemotePlayback } from './usePlayerCasting'

interface PlayerCastMenuProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  devices: CastDevice[]
  discovering: boolean
  connecting: boolean
  error: string
  systemSupported: boolean
  systemAvailable: boolean
  systemConnected: boolean
  google: GoogleCastSnapshot
  googleActive: boolean
  dlnaSession: CastSession | null
  activeRemote: ActiveRemotePlayback | null
  onRefresh: () => void
  onSystemPicker: () => void
  onGoogleCast: () => void
  onDLNA: (device: CastDevice) => void
  onStop: () => void
}

function deviceDescription(device: CastDevice): string {
  return [device.manufacturer, device.model, 'DLNA / UPnP'].filter(Boolean).join(' · ')
}

export function PlayerCastMenu({
  open,
  onOpenChange,
  devices,
  discovering,
  connecting,
  error,
  systemSupported,
  systemAvailable,
  systemConnected,
  google,
  googleActive,
  dlnaSession,
  activeRemote,
  onRefresh,
  onSystemPicker,
  onGoogleCast,
  onDLNA,
  onStop,
}: PlayerCastMenuProps) {
  const active = systemConnected || Boolean(activeRemote)
  const activeName = activeRemote?.deviceName || (systemConnected ? tr('AirPlay-seade') : '')
  const triggerLabel = activeName
    ? tr('Esitatakse seadmes {{device}}', { device: activeName })
    : tr('Esita teises seadmes')

  return (
    <Popover
      open={open}
      onOpenChange={onOpenChange}
      placement="top-end"
      role="dialog"
      ariaLabel={tr('Esita teises seadmes')}
      autoFocus
      rootClassName="player-menu player-cast-menu"
      className="player-menu-panel player-cast-menu__panel"
      trigger={(
        <IconButton
          icon="cast"
          label={triggerLabel}
          className={`player-control-button player-control-button--menu ${open ? 'player-control-button--menu-open' : ''} ${active ? 'player-control-button--menu-active' : ''}`}
          aria-pressed={active}
        />
      )}
    >
      <div className="player-menu-panel__header player-cast-menu__header">
        <strong>{tr('Esita teises seadmes')}</strong>
        <small>{tr('Vali AirPlay, Google Cast või samas võrgus olev teler.')}</small>
      </div>

      {activeName ? (
        <div className="player-cast-menu__active" role="status" aria-live="polite">
          <span className="player-cast-menu__active-icon"><Icon name="cast" size={19} /></span>
          <span className="player-cast-menu__active-copy">
            <strong>{activeName}</strong>
            <small>{activeRemote?.playing || systemConnected ? tr('Esitatakse praegu') : tr('Esitus on pausil')}</small>
          </span>
          {activeRemote ? (
            <Button compact variant="ghost" onClick={onStop} disabled={connecting}>
              {tr('Lõpeta')}
            </Button>
          ) : (
            <Button compact variant="ghost" onClick={onSystemPicker} disabled={connecting}>
              {tr('Muuda')}
            </Button>
          )}
        </div>
      ) : null}

      <div className="player-cast-menu__section">
        <span className="player-cast-menu__section-label">{tr('Selle seadme väljundid')}</span>
        <Menu role="presentation">
          {systemSupported ? (
            <MenuItem
              icon="cast"
              selected={systemConnected}
              disabled={connecting || (!systemAvailable && !systemConnected)}
              description={systemAvailable || systemConnected
                ? tr('AirPlay ja brauseri süsteemne seadmevalik')
                : tr('Ühtegi AirPlay-seadet pole praegu näha')}
              onClick={onSystemPicker}
            >
              {tr('AirPlay / süsteemi seadmed')}
            </MenuItem>
          ) : null}
          {google.supported || google.loading ? (
            <MenuItem
              icon="cast"
              selected={googleActive}
              disabled={connecting || google.loading || !google.available}
              description={google.loading
                ? tr('Laadin Google Casti tuge…')
                : google.available
                  ? tr('Chromecast ja Google Castiga telerid')
                  : tr('Ühtegi Google Casti seadet pole praegu näha')}
              onClick={onGoogleCast}
            >
              {tr('Google Cast')}
            </MenuItem>
          ) : null}
        </Menu>
        {!google.supported && google.error?.includes('HTTPS') ? (
          <p className="player-cast-menu__hint">{tr('Google Cast vajab HTTPS-ühendust. Kohalikus võrgus kasuta allolevat teleriühendust.')}</p>
        ) : null}
      </div>

      <div className="player-cast-menu__section player-cast-menu__section--network">
        <span className="player-cast-menu__section-label">{tr('Sama võrgu telerid')}</span>
        {discovering && devices.length === 0 ? (
          <div className="player-cast-menu__loading" role="status">
            <Spinner label={tr('Otsin samast võrgust seadmeid')} />
            <span>{tr('Otsin samast võrgust seadmeid…')}</span>
          </div>
        ) : devices.length > 0 ? (
          <Menu className="player-cast-menu__devices" role="presentation">
            {devices.map((device) => (
              <MenuItem
                key={device.id}
                icon="tv"
                selected={dlnaSession?.device.id === device.id}
                disabled={connecting}
                description={deviceDescription(device)}
                onClick={() => onDLNA(device)}
              >
                {device.name}
              </MenuItem>
            ))}
          </Menu>
        ) : (
          <div className="player-cast-menu__empty">
            <Icon name="tv" size={21} />
            <span>{tr('Ühtegi DLNA-telerit ei leitud.')}</span>
            <small>{tr('Veendu, et teler ja KoduMedia server on samas võrgus.')}</small>
          </div>
        )}
      </div>

      {error ? <p className="player-cast-menu__error" role="alert"><Icon name="alert" size={15} />{error}</p> : null}

      <div className="player-cast-menu__footer">
        <Button compact variant="ghost" icon="refresh" onClick={onRefresh} disabled={discovering || connecting}>
          {discovering ? tr('Otsin…') : tr('Otsi uuesti')}
        </Button>
        {connecting ? <Spinner label={tr('Ühendan seadmega')} /> : null}
      </div>
    </Popover>
  )
}
