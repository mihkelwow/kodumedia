import type { MediaItem } from "../../api/types";

interface MediaSessionActions {
  play: () => void;
  pause: () => void;
  seekTo: (seconds: number) => void;
  seekBy: (seconds: number) => void;
  next?: () => void;
}

interface MediaSessionPosition {
  durationSeconds: number;
  positionSeconds: number;
  playing: boolean;
  playbackRate?: number;
  force?: boolean;
}

interface LastPositionState {
  duration: number;
  position: number;
  playbackRate: number;
  playing: boolean;
  reportedAt: number;
}

const MAX_POSITION_REPORT_INTERVAL_MS = 15_000;
const POSITION_DRIFT_THRESHOLD_SECONDS = 2.5;
let lastPositionState: LastPositionState | null = null;

function safeSetActionHandler(
  action: MediaSessionAction,
  handler: MediaSessionActionHandler | null,
): void {
  try {
    navigator.mediaSession.setActionHandler(action, handler);
  } catch {
    // Browsers expose different Media Session action subsets.
  }
}

function mediaTitle(item: MediaItem): string {
  if (item.type === "episode" && item.seriesTitle) return item.seriesTitle;
  return item.title;
}

function mediaSubtitle(item: MediaItem): string {
  if (item.type === "episode") {
    const code = `S${String(item.season ?? 0).padStart(2, "0")}E${String(item.episode ?? 0).padStart(2, "0")}`;
    return [code, item.title].filter(Boolean).join(" · ");
  }
  return item.year ? String(item.year) : "KoduMedia";
}

function nowMilliseconds(): number {
  return typeof performance !== "undefined" ? performance.now() : Date.now();
}

function projectedPosition(state: LastPositionState, now: number): number {
  if (!state.playing) return state.position;
  const elapsed = Math.max(0, now - state.reportedAt) / 1000;
  return state.position + elapsed * state.playbackRate;
}

export function installMediaSession(
  item: MediaItem,
  actions: MediaSessionActions,
): () => void {
  if (!("mediaSession" in navigator)) return () => undefined;

  if (typeof MediaMetadata !== "undefined") {
    const artwork = item.metadata.posterUrl || item.metadata.backdropUrl;
    navigator.mediaSession.metadata = new MediaMetadata({
      title: mediaTitle(item),
      artist: mediaSubtitle(item),
      album: item.type === "episode" ? item.title : "KoduMedia",
      artwork: artwork ? [{ src: artwork }] : [],
    });
  }

  safeSetActionHandler("play", () => actions.play());
  safeSetActionHandler("pause", () => actions.pause());
  safeSetActionHandler("seekbackward", (details) =>
    actions.seekBy(-(details.seekOffset ?? 10)),
  );
  safeSetActionHandler("seekforward", (details) =>
    actions.seekBy(details.seekOffset ?? 10),
  );
  safeSetActionHandler("seekto", (details) => {
    if (typeof details.seekTime === "number") actions.seekTo(details.seekTime);
  });
  safeSetActionHandler("nexttrack", actions.next ? () => actions.next?.() : null);

  return () => {
    for (const action of [
      "play",
      "pause",
      "seekbackward",
      "seekforward",
      "seekto",
      "nexttrack",
    ] as MediaSessionAction[]) {
      safeSetActionHandler(action, null);
    }
    navigator.mediaSession.metadata = null;
    lastPositionState = null;
    try {
      navigator.mediaSession.setPositionState();
    } catch {
      // Clearing the position state is optional in older implementations.
    }
  };
}

export function updateMediaSessionPosition({
  durationSeconds,
  positionSeconds,
  playing,
  playbackRate = 1,
  force = false,
}: MediaSessionPosition): void {
  if (!("mediaSession" in navigator)) return;
  if (!Number.isFinite(durationSeconds) || durationSeconds <= 0) return;

  const duration = Math.max(0.001, durationSeconds);
  const position = Math.min(duration, Math.max(0, positionSeconds));
  const rate = Number.isFinite(playbackRate) && playbackRate > 0 ? playbackRate : 1;
  const now = nowMilliseconds();
  const previous = lastPositionState;

  if (previous && !force) {
    const durationChanged = Math.abs(previous.duration - duration) > 0.5;
    const rateChanged = Math.abs(previous.playbackRate - rate) > 0.01;
    const playbackStateChanged = previous.playing !== playing;
    const stale = now - previous.reportedAt >= MAX_POSITION_REPORT_INTERVAL_MS;
    const drift = Math.abs(projectedPosition(previous, now) - position);

    if (
      !durationChanged &&
      !rateChanged &&
      !playbackStateChanged &&
      !stale &&
      drift < POSITION_DRIFT_THRESHOLD_SECONDS
    ) {
      return;
    }
  }

  try {
    navigator.mediaSession.playbackState = playing ? "playing" : "paused";
  } catch {
    // playbackState is not implemented everywhere.
  }

  try {
    navigator.mediaSession.setPositionState({
      duration,
      playbackRate: rate,
      position,
    });
    lastPositionState = {
      duration,
      position,
      playbackRate: rate,
      playing,
      reportedAt: now,
    };
  } catch {
    // Native PiP falls back to the HTMLMediaElement state if unsupported.
  }
}

export function resetMediaSessionPositionThrottle(): void {
  lastPositionState = null;
}
