import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './app/App'
import { AppErrorBoundary } from './app/AppErrorBoundary'
import { ThemeProvider } from './features/theme/ThemeProvider'
import './i18n/i18n'
import './design/tokens.css'
import './design/motion.css'
import './design/global.css'
import './design/primitives.css'
import './design/performance.css'
import './design/interaction.css'

const root = document.getElementById('root')
if (!root) throw new Error('Rakenduse juurelement puudub.')

createRoot(root).render(
  <StrictMode>
    <AppErrorBoundary>
      <ThemeProvider>
        <App />
      </ThemeProvider>
    </AppErrorBoundary>
  </StrictMode>,
)
