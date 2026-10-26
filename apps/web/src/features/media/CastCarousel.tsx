import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import type { PersonCredit } from '../../api/types'
import { tr, trExternal } from '../../i18n/i18n'

interface DragState {
  pointerId: number
  startX: number
  startScrollLeft: number
}

interface ScrollLayout {
  scrollable: boolean
  thumbWidth: number
}

export function CastCarousel({ people }: { people: PersonCredit[] }) {
  const scrollerRef = useRef<HTMLDivElement>(null)
  const thumbRef = useRef<HTMLSpanElement>(null)
  const dragRef = useRef<DragState | null>(null)
  const thumbTravelRef = useRef({ availableTrack: 0, maxScroll: 1 })
  const [layout, setLayout] = useState<ScrollLayout>({ scrollable: false, thumbWidth: 0 })

  const syncThumbPosition = useCallback(() => {
    const scroller = scrollerRef.current
    const thumb = thumbRef.current
    if (!scroller || !thumb) return
    const { availableTrack, maxScroll } = thumbTravelRef.current
    const offset = availableTrack * (scroller.scrollLeft / maxScroll)
    thumb.style.transform = `translateX(${offset}px)`
  }, [])

  const updateLayout = useCallback(() => {
    const scroller = scrollerRef.current
    if (!scroller) return
    const clientWidth = scroller.clientWidth
    const scrollWidth = scroller.scrollWidth
    const scrollable = scrollWidth > clientWidth + 1
    const thumbWidth = scrollable ? Math.max(48, clientWidth * (clientWidth / scrollWidth)) : clientWidth
    thumbTravelRef.current = {
      availableTrack: Math.max(0, clientWidth - thumbWidth),
      maxScroll: Math.max(1, scrollWidth - clientWidth),
    }
    setLayout((current) => (
      current.scrollable === scrollable && current.thumbWidth === thumbWidth
        ? current
        : { scrollable, thumbWidth }
    ))
    syncThumbPosition()
  }, [syncThumbPosition])

  useEffect(() => {
    syncThumbPosition()
  }, [layout, syncThumbPosition])

  useEffect(() => {
    updateLayout()
    const scroller = scrollerRef.current
    if (!scroller || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(updateLayout)
    observer.observe(scroller)
    return () => observer.disconnect()
  }, [people.length, updateLayout])

  const finishDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    const scroller = scrollerRef.current
    const drag = dragRef.current
    if (!scroller || !drag || drag.pointerId !== event.pointerId) return
    if (scroller.hasPointerCapture(event.pointerId)) scroller.releasePointerCapture(event.pointerId)
    scroller.classList.remove('cast-list--dragging')
    dragRef.current = null
  }

  return (
    <div className="cast-carousel">
      <div
        ref={scrollerRef}
        className="cast-list cast-list--cinematic"
        aria-label={tr('Näitlejad ja tegijad')}
        onScroll={syncThumbPosition}
        onPointerDown={(event) => {
          const scroller = scrollerRef.current
          if (!scroller || event.pointerType !== 'mouse' || event.button !== 0 || scroller.scrollWidth <= scroller.clientWidth) return
          dragRef.current = { pointerId: event.pointerId, startX: event.clientX, startScrollLeft: scroller.scrollLeft }
          scroller.setPointerCapture(event.pointerId)
          scroller.classList.add('cast-list--dragging')
        }}
        onPointerMove={(event) => {
          const scroller = scrollerRef.current
          const drag = dragRef.current
          if (!scroller || !drag || drag.pointerId !== event.pointerId) return
          scroller.scrollLeft = drag.startScrollLeft - (event.clientX - drag.startX)
        }}
        onPointerUp={finishDrag}
        onPointerCancel={finishDrag}
        onLostPointerCapture={(event) => {
          if (dragRef.current?.pointerId === event.pointerId) {
            scrollerRef.current?.classList.remove('cast-list--dragging')
            dragRef.current = null
          }
        }}
      >
        {people.slice(0, 12).map((person) => (
          <div className="cast-card cast-card--cinematic" key={`${person.name}-${person.character ?? ''}`}>
            {person.imageUrl ? <img src={person.imageUrl} alt="" loading="lazy" decoding="async" referrerPolicy="no-referrer" draggable={false} /> : <span className="cast-card__avatar">{person.name.slice(0, 1)}</span>}
            <div><strong>{person.name}</strong><small>{trExternal(person.character || person.role) || tr('Näitleja')}</small></div>
          </div>
        ))}
      </div>
      {layout.scrollable ? (
        <div className="cast-carousel__rail" aria-hidden="true">
          <span ref={thumbRef} className="cast-carousel__thumb" style={{ width: `${layout.thumbWidth}px` }} />
        </div>
      ) : null}
    </div>
  )
}
