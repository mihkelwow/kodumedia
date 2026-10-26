import { tr } from '../i18n/i18n'
import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from '../shared/components/Button'
import { Icon } from '../shared/components/Icon'

interface AppErrorBoundaryProps {
  children: ReactNode
}

interface AppErrorBoundaryState {
  error: Error | null
}

export class AppErrorBoundary extends Component<AppErrorBoundaryProps, AppErrorBoundaryState> {
  state: AppErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): AppErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error(tr('KoduMedia UI renderdamine ebaõnnestus.'), error, info)
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children

    return (
      <main className="fatal-error" role="alert" aria-live="assertive">
        <div className="fatal-error__card">
          <span className="fatal-error__icon"><Icon name="alert" size={30} /></span>
          <span className="eyebrow">{tr('Rakenduse viga')}</span>
          <h1>{tr('Vaadet ei saanud avada')}</h1>
          <p>{tr('Veebiliides sattus ootamatusse olekusse. Server ja sinu meediafailid ei ole selle tõttu kahjustatud.')}</p>
          <code>{error.message || tr('Tundmatu kasutajaliidese viga')}</code>
          <div className="fatal-error__actions">
            <Button variant="primary" icon="refresh" onClick={() => window.location.reload()}>{tr('Laadi uuesti')}</Button>
            <Button variant="secondary" icon="home" onClick={() => window.location.assign('/')}>{tr('Avalehele')}</Button>
          </div>
        </div>
      </main>
    )
  }
}
