# 007 — Stabilize the ambient double buffer

- **Status**: DONE
- **Commit**: source-archive-0.8.34
- **Severity**: HIGH
- **Category**: Performance
- **Estimated scope**: 4 files, about 140 lines

## Problem

The homepage creates the previous and incoming full-viewport blur layers only
while a transition is active:

```tsx
// apps/web/src/shared/components/AmbientBackdrop.tsx:178 — current
return (
  <div className={backdropClassName} aria-hidden="true">
    {layers.previous ? (
      <div
        className="ambient-backdrop__layer ambient-backdrop__layer--previous ambient-backdrop__layer--visible"
        style={{ backgroundImage: `url(${layers.previous})` }}
      />
    ) : null}
    {layers.current ? (
      <div
        className={`ambient-backdrop__layer ambient-backdrop__layer--current ${layers.revealed ? 'ambient-backdrop__layer--visible' : ''}`.trim()}
        style={{ backgroundImage: `url(${layers.current})` }}
      />
    ) : null}
  </div>
)
```

The replacement is mounted, promoted through a transient parent class, hidden,
and then revealed across two animation frames:

```tsx
// apps/web/src/shared/components/AmbientBackdrop.tsx:114 — current
setLayers({ current: nextUrl, previous, revealed: false })

revealFrameRef.current = window.requestAnimationFrame(() => {
  revealFrameRef.current = window.requestAnimationFrame(() => {
    revealFrameRef.current = null
    setLayers((current) => (
      current.current === nextUrl
        ? { ...current, revealed: true }
        : current
    ))
```

At the same time, layer promotion changes with the transition state:

```css
/* apps/web/src/design/performance.css:28 — current */
.ambient-backdrop__layer {
  filter: blur(36px) saturate(1.08);
  opacity: 0;
  transition: opacity var(--motion-ui) var(--ease-in-out);
  will-change: auto;
}

.ambient-backdrop--transitioning .ambient-backdrop__layer {
  will-change: opacity;
}
```

The recorded browser frames show the resulting sequence as
`old → new/dark → old/brighter → new/dark` within roughly 130 ms. The incoming
filtered layer is therefore composited once before its hidden staging state is
stable. Creating, removing, and promoting large filtered layers during a
high-frequency thumbnail hover also makes the transition vulnerable to busy
frames.

## Target

- Keep exactly two `.ambient-backdrop__layer` elements mounted for the
  component's lifetime.
- Keep the settled base layer at `opacity: 1`.
- Load and decode the next URL before placing it in the hidden buffer layer.
- Stage the buffer at `opacity: 0` for two animation frames while the persistent
  layer remains promoted, then fade only that incoming layer to `opacity: 1`.
- Use `360ms cubic-bezier(.77, 0, .175, 1)` for the ambient crossfade. This
  intentionally exceeds the normal 300 ms UI budget because ambient artwork is
  a passive visual transition, not interaction feedback.
- On `transitionend`, atomically make the incoming layer the base and hide the
  old layer without fading either one again.
- During an active transition, retain only the latest decoded requested URL;
  never interrupt the current opacity transition or mount a third layer.
- Under `prefers-reduced-motion`, use `1ms` opacity timing while preserving the
  same double-buffer state machine.

## Repo conventions to follow

- `apps/web/src/design/tokens.css` defines
  `--ease-in-out: cubic-bezier(.77, 0, .175, 1)` for on-screen crossfades.
- `AmbientBackdrop` already deduplicates pending URL loads and retains a bounded
  decoded URL cache.
- `FeaturedHero` and homepage thumbnail handlers are out of scope; ambient
  changes continue to enter through the existing `url` prop.

## Steps

1. Replace the conditional previous/current layer model in
   `apps/web/src/shared/components/AmbientBackdrop.tsx` with two persistent
   indexed slots: one settled base, one hidden buffer.
2. Preserve URL decode and latest-request queueing, but stage a decoded URL only
   in the hidden slot and reveal it after two animation frames.
3. Finalize on the incoming layer's real `opacity` `transitionend`; switch the
   base index and hide the old layer in the same React commit. Keep a timer only
   as a missed-event fallback.
4. In `apps/web/src/design/performance.css`, give both persistent layers stable
   compositor promotion and apply the 360 ms transition only to the incoming
   role. Base and hidden roles must not animate during cleanup.
5. Replace the old DOM-count tests in
   `apps/web/src/shared/components/AmbientBackdrop.test.tsx` with regressions
   proving there are always two layers, the buffer remains hidden while its URL
   changes, only the incoming layer reveals, cleanup preserves full coverage,
   and rapid requests resolve to the latest URL.
6. Extend `apps/web/scripts/verify-ui-foundation.mjs` so future releases reject
   conditional ambient-layer mounting and transient `will-change` promotion.

## Boundaries

- Do NOT modify hero controls, their dimensions, markup, hover styles, or
  autoplay behavior.
- Do NOT change thumbnail hover timing.
- Do NOT remove the ambient blur, vignette, theme strength, or existing TMDb
  image-size optimization.
- Do NOT add a third layer or a dependency.
- Do NOT animate layout, filters, or background-image; animate opacity only.
- If the cited code has drifted from source archive 0.8.34, STOP and report
  instead of improvising.

## Verification

- **Mechanical**: run `npm run check:ui`,
  `npm test -- --run src/shared/components/AmbientBackdrop.test.tsx`,
  `npm run typecheck`, and `npm run build` in `apps/web`; all must pass.
- **Feel check**: sweep rapidly across at least ten homepage thumbnails, reverse
  direction twice, and pause on the last card. The current image must remain
  stable until one new image covers it, with no return to an earlier image and
  no luminance pulse.
- **Slow-motion check**: record at 30 fps and inspect every frame. Each hover
  must produce at most one monotonic old-to-new opacity transition; the sequence
  must never be `old → new → old → new`.
- **Compositor check**: in DevTools Layers, confirm two ambient artwork layers
  exist before, during, and after hover and are not promoted or destroyed during
  the transition.
- **Reduced motion**: emulate `prefers-reduced-motion`; the buffer swap may be
  effectively instant but the base remains covered throughout.
- **Done when**: two persistent layers remain mounted, only the decoded incoming
  layer animates, cleanup leaves one base plus one hidden buffer, and repeated
  hover produces no non-monotonic frame sequence.
