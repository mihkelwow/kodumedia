# 002 — Bound ambient backdrop work

- **Status**: DONE
- **Commit**: source-archive-0.8.32
- **Severity**: HIGH
- **Category**: Performance
- **Estimated scope**: 2 files, about 70 lines

## Problem

Every settled ambient crossfade retains the previous full-viewport filtered
layer in the DOM. The current implementation also decodes the original
`w1280` TMDb backdrop even though a heavily blurred ambient layer needs far
less source detail.

```tsx
// apps/web/src/shared/components/AmbientBackdrop.tsx:79 — current
{layers.previous ? (
  <div
    className={`ambient-backdrop__layer ambient-backdrop__layer--previous ${layers.revealed ? '' : 'ambient-backdrop__layer--visible'}`.trim()}
    style={{ backgroundImage: `url(${layers.previous})` }}
  />
) : null}
```

```css
/* apps/web/src/design/global.css:12398 — current */
.ambient-backdrop__layer {
  contain: paint;
  filter: blur(62px) saturate(1.12);
  opacity: 0;
  transform: translateZ(0) scale(1.12);
  transition: opacity var(--motion-ui) var(--ease-out);
  backface-visibility: hidden;
  will-change: opacity;
}
```

## Target

- Normalize only TMDb `/t/p/w1280/` ambient URLs to `/t/p/w500/`.
- Deduplicate pending and decoded image work by URL.
- Crossfade with `opacity 200ms cubic-bezier(.23, 1, .32, 1)`.
- Remove the previous layer after that transition settles.
- Reduce the static ambient filter to `blur(36px)` while preserving cover,
  no-repeat, overscan and the existing vignette.

## Repo conventions to follow

Motion tokens already live in `apps/web/src/design/tokens.css`:
`--motion-ui: 200ms` and `--ease-out: cubic-bezier(.23, 1, .32, 1)`.
`AmbientBackdrop` already waits for decode before revealing the new layer.

## Steps

1. Add a pure ambient URL normalizer and a module-level preload cache.
2. Apply the normalized URL before state comparison and preloading.
3. Schedule cleanup of `previous` after the 200 ms opacity transition and
   cancel stale animation frames/timers on URL changes or unmount.
4. Reduce the final ambient blur override to 36 px.
5. Add tests for URL normalization, preload deduplication and previous-layer
   cleanup.

## Boundaries

- Do NOT lower hero or detail artwork quality.
- Do NOT rewrite non-TMDb URLs.
- Do NOT remove the ambient vignette or blur entirely.
- Do NOT add dependencies.
- If cited code has drifted, STOP and report it.

## Verification

- **Mechanical**: component tests, TypeScript and production build must pass.
- **Feel check**: sweep rapidly across ten home cards, then stop. Confirm the
  last backdrop appears once decoded, the crossfade is under 200 ms, and
  DevTools Elements shows only one ambient layer after settling.
- **Performance check**: record a scroll trace and confirm no repeated large
  paint caused by an invisible previous ambient layer.
- **Done when**: one settled backdrop equals one filtered layer and TMDb
  ambient images request `w500`.
