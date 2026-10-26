# 001 — Make high-frequency scrolling instant

- **Status**: DONE
- **Commit**: source-archive-0.8.32
- **Severity**: HIGH
- **Category**: Purpose & frequency
- **Estimated scope**: 4 files, about 12 lines

## Problem

Global and keyboard-driven navigation animate scrolling even though these are
high-frequency actions. This delays the response and competes with image and
layout work.

```css
/* apps/web/src/design/global.css:8 — current */
html {
  scroll-behavior: smooth;
}
```

```ts
// apps/web/src/shared/hooks/useDirectionalNavigation.ts:50 — current
next.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' })
```

The same explicit smooth behavior appears in
`apps/web/src/features/media/MediaDetails.tsx:64` and
`apps/web/src/features/media/SeriesDetails.tsx:117`.

## Target

Use `scroll-behavior: auto` globally and `behavior: 'auto'` for keyboard and
detail-page jumps. The response must begin in the same frame as the input.

## Repo conventions to follow

`apps/web/src/app/router.ts:87` already uses
`window.scrollTo({ top: 0, left: 0, behavior: 'auto' })`.

## Steps

1. Set the root scroll behavior to `auto`.
2. Replace the three explicit `behavior: 'smooth'` calls with `behavior: 'auto'`.
3. Keep `prefers-reduced-motion` behavior intact.

## Boundaries

- Do NOT change routing or focus selection.
- Do NOT add a scrolling dependency.
- If cited code has drifted, STOP and report it.

## Verification

- **Mechanical**: run `npm test -- --run` and `npm run build` from `apps/web`;
  both must pass.
- **Feel check**: use arrow-key navigation across cards and activate both
  detail-page jump buttons. Confirm the target appears immediately with no
  animated travel.
- **Done when**: no production source contains `scroll-behavior: smooth` or
  `behavior: 'smooth'`.
