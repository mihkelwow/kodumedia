import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import useEmblaCarousel from 'embla-carousel-react'
import type { MediaItem } from '../../api/types'
import { tr } from '../../i18n/i18n'
import { Icon } from '../../shared/components/Icon'
import { formatDate, formatDuration } from '../../shared/utils/format'

interface EpisodeCarouselProps {
  episodes: MediaItem[]
  label: string
  onOpenEpisode: (episode: MediaItem) => void
}

interface CarouselState {
  canScroll: boolean
  canScrollPrev: boolean
  canScrollNext: boolean
}

export function episodeCode(item: MediaItem): string {
  if (!item.season || !item.episode) return item.season === 0 && item.episode ? `E${String(item.episode).padStart(2, '0')}` : ''
  return `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}`
}

export function EpisodeCarousel({ episodes, label, onOpenEpisode }: EpisodeCarouselProps) {
  const progressRef = useRef<HTMLSpanElement>(null)
  const cardRefs = useRef<Array<HTMLButtonElement | null>>([])
  const signature = useMemo(() => episodes.map((episode) => episode.id).join('|'), [episodes])
  const [viewportRef, emblaApi] = useEmblaCarousel({
    align: 'start',
    containScroll: 'trimSnaps',
    dragFree: false,
    duration: 25,
    skipSnaps: false,
    watchDrag: episodes.length > 1,
    breakpoints: {
      '(hover: hover) and (pointer: fine)': {
        dragFree: true,
        skipSnaps: true,
      },
    },
  })
  const [carouselState, setCarouselState] = useState<CarouselState>({
    canScroll: false,
    canScrollPrev: false,
    canScrollNext: false,
  })

  const syncProgress = useCallback(() => {
    if (!emblaApi || !progressRef.current) return
    const progress = Math.min(1, Math.max(0, emblaApi.scrollProgress()))
    const visibleProgress = carouselState.canScroll ? 0.18 + progress * 0.82 : 1
    progressRef.current.style.transform = `scaleX(${visibleProgress})`
  }, [carouselState.canScroll, emblaApi])

  const syncCarouselState = useCallback(() => {
    if (!emblaApi) return
    const canScroll = emblaApi.scrollSnapList().length > 1
    setCarouselState({
      canScroll,
      canScrollPrev: emblaApi.canScrollPrev(),
      canScrollNext: emblaApi.canScrollNext(),
    })
  }, [emblaApi])

  useEffect(() => {
    if (!emblaApi) return
    syncCarouselState()
    syncProgress()
    emblaApi.on('select', syncCarouselState)
    emblaApi.on('reInit', syncCarouselState)
    emblaApi.on('scroll', syncProgress)
    emblaApi.on('reInit', syncProgress)
    return () => {
      emblaApi.off('select', syncCarouselState)
      emblaApi.off('reInit', syncCarouselState)
      emblaApi.off('scroll', syncProgress)
      emblaApi.off('reInit', syncProgress)
    }
  }, [emblaApi, syncCarouselState, syncProgress])

  useEffect(() => {
    if (!emblaApi) return
    emblaApi.reInit()
    emblaApi.scrollTo(0, true)
    syncCarouselState()
    syncProgress()
  }, [emblaApi, signature, syncCarouselState, syncProgress])

  useEffect(() => {
    syncProgress()
  }, [carouselState, syncProgress])

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!emblaApi || !carouselState.canScroll || !(event.target instanceof HTMLElement)) return
    const activeCard = event.target.closest<HTMLButtonElement>('.episode-card')
    const currentIndex = activeCard ? cardRefs.current.indexOf(activeCard) : -1
    if (currentIndex < 0) return

    let nextIndex = currentIndex
    if (event.key === 'ArrowLeft') nextIndex = Math.max(0, currentIndex - 1)
    else if (event.key === 'ArrowRight') nextIndex = Math.min(episodes.length - 1, currentIndex + 1)
    else if (event.key === 'Home') nextIndex = 0
    else if (event.key === 'End') nextIndex = episodes.length - 1
    else return

    if (nextIndex === currentIndex) return
    event.preventDefault()
    cardRefs.current[nextIndex]?.focus({ preventScroll: true })
    emblaApi.scrollTo(nextIndex)
  }

  return (
    <div className="episode-carousel" role="region" aria-label={label}>
      <div className="episode-carousel__toolbar">
        <span>{tr('Episoodid')}</span>
        {carouselState.canScroll ? (
          <div className="episode-carousel__controls">
            <button
              type="button"
              className="episode-carousel__control episode-carousel__control--previous"
              aria-label={tr('Eelmine osa')}
              disabled={!carouselState.canScrollPrev}
              onClick={() => emblaApi?.scrollPrev()}
              data-focusable="true"
            >
              <Icon name="chevronRight" size={18} />
            </button>
            <button
              type="button"
              className="episode-carousel__control"
              aria-label={tr('Järgmine osa')}
              disabled={!carouselState.canScrollNext}
              onClick={() => emblaApi?.scrollNext()}
              data-focusable="true"
            >
              <Icon name="chevronRight" size={18} />
            </button>
          </div>
        ) : null}
      </div>

      <div className="episode-carousel__viewport" ref={viewportRef} onKeyDown={handleKeyDown}>
        <div className="episode-carousel__track" role="list">
          {episodes.map((episode, index) => {
            const code = episodeCode(episode)
            return (
              <div className="episode-carousel__slide" role="listitem" key={episode.id}>
                <button
                  ref={(node) => { cardRefs.current[index] = node }}
                  type="button"
                  className="episode-card"
                  aria-label={`${code} ${episode.title}`.trim()}
                  onClick={() => onOpenEpisode(episode)}
                  data-focusable="true"
                >
                  <span className="episode-card__art">
                    {episode.metadata.backdropUrl ? <img src={episode.metadata.backdropUrl} alt="" loading="lazy" decoding="async" referrerPolicy="no-referrer" draggable={false} /> : <span><Icon name="play" size={24} /></span>}
                    {episode.watch && episode.watch.progress > 0 ? <span className="episode-card__progress"><span style={{ transform: `scaleX(${Math.min(1, Math.max(0, episode.watch.progress))})` }} /></span> : null}
                  </span>
                  <span className="episode-card__body">
                    {code ? <small className="episode-card__code">{code}</small> : null}
                    <strong className="episode-card__title">{episode.title}</strong>
                    <span className="episode-card__summary">{episode.metadata.summary || tr('Kirjeldus puudub.')}</span>
                    <span className="episode-card__filename" title={episode.file.relativePath}>{episode.file.name}</span>
                    <span className="episode-card__meta">
                      {episode.releaseDate ? <small>{formatDate(episode.releaseDate)}</small> : null}
                      {episode.technical.durationMs ? <small>{formatDuration(episode.technical.durationMs)}</small> : null}
                      {episode.watch && !episode.watch.completed && episode.watch.progress >= 0.01 ? <small>{tr('Vaadatud {{percent}}%', { percent: Math.round(episode.watch.progress * 100) })}</small> : null}
                    </span>
                  </span>
                </button>
              </div>
            )
          })}
        </div>
      </div>

      {carouselState.canScroll ? (
        <div className="episode-carousel__rail" aria-hidden="true">
          <span ref={progressRef} className="episode-carousel__progress" />
        </div>
      ) : null}
    </div>
  )
}
