# 008 — Restore the cinematic hero trailer crossfade

- **Status**: DONE
- **Commit**: source-archive-0.8.36-e3be1e01
- **Severity**: HIGH
- **Category**: Easing & duration
- **Estimated scope**: 4 files, about 35 lines

## Problem

The hero correctly keeps its artwork mounted under the YouTube player and only
reveals the player after the IFrame API reports the real `PLAYING` state.
However, the reveal duration was shortened from the established cinematic
timing to the normal UI timing:

```css
/* apps/web/src/design/global.css:4023 — current */
.featured-hero__trailer {
  z-index: -3;
  overflow: hidden;
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--motion-ui) var(--ease-out);
}
```

`--motion-ui` is 200 ms:

```css
/* apps/web/src/design/tokens.css:24 — current */
--motion-ui: 200ms;
--motion-cinematic: 480ms;
```

This makes the large poster-to-moving-video state change read as a cut rather
than a soft Netflix-like blend. KoduMedia 0.8.31 used the same persistent
backdrop and trailer-layer structure with
`opacity var(--motion-cinematic) var(--ease-out)`, so the earlier desired
behavior can be restored without changing player lifecycle or markup.

## Target

Keep the backdrop fully mounted and visible below the trailer layer. When the
YouTube player reaches the real `PLAYING` state, fade only the trailer layer
from `opacity: 0` to `opacity: 1` over exactly 480 ms with
`cubic-bezier(.23, 1, .32, 1)` through the existing tokens:

```css
/* target */
.featured-hero__trailer {
  opacity: 0;
  transition: opacity var(--motion-cinematic) var(--ease-out);
}

.featured-hero--playing .featured-hero__trailer {
  opacity: 1;
}
```

Do not fade or remove the fallback backdrop. Do not reveal an unready,
buffering, blocked, or failed trailer. Under `prefers-reduced-motion: reduce`,
the existing global override keeps this transition effectively instant.

## Repo conventions to follow

- Motion tokens live in `apps/web/src/design/tokens.css`;
  `--motion-cinematic: 480ms` already exists for passive cinematic changes.
- `apps/web/src/features/media/FeaturedHero.tsx:326` waits for
  `YouTube.PlayerState.PLAYING` before setting `playerReady`.
- KoduMedia 0.8.31 used the exact
  `opacity var(--motion-cinematic) var(--ease-out)` transition for this same
  class and layer structure.
- Permanent UI invariants are enforced in
  `apps/web/scripts/verify-ui-foundation.mjs`.

## Steps

1. In `apps/web/src/design/global.css`, change only the hero trailer opacity
   duration token from `--motion-ui` to `--motion-cinematic`.
2. In `apps/web/scripts/verify-ui-foundation.mjs`, require the exact cinematic
   hero transition and retain the existing ban on duplicate hero runtimes.
3. In `apps/web/src/features/media/FeaturedHero.test.tsx`, preserve the
   behavioral gate proving that the playing class appears only after the mock
   player emits `PLAYING`, and add an assertion that buffering does not reveal
   the video.
4. Document the restored transition and its permanent design constraint in
   `CHANGELOG.md`.

## Boundaries

- Do NOT change hero autoplay delay, muted default, sound behavior, trailer
  filtering, fallback order, carousel behavior, arrows, dots, counter, markup,
  iframe crop, shade, or artwork.
- Do NOT add timers, keyframes, JavaScript animation, blur animation, or a new
  dependency.
- Do NOT fade out or unmount the fallback backdrop when playback starts.
- Do NOT reveal the trailer before the player reports `PLAYING`.
- Do NOT change the 480 ms value again unless the user explicitly requests it.
- If the cited code has drifted from source archive 0.8.36, stop and report
  instead of improvising.

## Verification

- **Mechanical**: in `apps/web`, run
  `npm test -- --run src/features/media/FeaturedHero.test.tsx`,
  `npm run check:ui`, `npm run typecheck`, and `npm run build`; all must pass.
- **Feel check**: open the homepage with a trailer-capable item and confirm that
  the static artwork remains steady until moving video is genuinely playing,
  then dissolves into it without a black frame or hard cut.
- **Slow-motion check**: use the DevTools Animations panel at 10% playback and
  confirm one monotonic 0 → 1 trailer-opacity transition lasting 480 ms, with
  the backdrop continuously present beneath it.
- **Failure check**: simulate buffering, an embed error, and an unavailable
  candidate; none may expose the iframe or remove the artwork.
- **Reduced motion**: emulate `prefers-reduced-motion: reduce`; the poster-to-
  trailer state change must be effectively instant and must not animate blur,
  scale, or position.
- **Done when**: the hero uses the exact established 480 ms opacity crossfade,
  only `PLAYING` reveals the trailer, failures remain on artwork, and automated
  checks reject a future return to normal UI timing.
