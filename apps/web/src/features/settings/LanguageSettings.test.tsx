import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { resetLocaleForTests } from '../../i18n/i18n'
import { LanguageSettings } from './LanguageSettings'

describe('LanguageSettings', () => {
  afterEach(() => {
    cleanup()
    window.localStorage.clear()
    resetLocaleForTests()
  })

  it('changes the interface language immediately and stores it per browser', async () => {
    const user = userEvent.setup()
    resetLocaleForTests('et-EE')
    render(<LanguageSettings />)

    const select = screen.getByRole('combobox', { name: 'Kasutajaliidese keel' })
    await user.selectOptions(select, 'en-US')

    expect(screen.getByRole('heading', { name: 'Application language' })).toBeTruthy()
    expect((screen.getByRole('combobox', { name: 'Interface language' }) as HTMLSelectElement).value).toBe('en-US')
    expect(window.localStorage.getItem('kodumedia:ui-language')).toBe('en-US')
    expect(document.documentElement.lang).toBe('en-US')
  })
})
