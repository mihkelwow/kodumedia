# Animation performance plans

| # | Title | Severity | Status |
| --- | --- | --- | --- |
| 001 | Make high-frequency scrolling instant | HIGH | DONE |
| 002 | Bound ambient backdrop work | HIGH | DONE |
| 003 | Keep cast scrolling out of React | HIGH | DONE |
| 004 | Remove layout animations from frequent UI | HIGH | DONE; hero-dot step superseded in 0.8.34 |
| 005 | Flatten glass on scrolling surfaces | HIGH | DONE |
| 006 | Remove ambient crossfade flash | HIGH | DONE |
| 007 | Stabilize the ambient double buffer | HIGH | DONE |
| 008 | Restore the cinematic hero trailer crossfade | HIGH | DONE |

Recommended execution order: 001 → 002 → 003 → 004 → 005 → 006 → 007 → 008.

Plan 002 and plan 005 jointly reduce scroll cost: the first bounds filtered
ambient artwork, while the second prevents moving content from being resampled
through large glass surfaces. The other plans are independent.

Plan 006 refines plan 002's decoded crossfade: it preserves constant visual
coverage and cleans up on the real opacity transition rather than trusting a
frame-sensitive timer.

Plan 007 supersedes plan 006's conditional layer lifecycle after recorded
browser frames proved that mounting and promoting the incoming filtered layer
during hover can still produce a non-monotonic flash. It keeps both buffers
mounted and promoted throughout the component lifetime.

Plan 008 is independent of the ambient backdrop plans. It restores the
established 480 ms hero poster-to-trailer opacity crossfade while preserving the
existing `PLAYING` readiness gate and static-artwork fallback.

Missed opportunities intentionally deferred:

- A brief 120 ms opacity bridge could soften manual hero copy swaps, but it
  should be evaluated only after the backdrop and trailer work is stable.
- Rare first-run and successful-library-scan moments have room for restrained
  delight; neither belongs in this performance-focused release.
