# 004 — Remove layout animations from frequent UI

- **Status**: DONE — hero-dot step superseded by the user-directed 0.8.34 style restoration
- **Commit**: source-archive-0.8.32
- **Severity**: HIGH
- **Category**: Performance
- **Estimated scope**: 1 CSS file, about 35 lines

## Problem

Several frequent interactions animate layout properties.

```css
/* apps/web/src/design/global.css:4141 — current */
transition: width var(--motion-fast) var(--ease-out), background-color var(--motion-fast) ease, transform var(--motion-press) var(--ease-out);
```

```css
/* apps/web/src/design/global.css:8593 — current */
transition: background 150ms ease, padding 180ms var(--ease-out), opacity 150ms ease;
```

```css
/* apps/web/src/design/global.css:1190 and :7443 — current */
transition: bottom 180ms var(--ease-out);
```

`width`, `padding` and `bottom` trigger layout and paint rather than compositor
work.

> **0.8.34 supersession:** the hero-dot transform requirement below was
> intentionally reversed to restore the user's established compact 7 px / 22 px
> slider styling. The notification and subtitle changes remain implemented.

## Target

- Give every hero dot a fixed 22 px box and represent inactive state with
  `transform: scaleX(.32)` from the center; selected state uses `scaleX(1)`.
- Transition only `transform 120ms cubic-bezier(.23, 1, .32, 1)` and
  `background-color 120ms ease`.
- Keep notification padding fixed, remove its layout shift, and transition only
  background color and opacity for 120 ms.
- Make subtitle/live-caption vertical state changes instant by setting
  `transition: none`.

## Repo conventions to follow

Use `--motion-press: 120ms` and `--ease-out` from
`apps/web/src/design/tokens.css`.

## Steps

1. Replace hero dot width animation with a fixed width and scale transform.
2. Remove notification-row padding animation and hover padding mutation.
3. Remove the two `bottom` transitions from subtitle layers.
4. Keep all focus-visible states and hit targets unchanged.

## Boundaries

- Do NOT shrink the clickable hero-dot area.
- Do NOT remove selected, hover or focus feedback.
- Do NOT change subtitle resting positions.
- If cited code has drifted, STOP and report it.

## Verification

- **Mechanical**: UI foundation check and production build pass; search confirms
  no `transition` includes `width`, `padding` or `bottom`.
- **Feel check**: click hero dots rapidly, sweep notification rows, and reveal
  player controls repeatedly. Responses must retarget or snap without layout
  movement.
- **Done when**: DevTools Performance shows no layout event caused by these
  transitions.
