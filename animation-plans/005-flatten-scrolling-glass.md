# 005 — Flatten glass on scrolling surfaces

- **Status**: DONE
- **Commit**: source-archive-0.8.32
- **Severity**: HIGH
- **Category**: Performance
- **Estimated scope**: 2 CSS files, about 65 lines

## Problem

Fixed controls and content surfaces use `backdrop-filter` while the content
behind or around them scrolls. The browser must repeatedly sample and filter
moving pixels. Examples include:

```css
/* apps/web/src/design/primitives.css:199 — current */
.app-header__actions > .icon-button,
.header-notifications > .icon-button,
.header-notifications__button.icon-button {
  background: rgba(47, 44, 51, .54);
  backdrop-filter: blur(16px);
}
```

```css
/* apps/web/src/design/global.css:4996 — current */
.details__technical-band {
  backdrop-filter: blur(20px) saturate(125%);
}
```

The same pattern remains on `.settings-navigation`, `.library-quick-search`,
`.music-track-search`, `.music-all-tracks-entry`, `.system-settings-actions`
and the fixed `.music-player`.

## Target

Add a final performance stylesheet loaded after primitives. Disable both
`backdrop-filter` and `-webkit-backdrop-filter` on the identified scrolling or
fixed high-traffic surfaces, and replace transparent glass with visually close
opaque/near-opaque dark and light surfaces. Keep blur only for occasional,
small popovers and dialogs where it does not run during page scroll.

## Repo conventions to follow

The existing KoduMedia 0.8.32 override at
`apps/web/src/design/global.css:12428` already uses the same opaque-surface
strategy, but import order currently allows `primitives.css` to restore blur on
header icon buttons.

## Steps

1. Create `apps/web/src/design/performance.css`.
2. Import it after `primitives.css` in `apps/web/src/main.tsx`.
3. Disable backdrop filtering on the listed high-traffic surfaces.
4. Provide dark and light background colors using existing surface tokens or
   explicit near-opaque values.
5. Leave small, occasional `.ui-popover` and non-scrolling dialogs unchanged.

## Boundaries

- Do NOT remove borders, focus rings or elevation.
- Do NOT change popover placement or dialog motion.
- Do NOT add `!important` unless import order and specificity cannot express
  the override.
- If cited code has drifted, STOP and report it.

## Verification

- **Mechanical**: UI foundation check, TypeScript and production build pass.
- **Feel check**: scroll Home, Library, Music, Settings and long detail/season
  pages. The header and controls remain visually distinct without shimmer.
- **Performance check**: DevTools Rendering paint flashing must not repaint the
  full fixed header or large glass panels on every scroll frame.
- **Done when**: the identified surfaces compute to `backdrop-filter: none`.
