import type { SubtitleSearchResult } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { Icon } from '../../shared/components/Icon'
import { IconButton } from '../../shared/components/IconButton'
import { Spinner } from '../../shared/components/Spinner'

interface PlayerSubtitleSearchPanelProps {
  results: SubtitleSearchResult[]
  searching: boolean
  error: string
  downloadingFileId: number | null
  onClose: () => void
  onDownload: (result: SubtitleSearchResult) => void
  matchLabel: (matchType?: string) => string
}

/** OpenSubtitles result panel separated from the playback surface and transport controls. */
export function PlayerSubtitleSearchPanel({
  results,
  searching,
  error,
  downloadingFileId,
  onClose,
  onDownload,
  matchLabel,
}: PlayerSubtitleSearchPanelProps) {
  return (
    <aside className="subtitle-search-panel" aria-label={tr('OpenSubtitles otsing')}>
      <header>
        <div>
          <strong>OpenSubtitles</strong>
          <small>{tr('Vali vaste ja KoduMedia salvestab selle serverisse.')}</small>
        </div>
        <IconButton icon="close" label={tr('Sulge')} className="player-control-button" onClick={onClose} />
      </header>
      {searching ? (
        <div className="subtitle-search-state">
          <Spinner label={tr('Otsin subtiitreid')} />
          <span>{tr('Otsin faili räsi, IMDb/TMDb ID ja ametliku pealkirja järgi…')}</span>
        </div>
      ) : null}
      {error ? (
        <div className="subtitle-search-error">
          <Icon name="alert" />
          <span>{error}</span>
        </div>
      ) : null}
      {!searching && !error && results.length === 0 ? (
        <p className="subtitle-search-empty">{tr('Sobivaid tulemusi ei leitud.')}</p>
      ) : null}
      <div className="subtitle-search-results">
        {results.map((result) => (
          <button
            key={result.id}
            type="button"
            disabled={downloadingFileId !== null}
            onClick={() => onDownload(result)}
          >
            <span className="subtitle-result__language">{result.language?.toUpperCase() || 'UND'}</span>
            <span className="subtitle-result__copy">
              <strong>{result.release || result.fileName}</strong>
              <small>
                {[
                  matchLabel(result.matchType),
                  result.trusted ? tr('kontrollitud') : '',
                  result.hearingImpaired ? 'SDH' : '',
                  result.rating ? tr('hinne {{rating}}', { rating: result.rating.toFixed(1) }) : '',
                  result.downloads ? tr('{{count}} laadimist', { count: result.downloads }) : '',
                ].filter(Boolean).join(' · ')}
              </small>
            </span>
            {downloadingFileId === result.fileId ? <Spinner label={tr('Laadin')} /> : <Icon name="download" />}
          </button>
        ))}
      </div>
    </aside>
  )
}
