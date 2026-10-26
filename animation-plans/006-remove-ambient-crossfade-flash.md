# 006 — Remove ambient crossfade flash

- **Status**: DONE
- **Commit**: source-archive-0.8.33
- **Severity**: HIGH
- **Category**: Performance
- **Estimated scope**: 3 files, about 70 lines

## Problem

The homepage ambient artwork swaps two full-viewport filtered layers by fading
the previous layer out while the current layer fades in. Both layers carry the
final ambient opacity themselves:

```tsx
// apps/web/src/shared/components/AmbientBackdrop.tsx:125 — current
className={`ambient-backdrop__layer ambient-backdrop__layer--previous ${layers.revealed ? '' : 'ambient-backdrop__layer--visible'}`.trim()}
```

```css
/* apps/web/src/design/global.css:12418 — current */
.ambient-backdrop__layer--visible {
  opacity: .34;
}
```

The two semi-transparent layers do not maintain constant perceived luminance
during the overlap. Cleanup is also driven by a 220 ms JavaScript timer rather
than the actual CSS transition:

```tsx
// apps/web/src/shared/components/AmbientBackdrop.tsx:107 — current
cleanupTimerRef.current = window.setTimeout(() => {
  setLayers((current) => current.current === normalizedUrl ? { ...current, previous: '' } : current)
}, backdropTransitionMs + 20)
```

On a busy frame, the old filtered layer can therefore disappear before the new
one is visibly settled, producing the reported flash.

## Target

- Keep the settled previous layer at `opacity: 1` underneath the incoming layer.
- Fade only the incoming layer from `opacity: 0` to `opacity: 1` over
  `200ms cubic-bezier(.77, 0, .175, 1)`.
- Apply the dark/light/subtle ambient strength once on the parent container:
  `.34`, `.17`, `.18`, and `.1` respectively.
- Remove the previous layer on the incoming layer's real `opacity`
  `transitionend`, with a conservative timer only as a fallback.
- Promote the two filtered layers only while a transition is active.
- Keep the already-decoded current image visible until the replacement has
  decoded and been committed for two animation frames.

## Repo conventions to follow

`apps/web/src/design/tokens.css` already defines
`--motion-ui: 200ms` and
`--ease-in-out: cubic-bezier(.77, 0, .175, 1)`.
`AmbientBackdrop` already preloads and decodes replacements before changing
the rendered URLs.

## Steps

1. Change `AmbientBackdrop` so the previous layer stays visible throughout the
   incoming layer's transition.
2. Add an `opacity` transition-end cleanup handler and retain a delayed fallback
   for browsers that do not emit the event.
3. Move ambient strength from each child layer to the parent container and use
   opacity `0`/`1` for the layers.
4. Add tests proving the previous layer remains visible while the incoming layer
   reveals and that `transitionend` performs cleanup without a timed gap.

## Boundaries

- Do NOT change hero navigation markup or redesign its controls.
- Do NOT remove the blurred ambient treatment or vignette.
- Do NOT change thumbnail hover timing.
- Do NOT add dependencies.
- If the cited code has drifted, stop and report it instead of improvising.

## Verification

- **Mechanical**: run the ambient component test, TypeScript, UI foundation
  check, and production build; all must pass.
- **Feel check**: sweep across homepage thumbnails, then change direction
  repeatedly. The old image must stay stable until the new image softly covers
  it; there must be no black/bright frame or brief loss of ambient strength.
- **Slow-motion check**: in DevTools Animations, set playback to 10% and confirm
  only the incoming layer changes opacity while the outgoing layer stays at 1.
- **Reduced motion**: emulate `prefers-reduced-motion`; the replacement may be
  effectively instant but must still never expose an empty background.
- **Done when**: transition cleanup leaves one filtered layer and no frame has
  both ambient images below full combined coverage.
