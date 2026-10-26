import type { ReactNode } from 'react'
import { Icon, type IconName } from '../../shared/components/Icon'
import { IconButton } from '../../shared/components/IconButton'
import { Menu, MenuItem } from '../../shared/components/Menu'
import { Popover } from '../../shared/components/Popover'

export interface PlayerTrackMenuOption {
  id: string
  title: string
  description?: string
  selected?: boolean
}

export interface PlayerTrackMenuAction {
  title: string
  description?: string
  icon?: IconName
  onClick: () => void
}

interface PlayerTrackMenuProps {
  icon: IconName
  ariaLabel: string
  title: string
  description?: string
  open: boolean
  active?: boolean
  onOpenChange: (open: boolean) => void
  options: PlayerTrackMenuOption[]
  onSelect: (id: string) => void
  footerAction?: PlayerTrackMenuAction
  extraContent?: ReactNode
}

/** Player menu using the same anchored dismissal and keyboard model as the rest of the app. */
export function PlayerTrackMenu({
  icon,
  ariaLabel,
  title,
  description,
  open,
  active = false,
  onOpenChange,
  options,
  onSelect,
  footerAction,
  extraContent,
}: PlayerTrackMenuProps) {
  return (
    <Popover
      open={open}
      onOpenChange={onOpenChange}
      placement="top-end"
      role="menu"
      ariaLabel={title}
      autoFocus
      rootClassName="player-menu"
      className="player-menu-panel"
      trigger={(
        <IconButton
          icon={icon}
          label={ariaLabel}
          className={`player-control-button player-control-button--menu ${open ? 'player-control-button--menu-open' : ''} ${active ? 'player-control-button--menu-active' : ''}`}
          aria-haspopup="menu"
        />
      )}
    >
      <div className="player-menu-panel__header">
        <strong>{title}</strong>
        {description ? <small>{description}</small> : null}
      </div>
      {extraContent ? <div className="player-menu-panel__content">{extraContent}</div> : null}
      {options.length > 0 ? (
        <Menu className="player-menu-panel__options" role="presentation">
          {options.map((option) => (
            <MenuItem
              key={option.id || 'off'}
              className="player-menu-option"
              description={option.description}
              selected={option.selected}
              role="menuitemradio"
              aria-checked={Boolean(option.selected)}
              onClick={() => onSelect(option.id)}
            >
              {option.title}
            </MenuItem>
          ))}
        </Menu>
      ) : null}
      {footerAction ? (
        <div className="player-menu-panel__footer">
          <Menu role="presentation">
            <MenuItem
              className="player-menu-option player-menu-option--action"
              description={footerAction.description}
              onClick={footerAction.onClick}
            >
              <span className="player-menu-option__action-title">
                {footerAction.title}
                {footerAction.icon ? <Icon name={footerAction.icon} size={16} /> : null}
              </span>
            </MenuItem>
          </Menu>
        </div>
      ) : null}
    </Popover>
  )
}
