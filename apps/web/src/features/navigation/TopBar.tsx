import { tr } from '../../i18n/i18n'
import { useTheme } from '../theme/ThemeProvider'
import { Button } from '../../shared/components/Button'

export function TopBar({ title, subtitle }: { title: string; subtitle?: string }) {
  const { theme, toggleTheme } = useTheme()
  return (
    <header className="topbar">
      <div>
        <h1>{title}</h1>
        {subtitle ? <p>{subtitle}</p> : null}
      </div>
      <Button
        compact
        variant="ghost"
        icon={theme === 'dark' ? 'sun' : 'moon'}
        onClick={toggleTheme}
        aria-label={theme === 'dark' ? tr('Kasuta heledat teemat') : tr('Kasuta tumedat teemat')}
        title={theme === 'dark' ? tr('Hele teema') : tr('Tume teema')}
      />
    </header>
  )
}
