# 003 — Keep cast scrolling out of React

- **Status**: DONE
- **Commit**: source-archive-0.8.32
- **Severity**: HIGH
- **Category**: Performance
- **Estimated scope**: 1 file, about 45 lines

## Problem

Every cast-list scroll event reads layout and updates React state, so dragging
the list re-renders every person card and the scrollbar on nearly every frame.

```tsx
// apps/web/src/features/media/CastCarousel.tsx:22 — current
const updateMetrics = useCallback(() => {
  const scroller = scrollerRef.current
  if (!scroller) return
  setMetrics({ clientWidth: scroller.clientWidth, scrollWidth: scroller.scrollWidth, scrollLeft: scroller.scrollLeft })
}, [])
```

```tsx
// apps/web/src/features/media/CastCarousel.tsx:58 — current
onScroll={updateMetrics}
```

## Target

React state stores only resize-dependent width and scrollability. The scroll
handler reads `scrollLeft` and writes the thumb's full
`transform: translateX(...)` string directly to the thumb element. No React
state update may occur on scroll or pointer move.

## Repo conventions to follow

`apps/web/src/design/global.css:12369` already transitions only the cast thumb's
`transform`, and disables that transition while dragging.

## Steps

1. Add a `thumbRef` and a ref for the resize-derived maximum scroll and track
   travel.
2. Keep `ResizeObserver` responsible for width state.
3. Add a direct `syncThumbPosition()` DOM transform write for scroll.
4. Call it from resize and `onScroll` without calling `setState`.
5. Preserve pointer capture and drag behavior.

## Boundaries

- Do NOT change person-card markup or the 12-person limit.
- Do NOT introduce CSS variables on a parent.
- Do NOT add a requestAnimationFrame loop.
- If cited code has drifted, STOP and report it.

## Verification

- **Mechanical**: TypeScript, tests and production build pass.
- **Feel check**: drag the cast strip quickly in both directions. The thumb
  tracks the list without lag and pointer capture remains active outside the
  strip.
- **Performance check**: React DevTools highlight updates must not flash cast
  cards while scrolling.
- **Done when**: the scroll callback performs no React state update.
