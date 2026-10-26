import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { resetLocaleForTests } from '../i18n/i18n'
import { SettingsPage } from './SettingsPage'

vi.mock('../features/settings/SystemStatus', () => ({
  SystemStatus: () => <div data-testid="settings-content" />,
}))

describe('SettingsPage', () => {
  beforeEach(() => {
    resetLocaleForTests('et-EE')
    window.history.replaceState({}, '', '/settings')
  })

  it('renders the detailed desktop navigation and keeps the compact mobile picker in sync', async () => {
    const user = userEvent.setup()
    render(<SettingsPage libraries={[]} onLibrariesChanged={async () => {}} />)

    expect(document.querySelectorAll('.settings-navigation__desktop .settings-navigation__link')).toHaveLength(7)
    expect(screen.getByRole('link', { name: /^Server ja töövalmidus/ }).getAttribute('aria-current')).toBe('location')
    expect(screen.getByText('Olek, tööriistad ja turvalisus')).toBeTruthy()
    expect(screen.getByText('Kaustad, TMDb, IMDb ja muusika')).toBeTruthy()
    expect(screen.getByText('Väljalasked ja tehniline olek')).toBeTruthy()
    expect(document.querySelectorAll('.settings-navigation__chevron')).toHaveLength(7)

    const sectionPicker = screen.getByLabelText('Seadete jaotis')
    expect(sectionPicker.closest('.settings-navigation__mobile-control')).toBeTruthy()
    expect(document.querySelector('.settings-navigation__mobile-control .select-control__chevron')).toBeTruthy()
    await user.selectOptions(sectionPicker, '#settings-language')

    expect(window.location.hash).toBe('#settings-language')
    expect(screen.getByRole('link', { name: /^Välimus ja keel/ }).getAttribute('aria-current')).toBe('location')
  })
})
