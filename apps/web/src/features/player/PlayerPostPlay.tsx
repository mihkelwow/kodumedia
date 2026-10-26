import type { MediaItem, PostPlay, PostPlayRecommendation } from '../../api/types'
import { tr, trExternal } from '../../i18n/i18n'
import { Icon } from '../../shared/components/Icon'

function episodeCode(item: MediaItem): string {
  if (item.type !== 'episode') return ''
  return `S${String(item.season ?? 0).padStart(2, '0')}E${String(item.episode ?? 0).padStart(2, '0')}`
}

function recommendationTarget(recommendation: PostPlayRecommendation): MediaItem {
  return recommendation.playTarget ?? recommendation.item
}

interface PlayerPostPlayProps {
  nextEpisode?: MediaItem
  showNextEpisodePreview: boolean
  playbackEnded: boolean
  autoplayCancelled: boolean
  remainingSeconds: number
  nextEpisodeProgress: number
  showRecommendations: boolean
  postPlay: PostPlay | null
  onPlayNext: (item: MediaItem) => void
  onCancelAutoplay: () => void
  onPlayRecommendation: (item: MediaItem) => void
  onClose: () => void
}

/** Post-play UI isolated from playback session state and media transport logic. */
export function PlayerPostPlay({
  nextEpisode,
  showNextEpisodePreview,
  playbackEnded,
  autoplayCancelled,
  remainingSeconds,
  nextEpisodeProgress,
  showRecommendations,
  postPlay,
  onPlayNext,
  onCancelAutoplay,
  onPlayRecommendation,
  onClose,
}: PlayerPostPlayProps) {
  return (
    <>
      {showNextEpisodePreview && nextEpisode ? (
        <aside className={`player-next-up ${playbackEnded ? 'player-next-up--ended' : ''}`} aria-label={tr('Järgmine osa')}>
          <div className="player-next-up__art">
            {nextEpisode.metadata.backdropUrl || nextEpisode.metadata.posterUrl ? (
              <img src={nextEpisode.metadata.backdropUrl || nextEpisode.metadata.posterUrl} alt="" decoding="async" />
            ) : null}
            <span className="player-next-up__eyebrow">{tr('Järgmine osa')}</span>
          </div>
          <div className="player-next-up__body">
            <div className="player-next-up__heading">
              <div>
                <small>{episodeCode(nextEpisode)}</small>
                <strong>{nextEpisode.episodeTitle || nextEpisode.title}</strong>
              </div>
              {!autoplayCancelled ? (
                <span>{playbackEnded ? tr('Algab kohe') : tr('Algab {{seconds}} s pärast', { seconds: Math.max(1, Math.ceil(remainingSeconds)) })}</span>
              ) : (
                <span>{tr('Automaatne esitus tühistatud')}</span>
              )}
            </div>
            <div className="player-next-up__timeline" aria-hidden="true">
              <span style={{ transform: `scaleX(${Math.min(1, Math.max(0, nextEpisodeProgress / 100))})` }} />
            </div>
            <div className="player-next-up__actions">
              <button type="button" className="player-next-up__play" onClick={() => onPlayNext(nextEpisode)}>
                <Icon name="play" size={16} /> {tr('Esita kohe')}
              </button>
              {!autoplayCancelled && !playbackEnded ? (
                <button type="button" className="player-next-up__cancel" onClick={onCancelAutoplay}>
                  {tr('Ära esita automaatselt')}
                </button>
              ) : null}
            </div>
          </div>
        </aside>
      ) : null}

      {showRecommendations ? (
        <section className="player-postplay" aria-label={tr('Mida järgmiseks vaadata')}>
          <div className="player-postplay__heading">
            <span>{tr('Vaatamine lõpetatud')}</span>
            <h2>{tr('Mida järgmiseks vaadata?')}</h2>
            <p>{tr('Valisime vaatamata sisu samast või sarnasest žanrist.')}</p>
          </div>
          {postPlay?.recommendations.length ? (
            <div className="player-postplay__grid">
              {postPlay.recommendations.slice(0, 3).map((recommendation) => {
                const target = recommendationTarget(recommendation)
                const art = recommendation.item.metadata.backdropUrl || recommendation.item.metadata.posterUrl
                return (
                  <button
                    type="button"
                    className="player-postplay-card"
                    key={recommendation.item.id}
                    onClick={() => onPlayRecommendation(target)}
                  >
                    <span className="player-postplay-card__art">
                      {art ? <img src={art} alt="" loading="lazy" decoding="async" /> : null}
                      <span className="player-postplay-card__play"><Icon name="play" size={18} /></span>
                    </span>
                    <span className="player-postplay-card__copy">
                      <strong>{recommendation.item.title}</strong>
                      <small>
                        {target.type === 'episode' ? `${episodeCode(target)} · ` : ''}
                        {trExternal(recommendation.reason) || tr('Vaatamata soovitus')}
                      </small>
                    </span>
                  </button>
                )
              })}
            </div>
          ) : (
            <div className="player-postplay__empty">{tr('Rohkem vaatamata soovitusi praegu ei leitud.')}</div>
          )}
          <button type="button" className="player-postplay__close" onClick={onClose}>{tr('Tagasi KODU avalehele')}</button>
        </section>
      ) : null}
    </>
  )
}
