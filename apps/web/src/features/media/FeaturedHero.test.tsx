import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { MediaItem } from '../../api/types'
import { FeaturedHero } from './FeaturedHero'

const carouselMock = vi.hoisted(() => {
  let selectedIndex = 0
  const listeners = new Map<string, Set<() => void>>()
  const emit = (event: string) => {
    for (const listener of listeners.get(event) ?? []) listener()
  }
  const api = {
    selectedScrollSnap: vi.fn(() => selectedIndex),
    scrollTo: vi.fn((index: number) => {
      selectedIndex = index
      emit('select')
    }),
    on: vi.fn((event: string, listener: () => void) => {
      const eventListeners = listeners.get(event) ?? new Set()
      eventListeners.add(listener)
      listeners.set(event, eventListeners)
    }),
    off: vi.fn((event: string, listener: () => void) => {
      listeners.get(event)?.delete(listener)
    }),
  }
  const viewportRef = vi.fn()
  const hook = vi.fn(() => [viewportRef, api])
  return {
    api,
    hook,
    select(index: number) {
      selectedIndex = index
      emit('select')
    },
    reset() {
      selectedIndex = 0
      listeners.clear()
      viewportRef.mockClear()
      hook.mockClear()
      for (const method of Object.values(api)) method.mockClear()
    },
  }
})

vi.mock('embla-carousel-react', () => ({ default: carouselMock.hook }))

const PLAYER_STATE = {
  UNSTARTED: -1,
  ENDED: 0,
  PLAYING: 1,
  PAUSED: 2,
  BUFFERING: 3,
  CUED: 5,
}

interface MockPlayerOptions {
  videoId: string
  playerVars: Record<string, string | number>
  events: {
    onReady: (event: { target: MockPlayer }) => void
    onStateChange: (event: { target: MockPlayer; data: number }) => void
    onError: (event: { target: MockPlayer; data: number }) => void
  }
}

const players: MockPlayer[] = []

class MockPlayer {
  readonly options: MockPlayerOptions
  state = PLAYER_STATE.UNSTARTED
  destroyed = false
  muted = true

  constructor(_element: HTMLElement, options: MockPlayerOptions) {
    this.options = options
    players.push(this)
  }

  destroy() { this.destroyed = true }
  mute() { this.muted = true }
  unMute() { this.muted = false }
  playVideo() { this.state = PLAYER_STATE.PLAYING }
  pauseVideo() { this.state = PLAYER_STATE.PAUSED }
  getPlayerState() { return this.state }

  ready() {
    this.options.events.onReady({ target: this })
  }

  emit(state: number) {
    this.state = state
    this.options.events.onStateChange({ target: this, data: state })
  }

  error(code = 100) {
    this.options.events.onError({ target: this, data: code })
  }
}

function featuredItem(index: number): MediaItem {
  return {
    id: `featured-${index}`,
    libraryId: 'movies',
    type: 'movie',
    title: `Featured ${index}`,
    sortTitle: `Featured ${index}`,
    year: 2020 + index,
    releaseDate: `202${index}-01-01`,
    file: {
      relativePath: `featured-${index}.mkv`,
      name: `featured-${index}.mkv`,
      extension: '.mkv',
      sizeBytes: 1,
      modifiedAt: '2026-07-23T00:00:00Z',
    },
    technical: {},
    externalIds: {},
    metadata: {
      summary: `Summary ${index}`,
      backdropUrl: `/backdrop-${index}.jpg`,
      trailerYoutubeKey: `trailer-${index}`,
    },
    addedAt: '2026-07-23T00:00:00Z',
    updatedAt: '2026-07-23T00:00:00Z',
  }
}

function touchList(identifier: number, clientX: number, clientY: number) {
  const touch = { identifier, clientX, clientY }
  return Object.assign([touch], { item: (index: number) => index === 0 ? touch : null })
}

const emptyTouchList = Object.assign([], { item: () => null })

async function startNextPlayer() {
  await act(async () => {
    vi.advanceTimersByTime(850)
    await Promise.resolve()
  })
  const current = players.at(-1)
  if (!current) throw new Error('Expected a YouTube player')
  act(() => {
    current.ready()
    current.emit(PLAYER_STATE.PLAYING)
  })
  return current
}

beforeEach(() => {
  carouselMock.reset()
  players.length = 0
  window.localStorage.clear()
  vi.useFakeTimers()
  window.YT = {
    Player: MockPlayer as unknown as typeof window.YT.Player,
    PlayerState: PLAYER_STATE,
  }
})

afterEach(() => {
  vi.useRealTimers()
  delete window.YT
})

describe('FeaturedHero trailer autoplay sequence', () => {
  it('keeps the protected compact slider dots as the buttons themselves', () => {
    render(<FeaturedHero items={[featuredItem(1), featuredItem(2)]} onOpen={vi.fn()} />)

    const dots = screen.getAllByRole('tab')
    expect(dots).toHaveLength(2)
    expect(dots.every((dot) => dot.classList.contains('featured-hero__dot'))).toBe(true)
    expect(dots.every((dot) => dot.childElementCount === 0)).toBe(true)
    expect(document.querySelector('.featured-hero__dot-indicator')).toBeNull()
  })

  it('uses the shared pill play action', () => {
    render(<FeaturedHero items={[featuredItem(1)]} onOpen={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Vaata kohe' }).classList.contains('media-play-button')).toBe(true)
  })

  it('uses a full-width Embla track so touch drag moves adjacent slides with the finger', () => {
    render(<FeaturedHero items={[featuredItem(1), featuredItem(2)]} onOpen={vi.fn()} />)

    const hero = screen.getByRole('region')
    expect(hero.querySelector('.featured-hero__viewport')).toBeTruthy()
    expect(hero.querySelector('.featured-hero__slides')?.children).toHaveLength(2)
    expect(carouselMock.hook).toHaveBeenCalledWith(expect.objectContaining({
      duration: 26,
      loop: true,
      watchDrag: true,
    }))
  })

  it('updates the active hero when the touch carousel snaps to another slide', () => {
    render(<FeaturedHero items={[featuredItem(1), featuredItem(2)]} onOpen={vi.fn()} />)

    const hero = screen.getByRole('region')
    act(() => {
      carouselMock.select(1)
    })
    expect(hero.getAttribute('aria-label')).toContain('Featured 2')
    expect(hero.querySelectorAll('.featured-hero__slide[aria-hidden="false"]')).toHaveLength(1)
  })

  it('commits a modest horizontal touch swipe to the adjacent slide', () => {
    render(<FeaturedHero items={[featuredItem(1), featuredItem(2)]} onOpen={vi.fn()} />)

    const hero = screen.getByRole('region')
    Object.defineProperty(hero, 'clientWidth', { configurable: true, value: 390 })
    fireEvent.touchStart(hero, { touches: touchList(4, 300, 220), changedTouches: touchList(4, 300, 220) })
    fireEvent.touchMove(hero, { touches: touchList(4, 210, 224), changedTouches: touchList(4, 210, 224) })
    fireEvent.touchEnd(hero, { touches: emptyTouchList, changedTouches: touchList(4, 210, 224) })

    expect(carouselMock.api.scrollTo).toHaveBeenCalledWith(1)
  })

  it('does not turn a vertical page scroll into a hero slide change', () => {
    render(<FeaturedHero items={[featuredItem(1), featuredItem(2)]} onOpen={vi.fn()} />)

    const hero = screen.getByRole('region')
    Object.defineProperty(hero, 'clientWidth', { configurable: true, value: 390 })
    fireEvent.touchStart(hero, { touches: touchList(5, 250, 180), changedTouches: touchList(5, 250, 180) })
    fireEvent.touchMove(hero, { touches: touchList(5, 242, 310), changedTouches: touchList(5, 242, 310) })
    fireEvent.touchEnd(hero, { touches: emptyTouchList, changedTouches: touchList(5, 242, 310) })

    expect(carouselMock.api.scrollTo).not.toHaveBeenCalled()
  })

  it('keeps the artwork visible until playback starts and then exposes the cinematic trailer layer', async () => {
    render(<FeaturedHero items={[featuredItem(1)]} onOpen={vi.fn()} />)

    const hero = screen.getByRole('region')
    const backdrop = hero.querySelector('.featured-hero__backdrop')
    expect(backdrop).toBeTruthy()
    expect(hero.classList.contains('featured-hero--playing')).toBe(false)

    await act(async () => {
      vi.advanceTimersByTime(850)
      await Promise.resolve()
    })
    const current = players.at(-1)
    if (!current) throw new Error('Expected a YouTube player')

    act(() => {
      current.ready()
      current.emit(PLAYER_STATE.BUFFERING)
    })
    expect(hero.classList.contains('featured-hero--playing')).toBe(false)
    expect(hero.querySelector('.featured-hero__backdrop')).toBe(backdrop)

    act(() => current.emit(PLAYER_STATE.PLAYING))
    expect(hero.classList.contains('featured-hero--playing')).toBe(true)
    expect(hero.querySelector('.featured-hero__backdrop')).toBe(backdrop)
  })

  it('advances after each trailer, stops on the final poster, and resumes after manual navigation', async () => {
    render(<FeaturedHero items={[featuredItem(1), featuredItem(2)]} onOpen={vi.fn()} />)

    const firstPlayer = await startNextPlayer()
    expect(firstPlayer.options.playerVars.loop).toBe(0)
    expect(firstPlayer.options.playerVars.playlist).toBeUndefined()
    expect(firstPlayer.muted).toBe(true)

    fireEvent.focus(screen.getByRole('button', { name: 'Vaata kohe' }))
    expect(firstPlayer.muted).toBe(true)
    fireEvent.click(screen.getByRole('button', { name: 'Lülita heli sisse' }))
    expect(firstPlayer.muted).toBe(false)

    act(() => firstPlayer.emit(PLAYER_STATE.ENDED))
    expect(screen.getByRole('region').getAttribute('aria-label')).toContain('Featured 2')

    const secondPlayer = await startNextPlayer()
    act(() => secondPlayer.emit(PLAYER_STATE.ENDED))

    const finalHero = screen.getByRole('region')
    expect(finalHero.getAttribute('aria-label')).toContain('Featured 2')
    expect(finalHero.classList.contains('featured-hero--playing')).toBe(false)
    expect(finalHero.querySelector('.featured-hero__trailer')).toBeNull()
    expect(secondPlayer.destroyed).toBe(true)

    fireEvent.click(screen.getByLabelText('Järgmine esiletõstetud sisu'))
    expect(screen.getByRole('region').getAttribute('aria-label')).toContain('Featured 1')

    await startNextPlayer()
    expect(players).toHaveLength(3)
  })

  it('keeps the hero artwork visible when every trailer candidate is unavailable', async () => {
    const item = featuredItem(1)
    item.metadata.trailerCandidates = [{ youtubeKey: 'trailer-fallback' }]
    render(<FeaturedHero items={[item]} onOpen={vi.fn()} />)

    await act(async () => {
      vi.advanceTimersByTime(850)
      await Promise.resolve()
    })
    const current = players.at(-1)
    if (!current) throw new Error('Expected a YouTube player')
    act(() => current.error(100))

    await act(async () => {
      vi.advanceTimersByTime(850)
      await Promise.resolve()
    })
    const fallback = players.at(-1)
    if (!fallback || fallback === current) throw new Error('Expected the fallback YouTube player')
    act(() => fallback.error(101))

    const hero = screen.getByRole('region', { name: /Featured 1/ })
    expect(hero.querySelector('.featured-hero__backdrop')).toBeTruthy()
    expect(hero.querySelector('.featured-hero__trailer')).toBeNull()
    expect(hero.classList.contains('featured-hero--playing')).toBe(false)

    const failures = JSON.parse(window.localStorage.getItem('kodumedia.trailer-failures.v2') || '{}')
    expect(failures['trailer-1']).toMatchObject({ code: 100 })
    expect(failures['trailer-fallback']).toMatchObject({ code: 101 })
  })

  it('clears the legacy failure cache and retries its falsely blocked trailer', async () => {
    window.localStorage.setItem('kodumedia.trailer-failures.v1', JSON.stringify({
      'trailer-1': Date.now() + 30 * 24 * 60 * 60 * 1000,
    }))

    render(<FeaturedHero items={[featuredItem(1)]} onOpen={vi.fn()} />)
    const current = await startNextPlayer()

    expect(current.options.videoId).toBe('trailer-1')
    expect(window.localStorage.getItem('kodumedia.trailer-failures.v1')).toBeNull()
  })

  it('does not cache a transient player error as a permanent trailer failure', async () => {
    render(<FeaturedHero items={[featuredItem(1)]} onOpen={vi.fn()} />)

    await act(async () => {
      vi.advanceTimersByTime(850)
      await Promise.resolve()
    })
    const current = players.at(-1)
    if (!current) throw new Error('Expected a YouTube player')
    act(() => current.error(5))

    const failures = JSON.parse(window.localStorage.getItem('kodumedia.trailer-failures.v2') || '{}')
    expect(failures['trailer-1']).toBeUndefined()
    expect(screen.getByRole('region', { name: /Featured 1/ })).toBeTruthy()
  })
})
