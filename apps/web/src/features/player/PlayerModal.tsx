import type HlsType from "hls.js";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  ClientCapabilities,
  HDRMode,
  MediaItem,
  PlaybackOptions,
  PlaybackPlan,
  PlaybackPreferences,
  PlaybackRuntimeStats,
  PlaybackSession,
  PostPlay,
  SubtitleSearchResult,
  SubtitleTrack,
} from "../../api/types";
import { api } from "../../api/client";
import { Button } from "../../shared/components/Button";
import { Icon } from "../../shared/components/Icon";
import { IconButton } from "../../shared/components/IconButton";
import { Spinner } from "../../shared/components/Spinner";
import { detectClientCapabilities } from "./capabilities";
import { activeSubtitleCues, parseWebVtt, type SubtitleCue } from "./webvtt";
import { getPlaybackOptions, preloadPlayback } from "./preload";
import { PlayerTrackMenu, type PlayerTrackMenuOption } from "./PlayerTrackMenu";
import { PlayerPostPlay } from "./PlayerPostPlay";
import { PlayerControls } from "./PlayerControls";
import { PlayerDiagnostics } from "./PlayerDiagnostics";
import { PlayerPlaybackStatus } from "./PlayerPlaybackStatus";
import { PlayerCastMenu } from "./PlayerCastMenu";
import { decoderLabel, encoderLabel, profileLabel, resolutionLabel } from "./playerLabels";
import { PlayerSubtitleSearchPanel } from "./PlayerSubtitleSearchPanel";
import { usePlayerControlsVisibility } from "./usePlayerControlsVisibility";
import { installMediaSession, updateMediaSessionPosition } from "./mediaSession";
import { usePlayerCasting } from "./usePlayerCasting";
import { getLocale, tr, trExternal, useLocale } from "../../i18n/i18n";
import {
  createLiveCaptionController,
  liveCaptionLanguageLabel,
  type LiveCaptionController,
  type LiveCaptionSnapshot,
} from "./liveCaption";

function errorText(value: unknown, fallback: string): string {
  return value instanceof Error && value.message ? value.message : fallback;
}


const HDR_MODE_STORAGE_KEY = "kodumedia:hdr-mode";

function storedHDRMode(): HDRMode {
  const value = window.localStorage.getItem(HDR_MODE_STORAGE_KEY);
  return value === "hdr" || value === "sdr" ? value : "auto";
}

function describeHlsFailure(type: string, details: string): string {
  const normalized = `${type} ${details}`.toLowerCase();
  if (
    normalized.includes("incompatiblecodec") ||
    normalized.includes("addcodec") ||
    normalized.includes("codec") && normalized.includes("unsupported")
  ) {
    return tr("Brauser ei võtnud HDR/HEVC codec'it vastu ({{details}}).", { details });
  }
  if (normalized.includes("network")) {
    return tr('HDR-voo laadimine ebaõnnestus ({{details}}).', { details });
  }
  if (normalized.includes("media")) {
    return tr('Brauser ei suutnud HDR-videot dekodeerida ({{details}}).', { details });
  }
  return tr('HDR HLS-voog ebaõnnestus ({{details}}).', { details });
}

function hdrFallbackMessage(reason: string): string {
  const trimmed = reason.trim().replace(/[.]+$/, "");
  return tr('{{reason}}. KoduMedia jätkas samast kohast SDR fallback-vooga.', { reason: trimmed });
}

function isClearlyUnsupportedHlsDetail(details: string): boolean {
  const normalized = details.toLowerCase();
  return (
    normalized.includes("incompatiblecodec") ||
    normalized.includes("addcodec") ||
    normalized.includes("manifestincompatible") ||
    normalized.includes("mediasourceopen")
  );
}

export function actualPositionMs(
  video: HTMLVideoElement | null,
  session: PlaybackSession | null,
): number {
  if (!video || !session) return session?.plan.startPositionMs ?? 0;
  const base =
    session.plan.mode === "direct" ? 0 : (session.plan.startPositionMs ?? 0);
  return Math.max(0, base + Math.round(video.currentTime * 1000));
}

export function bufferedAheadSeconds(video: HTMLVideoElement): number {
  const position = video.currentTime;
  for (let index = 0; index < video.buffered.length; index += 1) {
    const start = video.buffered.start(index);
    const end = video.buffered.end(index);
    if (position >= start - 0.15 && position <= end + 0.15)
      return Math.max(0, end - position);
  }
  return 0;
}


export function isBufferedAt(video: HTMLVideoElement, seconds: number): boolean {
  for (let index = 0; index < video.buffered.length; index += 1) {
    if (seconds >= video.buffered.start(index) - 0.1 && seconds <= video.buffered.end(index) - 0.1) return true;
  }
  return false;
}

export type BufferPhase = "preparing" | "startup" | "ready" | "playing" | "paused" | "rebuffering";

export interface PlayerBufferPolicy {
  startupSeconds: number;
  rebufferSeconds: number;
  desiredSeconds: number;
  lowSeconds: number;
}

export function startupBufferTargetSeconds(
  durationSeconds: number,
  plan?: PlaybackPlan,
): number {
  const short = durationSeconds > 0 && durationSeconds < 90;
  const planned = (plan?.startupBufferSeconds ?? 0) > 0
    ? plan?.startupBufferSeconds ?? 8
    : short ? 3 : 8;
  // The larger plan value also describes how much headroom the encoder should
  // build and how long a DLNA hand-off may wait on the server. Requiring the
  // browser's paused media element to reach 14–20 seconds can deadlock startup:
  // Chromium may stop appending at roughly ten seconds until playback begins.
  return Math.min(planned, short ? 3 : 8);
}

export function shouldStartBufferedPlayback(
  bufferedAhead: number,
  targetSeconds: number,
  waitingMs: number,
  readyState: number,
): boolean {
  if (bufferedAhead >= targetSeconds) return true;
  // Fail open once the media element has playable data. This keeps a slow or
  // conservative MSE implementation from waiting forever just below target.
  return waitingMs >= 6_000 && (bufferedAhead >= 2 || readyState >= 3);
}

export function bufferPolicyForPlan(
  plan: PlaybackPlan | undefined,
  durationSeconds: number,
): PlayerBufferPolicy {
  const startupSeconds = startupBufferTargetSeconds(durationSeconds, plan);
  const short = durationSeconds > 0 && durationSeconds < 90;
  return {
    startupSeconds,
    rebufferSeconds:
      Math.min(
        plan?.rebufferBufferSeconds ?? (short ? 6 : plan?.mode === "direct" ? 8 : 20),
        short ? 6 : 10,
      ),
    desiredSeconds:
      plan?.desiredBufferSeconds ?? (short ? 12 : plan?.mode === "direct" ? 20 : 40),
    lowSeconds:
      Math.min(
        plan?.lowBufferSeconds ?? (short ? 3 : plan?.mode === "direct" ? 4 : 12),
        short ? 3 : 6,
      ),
  };
}

export function hlsBufferSizeBytes(bitrate: number | undefined, desiredSeconds: number): number {
  const minimum = 64 * 1024 * 1024;
  // This is a ceiling, not an allocation. High-bitrate 4K streams need enough
  // byte headroom to reach the 30-second rebuffer target; 256 MiB could cap a
  // 90 Mb/s stream below that target and keep the player in a rebuffer loop.
  const maximum = 512 * 1024 * 1024;
  if (!bitrate || bitrate <= 0) return minimum;
  const estimated = Math.ceil((bitrate * Math.max(20, desiredSeconds) * 1.25) / 8);
  return Math.max(minimum, Math.min(maximum, estimated));
}

export function shouldUseNativeHls(
  adaptive: boolean,
  nativeHlsSupported: boolean,
  webKitAirPlaySupported: boolean,
): boolean {
  // A non-empty canPlayType() answer is not strong enough to select a native
  // HLS implementation. Chromium builds can advertise partial/native HLS
  // support while still rejecting the generated event playlist at playback
  // time with MEDIA_ERR_SRC_NOT_SUPPORTED. Keep the network URL required by
  // AirPlay only on WebKit clients that expose the native playback-target
  // picker; every other MSE-capable browser stays on the proven hls.js path.
  return adaptive && nativeHlsSupported && webKitAirPlaySupported;
}


function trackLabel(track: SubtitleTrack): string {
  const primary = track.title || track.language?.toUpperCase() || tr('Subtiitrid');
  const details = [primary];
  if (track.forced && !primary.toLowerCase().includes("forced"))
    details.push("forced");
  if (track.hearingImpaired && !primary.toLowerCase().includes("sdh"))
    details.push("SDH");
  if (track.requiresBurnIn) details.push(tr('pildisubtiiter'));
  return details.join(" · ");
}

function audioLabel(
  track: PlaybackOptions["audioTracks"][number],
  index: number,
): string {
  const label =
    track.title || track.language?.toUpperCase() || tr('Helirada {{number}}', { number: index + 1 });
  const details = [
    track.codec?.toUpperCase(),
    track.channels ? tr('{{count}} kanalit', { count: track.channels }) : "",
  ].filter(Boolean);
  return details.length ? `${label} · ${details.join(" · ")}` : label;
}

function audioTrackTitle(
  track: PlaybackOptions["audioTracks"][number],
  index: number,
): string {
  return track.title || track.language?.toUpperCase() || tr('Helirada {{number}}', { number: index + 1 });
}

function audioTrackDescription(
  track: PlaybackOptions["audioTracks"][number],
): string {
  return [
    track.language?.toUpperCase(),
    track.codec?.toUpperCase(),
    track.channels ? tr('{{count}} kanalit', { count: track.channels }) : "",
    track.channelLayout,
    track.default ? tr('vaikimisi') : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

function subtitleMatchLabel(matchType?: string): string {
  switch (matchType) {
    case "hash":
      return tr('faili räsi vaste');
    case "imdb":
      return tr('IMDb vaste');
    case "tmdb":
      return tr('TMDb vaste');
    case "title":
      return tr('pealkirja vaste');
    case "broad-title":
      return tr('lai pealkirjaotsing');
    default:
      return "";
  }
}

const LANGUAGE_ALIASES: Record<string, string> = {
  est: "et",
  et: "et",
  eng: "en",
  en: "en",
  fin: "fi",
  fi: "fi",
  swe: "sv",
  sv: "sv",
  rus: "ru",
  ru: "ru",
  ger: "de",
  deu: "de",
  de: "de",
  fre: "fr",
  fra: "fr",
  fr: "fr",
  spa: "es",
  es: "es",
  ita: "it",
  it: "it",
  lav: "lv",
  lv: "lv",
  lit: "lt",
  lt: "lt",
  nor: "no",
  no: "no",
  dan: "da",
  da: "da",
  pol: "pl",
  pl: "pl",
  ukr: "uk",
  uk: "uk",
  jpn: "ja",
  ja: "ja",
  zho: "zh",
  chi: "zh",
  zh: "zh",
  ces: "cs",
  cze: "cs",
  cs: "cs",
  dut: "nl",
  nld: "nl",
  nl: "nl",
  por: "pt",
  pt: "pt",
};

function normalizeLanguageCode(language?: string): string {
  if (!language) return "";
  const trimmed = language.trim().toLowerCase();
  const base = trimmed.split(/[-_]/)[0] || trimmed;
  return LANGUAGE_ALIASES[base] || base;
}

function languagePriority(language: string | undefined, preferredLanguages: string[]): number {
  if (preferredLanguages.length === 0) return Number.MAX_SAFE_INTEGER;
  const normalized = normalizeLanguageCode(language);
  if (!normalized) return Number.MAX_SAFE_INTEGER;
  const index = preferredLanguages.findIndex(
    (value) => normalizeLanguageCode(value) === normalized,
  );
  return index === -1 ? Number.MAX_SAFE_INTEGER : index;
}

function sortSubtitleTracks(
  tracks: SubtitleTrack[],
  preferredLanguages: string[],
): SubtitleTrack[] {
  return [...tracks].sort((left, right) => {
    const priorityDifference =
      languagePriority(left.language, preferredLanguages) -
      languagePriority(right.language, preferredLanguages);
    if (priorityDifference !== 0) return priorityDifference;
    if (left.forced !== right.forced) return left.forced ? -1 : 1;
    return trackLabel(left).localeCompare(trackLabel(right), getLocale(), {
      sensitivity: "base",
    });
  });
}

function sortSubtitleResults(
  results: SubtitleSearchResult[],
  preferredLanguages: string[],
): SubtitleSearchResult[] {
  return [...results].sort((left, right) => {
    const priorityDifference =
      languagePriority(left.language, preferredLanguages) -
      languagePriority(right.language, preferredLanguages);
    if (priorityDifference !== 0) return priorityDifference;
    if (left.trusted !== right.trusted) return left.trusted ? -1 : 1;
    return (right.downloads ?? 0) - (left.downloads ?? 0);
  });
}

export function postPlayWindowSeconds(durationSeconds: number): number {
  if (durationSeconds <= 0) return 30;
  return Math.max(15, Math.min(45, durationSeconds * 0.08));
}

export function PlayerModal({
  item,
  onClose,
  onWatchUpdated,
  onPlayItem,
}: {
  item: MediaItem;
  onClose: () => void;
  onWatchUpdated?: () => void;
  onPlayItem: (item: MediaItem) => void;
}) {
  const locale = useLocale();
  const shellRef = useRef<HTMLDivElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const liveCaptionControllerRef = useRef<LiveCaptionController | null>(null);
  const capabilitiesRef = useRef<ClientCapabilities | null>(null);
  const [capabilities, setCapabilities] = useState<ClientCapabilities | null>(null);
  const sessionRef = useRef<PlaybackSession | null>(null);
  const generationRef = useRef(0);
  const fallbackUsed = useRef(false);
  const recoveryAttempts = useRef(0);
  const lastPositionMs = useRef(0);
  const durationMsRef = useRef(
    Math.max(0, item.technical.durationMs ?? item.watch?.durationMs ?? 0),
  );
  const lastProgressSentAt = useRef(0);
  const lastProgressPosition = useRef(-1);
  const fatalHandler = useRef<(message: string) => void>(() => undefined);
  const bufferPhaseRef = useRef<BufferPhase>("preparing");
  const resumeAfterBufferRef = useRef(false);
  const stallStartedAtRef = useRef<number | null>(null);
  const recentStallsRef = useRef<number[]>([]);
  const lowBufferSinceRef = useRef<number | null>(null);
  const stabilityFallbackInFlightRef = useRef(false);
  const requestedHDRModeRef = useRef<HDRMode>(
    item.technical.hdrFormat ? storedHDRMode() : "auto",
  );
  const finishedRef = useRef(false);
  const autoplayCancelledRef = useRef(false);
  const advanceStartedRef = useRef(false);
  const castProgressRef = useRef<number | null>(null);

  const [options, setOptions] = useState<PlaybackOptions>({
    mediaId: item.id,
    audioTracks: [],
    subtitles: [],
  });
  const [preferences, setPreferences] = useState<PlaybackPreferences>({});
  const [session, setSession] = useState<PlaybackSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [playing, setPlaying] = useState(false);
  const [mediaReady, setMediaReady] = useState(false);
  const [bufferPhase, setBufferPhase] = useState<BufferPhase>("preparing");
  const [bufferedAhead, setBufferedAhead] = useState(0);
  const [runtimeStats, setRuntimeStats] = useState<PlaybackRuntimeStats | null>(null);
  const [stallCount, setStallCount] = useState(0);
  const [stallDurationMs, setStallDurationMs] = useState(0);
  const [stabilityNotice, setStabilityNotice] = useState("");
  const [diagnosticsOpen, setDiagnosticsOpen] = useState(false);
  const [currentSeconds, setCurrentSeconds] = useState(0);
  const [durationSeconds, setDurationSeconds] = useState(
    Math.max(0, (item.technical.durationMs ?? 0) / 1000),
  );
  const [seekDraft, setSeekDraft] = useState<number | null>(null);
  const [volume, setVolume] = useState(1);
  const [muted, setMuted] = useState(false);
  const [subtitlePanel, setSubtitlePanel] = useState(false);
  const [audioMenuOpen, setAudioMenuOpen] = useState(false);
  const [subtitleMenuOpen, setSubtitleMenuOpen] = useState(false);
  const [liveCaptionMenuOpen, setLiveCaptionMenuOpen] = useState(false);
  const [castMenuOpen, setCastMenuOpen] = useState(false);
  const casting = usePlayerCasting({
    item,
    videoRef,
    preferences,
    positionMs: currentSeconds * 1000,
    durationMs: durationSeconds * 1000,
    menuOpen: castMenuOpen,
    progressRef: castProgressRef,
  });
  const activeRemote = casting.activeRemote;
  const effectivePlaying = activeRemote?.playing ?? playing;
  const effectiveCurrentSeconds = (activeRemote?.positionMs ?? currentSeconds * 1000) / 1000;
  const effectiveDurationSeconds = (activeRemote?.durationMs ?? durationSeconds * 1000) / 1000;
  const effectiveVolume = activeRemote?.volume ?? volume;
  const effectiveMuted = activeRemote?.muted ?? muted;
  const controlsLocked = subtitlePanel || audioMenuOpen || subtitleMenuOpen || liveCaptionMenuOpen || castMenuOpen;
  const { visible: controlsVisible, reveal: revealControls } = usePlayerControlsVisibility({
    playing: effectivePlaying,
    locked: controlsLocked,
  });
  const [liveCaptionEnabled, setLiveCaptionEnabled] = useState(false);
  const [liveCaptionSnapshot, setLiveCaptionSnapshot] = useState<LiveCaptionSnapshot>({
    state: "idle",
    text: "",
  });
  const [preferredSubtitleLanguages, setPreferredSubtitleLanguages] = useState<string[]>(["en"]);
  const [searchingSubtitles, setSearchingSubtitles] = useState(false);
  const [subtitleSearchError, setSubtitleSearchError] = useState("");
  const [subtitleResults, setSubtitleResults] = useState<
    SubtitleSearchResult[]
  >([]);
  const [downloadingSubtitle, setDownloadingSubtitle] = useState<number | null>(
    null,
  );
  const [subtitleCues, setSubtitleCues] = useState<SubtitleCue[]>([]);
  const [subtitleLoading, setSubtitleLoading] = useState(false);
  const [subtitleLoadError, setSubtitleLoadError] = useState("");
  const [postPlay, setPostPlay] = useState<PostPlay | null>(null);
  const [playbackEnded, setPlaybackEnded] = useState(false);
  const [autoplayCancelled, setAutoplayCancelled] = useState(false);
  const [requestedHDRMode, setRequestedHDRMode] = useState<HDRMode>(
    requestedHDRModeRef.current,
  );
  const [hdrFallbackNotice, setHdrFallbackNotice] = useState("");
  const startupBuffering =
    bufferPhase === "preparing" ||
    bufferPhase === "startup" ||
    bufferPhase === "rebuffering";

  const updateBufferPhase = useCallback((phase: BufferPhase) => {
    bufferPhaseRef.current = phase;
    setBufferPhase(phase);
  }, []);

  useEffect(() => {
    sessionRef.current = session;
  }, [session]);

  useEffect(() => {
    recentStallsRef.current = [];
    lowBufferSinceRef.current = null;
    stallStartedAtRef.current = null;
    setStallCount(0);
    setStallDurationMs(0);
    setStabilityNotice("");
    setDiagnosticsOpen(false);
  }, [item.id]);

  useEffect(() => {
    const mode = item.technical.hdrFormat ? storedHDRMode() : "auto";
    requestedHDRModeRef.current = mode;
    setRequestedHDRMode(mode);
    setHdrFallbackNotice("");
  }, [item.id, item.technical.hdrFormat]);

  useEffect(() => {
    if (item.type !== "episode") {
      setPostPlay(null);
      return;
    }
    let cancelled = false;
    void api.media
      .postPlay(item.id)
      .then((value) => {
        if (cancelled) return;
        setPostPlay(value);
        if (value.nextEpisode?.id) preloadPlayback(value.nextEpisode.id);
      })
      .catch(() => {
        if (!cancelled) setPostPlay({ recommendations: [] });
      });
    return () => {
      cancelled = true;
    };
  }, [item.id, item.type]);

  useEffect(() => {
    const authoritative =
      session?.plan.durationMs ??
      options.durationMs ??
      item.technical.durationMs ??
      0;
    if (authoritative > 0) {
      durationMsRef.current = authoritative;
      setDurationSeconds(authoritative / 1000);
    }
  }, [item.technical.durationMs, options.durationMs, session?.plan.durationMs]);

  const bufferPolicy = useMemo(
    () => bufferPolicyForPlan(session?.plan, durationSeconds),
    [durationSeconds, session?.plan],
  );

  useEffect(() => {
    const sessionID = session?.id;
    if (!sessionID || session?.plan.mode === "direct") {
      setRuntimeStats(null);
      return;
    }
    const controller = new AbortController();
    let disposed = false;
    const refresh = async () => {
      try {
        const stats = await api.playback.stats(sessionID, controller.signal);
        if (!disposed) setRuntimeStats(stats);
      } catch {
        // Runtime diagnostics must never interrupt playback.
      }
    };
    void refresh();
    const interval = window.setInterval(() => void refresh(), 2500);
    return () => {
      disposed = true;
      controller.abort();
      window.clearInterval(interval);
    };
  }, [session?.id, session?.plan.mode]);

  useEffect(() => {
    let cancelled = false;
    void api.system
      .configuration()
      .then((configuration) => {
        if (cancelled) return;
        const languages = configuration.settings.subtitles.preferredLanguages;
        setPreferredSubtitleLanguages(languages.length > 0 ? languages : ["en"]);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  const selectedAudioTrack = useMemo(
    () =>
      options.audioTracks.find(
        (track) => track.index === preferences.audioTrackIndex,
      ) ?? options.audioTracks.find((track) => track.default) ?? options.audioTracks[0],
    [options.audioTracks, preferences.audioTrackIndex],
  );
  const audioMenuOptions = useMemo<PlayerTrackMenuOption[]>(
    () =>
      options.audioTracks.map((track, index) => ({
        id: String(track.index),
        title: audioTrackTitle(track, index),
        description: audioTrackDescription(track),
        selected: selectedAudioTrack?.index === track.index,
      })),
    [locale, options.audioTracks, selectedAudioTrack?.index],
  );
  const audioMenuValue = selectedAudioTrack
    ? audioLabel(
        selectedAudioTrack,
        Math.max(0, options.audioTracks.findIndex((track) => track.index === selectedAudioTrack.index)),
      )
    : tr('Helirada');
  const liveCaptionLanguage =
    selectedAudioTrack?.language ?? preferredSubtitleLanguages[0] ?? navigator.language;
  const liveCaptionLanguageName = liveCaptionLanguageLabel(liveCaptionLanguage);

  const selectedSubtitle = useMemo(
    () =>
      options.subtitles.find((track) => track.id === preferences.subtitleId),
    [options.subtitles, preferences.subtitleId],
  );
  const selectedTextTrack =
    selectedSubtitle && !selectedSubtitle.requiresBurnIn
      ? selectedSubtitle
      : undefined;
  const orderedSubtitles = useMemo(
    () => sortSubtitleTracks(options.subtitles, preferredSubtitleLanguages),
    [options.subtitles, preferredSubtitleLanguages],
  );
  const orderedSubtitleResults = useMemo(
    () => sortSubtitleResults(subtitleResults, preferredSubtitleLanguages),
    [preferredSubtitleLanguages, subtitleResults],
  );
  const subtitleMenuValue = selectedSubtitle ? trackLabel(selectedSubtitle) : tr('Väljas');
  const subtitleMenuOptions = useMemo<PlayerTrackMenuOption[]>(
    () => [
      {
        id: "",
        title: tr('Väljas'),
        description: tr('Subtiitreid ei kuvata'),
        selected: !preferences.subtitleId,
      },
      ...orderedSubtitles.map((track) => ({
        id: track.id,
        title: trackLabel(track),
        description: `${track.language?.toUpperCase() || tr('Tundmatu keel')}${track.requiresBurnIn ? ` · ${tr('põletatud pildile')}` : ` · ${tr('lülitatav')}`}`,
        selected: preferences.subtitleId === track.id,
      })),
    ],
    [locale, orderedSubtitles, preferences.subtitleId],
  );

  useEffect(() => {
    if (!liveCaptionEnabled) {
      liveCaptionControllerRef.current?.stop();
      liveCaptionControllerRef.current = null;
      return;
    }

    const video = videoRef.current;
    if (!video) return;
    let cancelled = false;
    let startTimer: number | null = null;

    const start = () => {
      if (cancelled) return;
      liveCaptionControllerRef.current?.stop();
      const controller = createLiveCaptionController(
        video,
        liveCaptionLanguage,
        (snapshot) => {
          if (cancelled) return;
          setLiveCaptionSnapshot(snapshot);
          if (snapshot.state === "error" || snapshot.state === "unsupported") {
            setLiveCaptionEnabled(false);
          }
        },
      );
      liveCaptionControllerRef.current = controller;
      void controller.start();
    };

    if (video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
      start();
    } else {
      const onReady = () => start();
      video.addEventListener("loadeddata", onReady, { once: true });
      startTimer = window.setTimeout(start, 1_500);
      return () => {
        cancelled = true;
        if (startTimer !== null) window.clearTimeout(startTimer);
        video.removeEventListener("loadeddata", onReady);
        liveCaptionControllerRef.current?.stop();
        liveCaptionControllerRef.current = null;
      };
    }

    return () => {
      cancelled = true;
      if (startTimer !== null) window.clearTimeout(startTimer);
      liveCaptionControllerRef.current?.stop();
      liveCaptionControllerRef.current = null;
    };
  }, [liveCaptionEnabled, liveCaptionLanguage, session?.id]);

  const reportProgress = useCallback(
    (force = false, completed = false, keepalive = false, notify = false) => {
      const currentSession = sessionRef.current;
      const position = castProgressRef.current ?? (currentSession
        ? actualPositionMs(videoRef.current, currentSession)
        : lastPositionMs.current);
      const duration = Math.max(
        durationMsRef.current,
        item.technical.durationMs ?? 0,
        item.watch?.durationMs ?? 0,
      );
      if (duration <= 0 || position < 0) return Promise.resolve();
      const now = Date.now();
      if (
        !force &&
        now - lastProgressSentAt.current < 15_000 &&
        Math.abs(position - lastProgressPosition.current) < 15_000
)
        return Promise.resolve();
      lastProgressSentAt.current = now;
      lastProgressPosition.current = position;
      const effectiveCompleted = completed || finishedRef.current;
      return api.watch
        .update(
          item.id,
          { positionMs: position, durationMs: duration, completed: effectiveCompleted },
          keepalive,
        )
        .then(() => {
          if (notify) onWatchUpdated?.();
        })
        .catch(() => undefined);
    },
    [item.id, item.technical.durationMs, item.watch?.durationMs, onWatchUpdated],
  );

  const closePlayer = useCallback(() => {
    void reportProgress(true, false, true, true);
    onClose();
  }, [onClose, reportProgress]);

  const playPostPlayItem = useCallback(
    async (target: MediaItem, completeCurrent = true) => {
      if (completeCurrent) finishedRef.current = true;
      setPlaying(false);
      updateBufferPhase("paused");
      await reportProgress(true, completeCurrent, false, true);
      onPlayItem(target);
    },
    [onPlayItem, reportProgress, updateBufferPhase],
  );

  const cancelAutoplay = useCallback(() => {
    autoplayCancelledRef.current = true;
    setAutoplayCancelled(true);
  }, []);

  const handleEnded = useCallback(async () => {
    finishedRef.current = true;
    setPlaying(false);
    updateBufferPhase("paused");
    setPlaybackEnded(true);
    await reportProgress(true, true, false, true);
  }, [reportProgress, updateBufferPhase]);

  useEffect(() => {
    const nextEpisode = postPlay?.nextEpisode;
    if (
      !playbackEnded ||
      !nextEpisode ||
      autoplayCancelled ||
      advanceStartedRef.current
    )
      return;
    advanceStartedRef.current = true;
    onPlayItem(nextEpisode);
  }, [autoplayCancelled, onPlayItem, playbackEnded, postPlay?.nextEpisode]);

  const stopSession = useCallback((value: PlaybackSession | null) => {
    if (value?.id) void api.playback.stop(value.id).catch(() => undefined);
  }, []);

  const startSession = useCallback(
    async (next: PlaybackPreferences, preserveFallback = false) => {
      const generation = ++generationRef.current;
      lastPositionMs.current = Math.max(0, next.startPositionMs ?? 0);
      setCurrentSeconds(lastPositionMs.current / 1000);
      setSeekDraft(null);
      const previous = sessionRef.current;
      sessionRef.current = null;
      setSession(null);
      setRuntimeStats(null);
      setLoading(true);
      setMediaReady(false);
      resumeAfterBufferRef.current = false;
      stallStartedAtRef.current = null;
      lowBufferSinceRef.current = null;
      updateBufferPhase("preparing");
      setBufferedAhead(0);
      setPlaying(false);
      setError("");
      if (!preserveFallback)
        fallbackUsed.current = Boolean(next.forceTranscode);
      stopSession(previous);
      let activeCapabilities = capabilitiesRef.current;
      if (!activeCapabilities) {
        activeCapabilities = await detectClientCapabilities(item.technical);
        capabilitiesRef.current = activeCapabilities;
        setCapabilities(activeCapabilities);
      }
      try {
        const created = await api.playback.start(item.id, activeCapabilities, next);
        if (generation !== generationRef.current) {
          stopSession(created);
          return;
        }
        sessionRef.current = created;
        setSession(created);
        setPreferences(next);
      } catch (caught) {
        if (
          generation === generationRef.current &&
          item.technical.hdrFormat &&
          next.hdrMode !== "sdr" &&
          !next.forceTranscode
        ) {
          const fallbackPreferences: PlaybackPreferences = {
            ...next,
            hdrMode: "sdr",
            forceTranscode: true,
          };
          setHdrFallbackNotice(
            hdrFallbackMessage(
              errorText(caught, tr('HDR voogu ei saanud käivitada')),
            ),
          );
          try {
            const fallback = await api.playback.start(
              item.id,
              activeCapabilities,
              fallbackPreferences,
            );
            if (generation !== generationRef.current) {
              stopSession(fallback);
              return;
            }
            fallbackUsed.current = true;
            sessionRef.current = fallback;
            setSession(fallback);
            setPreferences(fallbackPreferences);
          } catch (fallbackCaught) {
            if (generation === generationRef.current) {
              setError(
                errorText(
                  fallbackCaught,
                  errorText(caught, tr('Esituse ettevalmistamine ebaõnnestus.')),
                ),
              );
            }
          }
        } else if (generation === generationRef.current) {
          setError(errorText(caught, tr('Esituse ettevalmistamine ebaõnnestus.')));
        }
      } finally {
        if (generation === generationRef.current) {
          setLoading(false);
          stabilityFallbackInFlightRef.current = false;
        }
      }
    },
    [item.id, item.technical, item.technical.hdrFormat, stopSession, updateBufferPhase],
  );

  useEffect(() => {
    let cancelled = false;
    const initialize = async () => {
      setLoading(true);
      setError("");
      try {
        const [loaded, detectedCapabilities] = await Promise.all([
          getPlaybackOptions(item.id),
          detectClientCapabilities(item.technical),
        ]);
        if (cancelled) return;
        capabilitiesRef.current = detectedCapabilities;
        setCapabilities(detectedCapabilities);
        const defaultAudio =
          loaded.audioTracks.find((track) => track.default) ??
          loaded.audioTracks[0];
        const forcedSubtitle =
          loaded.subtitles.find((track) => track.forced && track.default) ??
          loaded.subtitles.find((track) => track.forced);
        const resumePosition =
          item.watch &&
          !item.watch.completed &&
          item.watch.progress >= 0.01 &&
          item.watch.progress <= 0.95
            ? item.watch.positionMs
            : 0;
        const initial: PlaybackPreferences = {
          ...(defaultAudio ? { audioTrackIndex: defaultAudio.index } : {}),
          ...(forcedSubtitle ? { subtitleId: forcedSubtitle.id } : {}),
          ...(resumePosition > 0 ? { startPositionMs: resumePosition } : {}),
          ...(item.technical.hdrFormat
            ? { hdrMode: requestedHDRModeRef.current }
            : {}),
        };
        setOptions(loaded);
        if ((loaded.durationMs ?? 0) > 0)
          setDurationSeconds((loaded.durationMs ?? 0) / 1000);
        setPreferences(initial);
        await startSession(initial);
      } catch (caught) {
        if (!cancelled) {
          setLoading(false);
          setError(errorText(caught, tr('Esitusvalikuid ei saanud laadida.')));
        }
      }
    };
    void initialize();
    return () => {
      cancelled = true;
      generationRef.current += 1;
      stopSession(sessionRef.current);
      sessionRef.current = null;
    };
  }, [item.id, item.watch, startSession, stopSession]);

  const restartWith = useCallback(
    (changes: Partial<PlaybackPreferences>, atMs?: number) => {
      const next: PlaybackPreferences = {
        ...preferences,
        ...changes,
        ...(typeof atMs === "number"
          ? { startPositionMs: Math.max(0, Math.round(atMs)) }
          : {}),
      };
      fallbackUsed.current = Boolean(next.forceTranscode);
      recoveryAttempts.current = 0;
      void startSession(next);
    },
    [preferences, startSession],
  );

  const requestStabilityFallback = useCallback(
    (reason: string) => {
      const current = sessionRef.current;
      if (!current || stabilityFallbackInFlightRef.current) return;
      const currentLevel = Math.max(
        preferences.recoveryLevel ?? 0,
        current.plan.recoveryLevel ?? 0,
      );
      if (currentLevel >= 3) return;
      stabilityFallbackInFlightRef.current = true;
      const nextLevel = currentLevel + 1;
      const position = actualPositionMs(videoRef.current, current);
      const nextLabel =
        nextLevel === 1
          ? tr('stabiilsele 1080p')
          : nextLevel === 2
            ? tr('andmesäästlikule 720p')
            : tr('480p turvarežiimi');
      setStabilityNotice(tr('{{reason}} KoduMedia lülitub {{profile}} profiilile.', { reason, profile: nextLabel }));
      restartWith({ recoveryLevel: nextLevel }, position);
    },
    [preferences.recoveryLevel, restartWith],
  );

  useEffect(() => {
    if (
      !session ||
      session.plan.mode === "direct" ||
      (bufferPhase !== "playing" && bufferPhase !== "rebuffering")
    ) {
      lowBufferSinceRef.current = null;
      return;
    }
    const low = bufferedAhead < bufferPolicy.lowSeconds;
    if (!low) {
      lowBufferSinceRef.current = null;
      return;
    }
    if (lowBufferSinceRef.current === null)
      lowBufferSinceRef.current = Date.now();
    const lowFor = Date.now() - lowBufferSinceRef.current;
    if (
      lowFor >= 6000 &&
      runtimeStats?.fallbackRecommended &&
      (session.plan.recoveryLevel ?? 0) < 3
    ) {
      requestStabilityFallback(
        trExternal(runtimeStats.fallbackReason) || tr('Encoder ei hoidnud piisavat reaalajavaru.'),
      );
    }
  }, [
    bufferPhase,
    bufferPolicy.lowSeconds,
    bufferedAhead,
    requestStabilityFallback,
    runtimeStats?.fallbackReason,
    runtimeStats?.fallbackRecommended,
    session,
  ]);

  const changeHDRMode = useCallback(
    (mode: HDRMode) => {
      window.localStorage.setItem(HDR_MODE_STORAGE_KEY, mode);
      requestedHDRModeRef.current = mode;
      setRequestedHDRMode(mode);
      setHdrFallbackNotice("");
      const position = actualPositionMs(videoRef.current, sessionRef.current);
      restartWith(
        {
          hdrMode: mode,
          forceTranscode: mode === "sdr",
        },
        position,
      );
    },
    [restartWith],
  );

  const handleFatal = useCallback(
    (message: string) => {
      const current = sessionRef.current;
      if (!current) {
        setError(message);
        return;
      }
      const position = actualPositionMs(videoRef.current, current);
      lastPositionMs.current = position;
      if (current.plan.preserveHdr && preferences.hdrMode !== "sdr") {
        fallbackUsed.current = true;
        recoveryAttempts.current = 0;
        setHdrFallbackNotice(hdrFallbackMessage(message));
        void startSession(
          {
            ...preferences,
            hdrMode: "sdr",
            forceTranscode: true,
            startPositionMs: position,
          },
          true,
        );
        return;
      }
      if (!fallbackUsed.current && current.plan.mode !== "transcode") {
        fallbackUsed.current = true;
        void startSession(
          { ...preferences, forceTranscode: true, startPositionMs: position },
          true,
        );
        return;
      }
      if (current.plan.mode === "transcode" && recoveryAttempts.current < 1) {
        recoveryAttempts.current += 1;
        stopSession(current);
        sessionRef.current = null;
        setSession(null);
        void startSession(
          { ...preferences, forceTranscode: true, startPositionMs: position },
          true,
        );
        return;
      }

      // A fatal HLS error must also end the server-side producer. In 0.3.0 the
      // error dialog stayed open while FFmpeg continued encoding the whole film.
      stopSession(current);
      sessionRef.current = null;
      setSession(null);
      setPlaying(false);
      setMediaReady(false);
      setError(
        tr('{{message}} Universaalne serveripoolne transkodeerimine ei käivitunud.', { message }),
      );
    },
    [preferences, startSession, stopSession],
  );
  fatalHandler.current = handleFatal;

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !session) return;
    const source =
      session.plan.mode === "direct"
        ? session.plan.directUrl
        : (session.playlistUrl ?? session.plan.playlistUrl);
    if (!source) {
      setError(tr('Server ei tagastanud esitusallikat.'));
      return;
    }

    let cancelled = false;
    let hls: HlsType | undefined;
    let hlsJsAttached = false;
    let autoPlayStarted = false;
    const startupWaitStartedAt = Date.now();
    let networkRecoveryAttempts = 0;
    let mediaRecoveryAttempts = 0;
    const recoveryTimers = new Set<number>();
    const adaptive = session.plan.mode !== "direct";
    const knownDuration = Math.max(
      0,
      (session.plan.durationMs ??
        options.durationMs ??
        item.technical.durationMs ??
        0) / 1000,
    );
    const sessionBufferPolicy = bufferPolicyForPlan(session.plan, knownDuration);
    const targetBuffer = sessionBufferPolicy.startupSeconds;
    const rebufferTarget = sessionBufferPolicy.rebufferSeconds;
    const mediaElementInitialSeconds = Math.max(
      0,
      session.plan.mode === "direct"
        ? (session.plan.startPositionMs ?? 0) / 1000
        : (session.plan.initialPositionMs ?? 0) / 1000,
    );

    const startWhenReady = () => {
      if (cancelled) return;
      setMediaReady(true);
      const ahead = bufferedAheadSeconds(video);
      setBufferedAhead(ahead);
      if (!adaptive) {
        if (Number.isFinite(video.duration) && video.duration > 0)
          setDurationSeconds(video.duration);
        updateBufferPhase("ready");
        if (!autoPlayStarted) {
          autoPlayStarted = true;
          void video.play().catch(() => undefined);
        }
        return;
      }
      if (!autoPlayStarted) {
        updateBufferPhase("startup");
        if (shouldStartBufferedPlayback(
          ahead,
          targetBuffer,
          Date.now() - startupWaitStartedAt,
          video.readyState,
        )) {
          autoPlayStarted = true;
          updateBufferPhase("ready");
          void video.play().catch(() => undefined);
        }
        return;
      }
      if (
        bufferPhaseRef.current === "rebuffering" &&
        resumeAfterBufferRef.current &&
        shouldStartBufferedPlayback(
          ahead,
          rebufferTarget,
          Date.now() - (stallStartedAtRef.current ?? Date.now()),
          video.readyState,
        )
      ) {
        resumeAfterBufferRef.current = false;
        updateBufferPhase("ready");
        void video.play().catch(() => undefined);
      }
    };

    const beginRebuffer = () => {
      if (
        cancelled ||
        !adaptive ||
        !autoPlayStarted ||
        bufferPhaseRef.current === "rebuffering"
      )
        return;
      const now = Date.now();
      recentStallsRef.current = recentStallsRef.current
        .filter((value) => now - value < 90_000)
        .concat(now);
      setStallCount((value) => value + 1);
      stallStartedAtRef.current = now;
      resumeAfterBufferRef.current = true;
      updateBufferPhase("rebuffering");
      video.pause();
      if (
        recentStallsRef.current.length >= 2 &&
        (session.plan.recoveryLevel ?? 0) < 3
      ) {
        requestStabilityFallback(
          tr('Esituspuhver tühjenes lühikese aja jooksul korduvalt.'),
        );
      }
    };

    const finishRebuffer = () => {
      if (stallStartedAtRef.current !== null) {
        setStallDurationMs(
          (value) =>
            value +
            Math.max(
              0,
              Date.now() - (stallStartedAtRef.current ?? Date.now()),
            ),
        );
        stallStartedAtRef.current = null;
      }
      resumeAfterBufferRef.current = false;
      updateBufferPhase("playing");
    };

    const onLoaded = () => {
      if (cancelled) return;
      if (!hls && mediaElementInitialSeconds > 0) {
        video.currentTime = Math.min(
          video.duration || Number.POSITIVE_INFINITY,
          mediaElementInitialSeconds,
        );
      }
      startWhenReady();
    };
    const onProgress = () => startWhenReady();
    const onCanPlay = () => startWhenReady();
    const onWaiting = () => beginRebuffer();
    const onStalled = () => beginRebuffer();
    const onPlaying = () => finishRebuffer();
    const bufferTimer = window.setInterval(startWhenReady, 500);
    const onNativeError = () => {
      if (cancelled) return;
      // hls.js owns errors after it attaches MediaSource. Its ERROR event has
      // the actionable network/media detail and recovery policy; reacting to
      // the media element's generic error as well can race that handler and
      // start duplicate fallback sessions.
      if (hlsJsAttached) return;
      const code = video.error?.code;
      fatalHandler.current(
        code
          ? `Brauser katkestas esituse (MediaError ${code}).`
          : tr('Brauser ei saanud meediavoogu avada.'),
      );
    };
    video.addEventListener("loadedmetadata", onLoaded);
    video.addEventListener("progress", onProgress);
    video.addEventListener("canplay", onCanPlay);
    video.addEventListener("waiting", onWaiting);
    video.addEventListener("stalled", onStalled);
    video.addEventListener("playing", onPlaying);
    video.addEventListener("error", onNativeError);

    const attach = async () => {
      const nativeHlsSupported =
        video.canPlayType("application/vnd.apple.mpegurl") !== "";
      const webKitAirPlaySupported =
        typeof (
          video as HTMLVideoElement & {
            webkitShowPlaybackTargetPicker?: () => void;
          }
        ).webkitShowPlaybackTargetPicker === "function";
      // Safari must own HLS playback whenever it supports HLS natively. If
      // hls.js attaches first, the media element receives a blob: URL that is
      // valid only inside this tab; an AirPlay television can connect to the
      // route but cannot fetch or play that private browser URL.
      if (
        shouldUseNativeHls(
          adaptive,
          nativeHlsSupported,
          webKitAirPlaySupported,
        )
      ) {
        video.src = source;
        video.load();
        return;
      }

      if (adaptive && typeof MediaSource !== "undefined") {
        try {
          const module = await import("hls.js");
          if (cancelled) return;
          const Hls = module.default;
          if (Hls.isSupported()) {
            hls = new Hls({
              enableWorker: true,
              lowLatencyMode: false,
              backBufferLength: 45,
              maxBufferLength: Math.max(30, sessionBufferPolicy.desiredSeconds),
              maxMaxBufferLength: Math.max(
                60,
                sessionBufferPolicy.desiredSeconds * 2,
              ),
              maxBufferSize: hlsBufferSizeBytes(
                session.plan.maxVideoBitrate || item.technical.bitrate,
                sessionBufferPolicy.desiredSeconds,
              ),
              startFragPrefetch: true,
              fragLoadingMaxRetry: 6,
              fragLoadingRetryDelay: 250,
              fragLoadingMaxRetryTimeout: 3000,
              manifestLoadingMaxRetry: 4,
              manifestLoadingRetryDelay: 250,
              startPosition:
                mediaElementInitialSeconds > 0
                  ? mediaElementInitialSeconds
                  : -1,
            });
            hls.loadSource(source);
            hlsJsAttached = true;
            hls.attachMedia(video);
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
              if (!cancelled) {
                setMediaReady(true);
                startWhenReady();
              }
            });
            hls.on(Hls.Events.FRAG_BUFFERED, () => {
              if (!cancelled) startWhenReady();
            });
            hls.on(Hls.Events.ERROR, (_event, data) => {
              if (cancelled || !data.fatal) return;

              const details = String(data.details || "unknownError");
              const failure = describeHlsFailure(String(data.type), details);
              const unsupported = isClearlyUnsupportedHlsDetail(details);

              if (
                data.type === Hls.ErrorTypes.NETWORK_ERROR &&
                networkRecoveryAttempts < 2
              ) {
                networkRecoveryAttempts += 1;
                const timer = window.setTimeout(() => {
                  recoveryTimers.delete(timer);
                  if (!cancelled) hls?.startLoad();
                }, 250 * networkRecoveryAttempts);
                recoveryTimers.add(timer);
                return;
              }

              if (
                data.type === Hls.ErrorTypes.MEDIA_ERROR &&
                !unsupported &&
                mediaRecoveryAttempts < 1
              ) {
                mediaRecoveryAttempts += 1;
                try {
                  hls?.recoverMediaError();
                  return;
                } catch {
                  // Continue into the normal fallback below.
                }
              }

              fatalHandler.current(failure);
            });
            return;
          }
        } catch (caught) {
          if (!cancelled)
            fatalHandler.current(
              errorText(caught, tr('HLS-mängijat ei saanud laadida.')),
            );
          return;
        }
      }
      video.src = source;
      video.load();
    };

    void attach();
    return () => {
      cancelled = true;
      recoveryTimers.forEach((timer) => window.clearTimeout(timer));
      recoveryTimers.clear();
      hls?.destroy();
      video.removeEventListener("loadedmetadata", onLoaded);
      video.removeEventListener("progress", onProgress);
      window.clearInterval(bufferTimer);
      video.removeEventListener("canplay", onCanPlay);
      video.removeEventListener("waiting", onWaiting);
      video.removeEventListener("stalled", onStalled);
      video.removeEventListener("playing", onPlaying);
      video.removeEventListener("error", onNativeError);
      video.pause();
      video.removeAttribute("src");
      video.load();
    };
  }, [
    item.technical.bitrate,
    item.technical.durationMs,
    options.durationMs,
    requestStabilityFallback,
    session,
    updateBufferPhase,
  ]);

  useEffect(() => {
    const controller = new AbortController();
    setSubtitleCues([]);
    setSubtitleLoadError("");
    if (!selectedTextTrack?.url) {
      setSubtitleLoading(false);
      return () => controller.abort();
    }

    setSubtitleLoading(true);
    void api.playback
      .loadSubtitle(selectedTextTrack.url, controller.signal)
      .then((content) => {
        if (controller.signal.aborted) return;
        const cues = parseWebVtt(content);
        if (cues.length === 0) {
          throw new Error(
            tr('Subtiitrifail avanes, kuid ei sisaldanud loetavaid ajakoode.'),
          );
        }
        setSubtitleCues(cues);
      })
      .catch((caught) => {
        if (!controller.signal.aborted) {
          setSubtitleLoadError(
            errorText(caught, tr('Subtiitreid ei saanud laadida.')),
          );
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setSubtitleLoading(false);
      });

    return () => controller.abort();
  }, [selectedTextTrack?.id, selectedTextTrack?.url]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    video.volume = volume;
    video.muted = muted;
  }, [muted, volume]);

  const togglePlayback = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    if (
      bufferPhaseRef.current === "preparing" ||
      bufferPhaseRef.current === "startup" ||
      bufferPhaseRef.current === "rebuffering"
    )
      return;
    if (video.paused) void video.play().catch(() => undefined);
    else video.pause();
  }, []);

  const seekTo = useCallback(
    (seconds: number) => {
      const clamped = Math.max(
        0,
        Math.min(durationSeconds || seconds, seconds),
      );
      setSeekDraft(null);
      const current = sessionRef.current;
      const video = videoRef.current;
      if (!current || !video) return;
      if (current.plan.mode === "direct") {
        video.currentTime = clamped;
        setCurrentSeconds(clamped);
        updateMediaSessionPosition({
          durationSeconds,
          positionSeconds: clamped,
          playing,
          playbackRate: video.playbackRate || 1,
          force: true,
        });
        return;
      }
      const sessionBaseSeconds = Math.max(0, (current.plan.startPositionMs ?? 0) / 1000);
      const localTarget = clamped - sessionBaseSeconds;
      if (localTarget >= 0 && isBufferedAt(video, localTarget)) {
        video.currentTime = localTarget;
        setCurrentSeconds(clamped);
        updateMediaSessionPosition({
          durationSeconds,
          positionSeconds: clamped,
          playing,
          playbackRate: video.playbackRate || 1,
          force: true,
        });
        return;
      }
      // Re-encoding after a remote seek gives FFmpeg an exact audio/video
      // timestamp reset. Stream-copy seeks can start on different keyframes.
      restartWith({ forceTranscode: true }, clamped * 1000);
    },
    [durationSeconds, playing, restartWith],
  );

  const toggleActivePlayback = useCallback(() => {
    if (activeRemote) {
      casting.togglePlayback();
      return;
    }
    togglePlayback();
  }, [activeRemote, casting.togglePlayback, togglePlayback]);

  const seekActivePlayback = useCallback((seconds: number) => {
    if (activeRemote) {
      casting.seek(Math.max(0, seconds * 1000));
      setSeekDraft(null);
      return;
    }
    seekTo(seconds);
  }, [activeRemote, casting.seek, seekTo]);

  const toggleActiveMuted = useCallback(() => {
    const next = !effectiveMuted;
    if (activeRemote?.canSetVolume) casting.setMuted(next);
    setMuted(next);
  }, [activeRemote?.canSetVolume, casting.setMuted, effectiveMuted]);

  const changeActiveVolume = useCallback((next: number) => {
    if (activeRemote?.canSetVolume) casting.setVolume(next);
    setVolume(next);
    setMuted(next === 0);
  }, [activeRemote?.canSetVolume, casting.setVolume]);


  useEffect(() =>
    installMediaSession(item, {
      play: () => {
        if (activeRemote) casting.play();
        else void videoRef.current?.play().catch(() => undefined);
      },
      pause: () => {
        if (activeRemote) casting.pause();
        else videoRef.current?.pause();
      },
      seekTo: seekActivePlayback,
      seekBy: (offsetSeconds) =>
        seekActivePlayback(effectiveCurrentSeconds + offsetSeconds),
      next: postPlay?.nextEpisode
        ? () => {
            void playPostPlayItem(postPlay.nextEpisode as MediaItem, true);
          }
        : undefined,
    }),
  [activeRemote, casting.pause, casting.play, effectiveCurrentSeconds, item, playPostPlayItem, postPlay?.nextEpisode, seekActivePlayback]);

  useEffect(() => {
    updateMediaSessionPosition({
      durationSeconds: effectiveDurationSeconds,
      positionSeconds: effectiveCurrentSeconds,
      playing: effectivePlaying,
      playbackRate: videoRef.current?.playbackRate ?? 1,
      force: true,
    });
  }, [effectiveCurrentSeconds, effectiveDurationSeconds, effectivePlaying, session?.id]);

  useEffect(() => {
    const keydown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.matches("input, select, textarea, button")) return;
      switch (event.key.toLowerCase()) {
        case "escape":
          event.preventDefault();
          if (subtitlePanel) setSubtitlePanel(false);
          else if (castMenuOpen) setCastMenuOpen(false);
          else if (liveCaptionMenuOpen) setLiveCaptionMenuOpen(false);
          else if (subtitleMenuOpen) setSubtitleMenuOpen(false);
          else if (audioMenuOpen) setAudioMenuOpen(false);
          else closePlayer();
          break;
        case " ":
        case "k":
          event.preventDefault();
          toggleActivePlayback();
          break;
        case "arrowleft":
          event.preventDefault();
          seekActivePlayback(effectiveCurrentSeconds - 10);
          break;
        case "arrowright":
          event.preventDefault();
          seekActivePlayback(effectiveCurrentSeconds + 10);
          break;
        case "f":
          event.preventDefault();
          void shellRef.current?.requestFullscreen?.();
          break;
        case "m":
          event.preventDefault();
          toggleActiveMuted();
          break;
      }
      revealControls();
    };
    window.addEventListener("keydown", keydown);
    return () => window.removeEventListener("keydown", keydown);
  }, [
    audioMenuOpen,
    castMenuOpen,
    closePlayer,
    effectiveCurrentSeconds,
    liveCaptionMenuOpen,
    revealControls,
    seekActivePlayback,
    subtitleMenuOpen,
    subtitlePanel,
    toggleActiveMuted,
    toggleActivePlayback,
  ]);

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  useEffect(() => {
    const flush = () => reportProgress(true, false, true);
    window.addEventListener("pagehide", flush);
    return () => {
      window.removeEventListener("pagehide", flush);
      flush();
    };
  }, [reportProgress]);

  const changeAudioMenu = useCallback((open: boolean) => {
    if (open) {
      setSubtitleMenuOpen(false);
      setLiveCaptionMenuOpen(false);
      setCastMenuOpen(false);
    }
    setAudioMenuOpen(open);
  }, []);

  const changeSubtitleMenu = useCallback((open: boolean) => {
    if (open) {
      setAudioMenuOpen(false);
      setLiveCaptionMenuOpen(false);
      setCastMenuOpen(false);
    }
    setSubtitleMenuOpen(open);
  }, []);

  const changeLiveCaptionMenu = useCallback((open: boolean) => {
    if (open) {
      setAudioMenuOpen(false);
      setSubtitleMenuOpen(false);
      setCastMenuOpen(false);
    }
    setLiveCaptionMenuOpen(open);
  }, []);

  const changeCastMenu = useCallback((open: boolean) => {
    if (open) {
      setAudioMenuOpen(false);
      setSubtitleMenuOpen(false);
      setLiveCaptionMenuOpen(false);
      setSubtitlePanel(false);
    }
    setCastMenuOpen(open);
  }, []);

  const toggleLiveCaptions = useCallback(() => {
    setLiveCaptionEnabled((value) => {
      if (value) setLiveCaptionSnapshot({ state: "idle", text: "" });
      return !value;
    });
  }, []);

  const selectAudioTrack = useCallback(
    (trackID: string) => {
      setAudioMenuOpen(false);
      const audioTrackIndex = Number(trackID);
      if (!Number.isFinite(audioTrackIndex)) return;
      if (selectedAudioTrack?.index === audioTrackIndex) return;
      restartWith(
        { audioTrackIndex },
        actualPositionMs(videoRef.current, sessionRef.current),
      );
    },
    [restartWith, selectedAudioTrack?.index],
  );

  const selectSubtitle = useCallback(
    (subtitleId?: string) => {
      setAudioMenuOpen(false);
      setSubtitleMenuOpen(false);
      setLiveCaptionMenuOpen(false);
      const currentTrack = options.subtitles.find(
        (track) => track.id === preferences.subtitleId,
      );
      const nextTrack = options.subtitles.find(
        (track) => track.id === subtitleId,
      );
      const position = actualPositionMs(videoRef.current, sessionRef.current);
      if (currentTrack?.requiresBurnIn || nextTrack?.requiresBurnIn) {
        restartWith({ subtitleId }, position);
        return;
      }
      setPreferences((current) => ({ ...current, subtitleId }));
    },
    [options.subtitles, preferences.subtitleId, restartWith],
  );

  const searchSubtitles = async () => {
    setAudioMenuOpen(false);
    setSubtitleMenuOpen(false);
    setLiveCaptionMenuOpen(false);
    setSubtitlePanel(true);
    setSearchingSubtitles(true);
    setSubtitleSearchError("");
    try {
      setSubtitleResults(await api.playback.searchSubtitles(item.id));
    } catch (caught) {
      setSubtitleSearchError(
        errorText(caught, tr('OpenSubtitles otsing ebaõnnestus.')),
      );
    } finally {
      setSearchingSubtitles(false);
    }
  };

  const downloadSubtitle = async (result: SubtitleSearchResult) => {
    setDownloadingSubtitle(result.fileId);
    setSubtitleSearchError("");
    try {
      const track = await api.playback.downloadSubtitle(item.id, result);
      setOptions((current) => ({
        ...current,
        subtitles: [
          ...current.subtitles.filter((value) => value.id !== track.id),
          track,
        ],
      }));
      setSubtitlePanel(false);
      if (track.requiresBurnIn)
        restartWith(
          { subtitleId: track.id },
          actualPositionMs(videoRef.current, sessionRef.current),
        );
      else setPreferences((current) => ({ ...current, subtitleId: track.id }));
    } catch (caught) {
      setSubtitleSearchError(
        errorText(caught, tr('Subtiitri allalaadimine ebaõnnestus.')),
      );
    } finally {
      setDownloadingSubtitle(null);
    }
  };

  const toggleFullscreen = async () => {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await shellRef.current?.requestFullscreen?.();
    } catch {
      /* Fullscreen can be denied outside a user gesture. */
    }
  };

  const togglePiP = async () => {
    const video = videoRef.current;
    if (!video || !document.pictureInPictureEnabled) return;
    try {
      if (document.pictureInPictureElement) {
        await document.exitPictureInPicture();
      } else {
        updateMediaSessionPosition({
          durationSeconds,
          positionSeconds: sessionRef.current
            ? actualPositionMs(video, sessionRef.current) / 1000
            : currentSeconds,
          playing: !video.paused,
          playbackRate: video.playbackRate || 1,
          force: true,
        });
        await video.requestPictureInPicture();
      }
    } catch {
      /* PiP availability depends on the active stream and browser. */
    }
  };

  const activeCues = useMemo(
    () => activeRemote ? [] : activeSubtitleCues(subtitleCues, currentSeconds * 1000),
    [activeRemote, currentSeconds, subtitleCues],
  );

  const nextEpisode = postPlay?.nextEpisode;
  const remainingSeconds = Math.max(0, effectiveDurationSeconds - effectiveCurrentSeconds);
  const postPlayWindow = postPlayWindowSeconds(effectiveDurationSeconds);
  const showNextEpisodePreview = Boolean(
    item.type === "episode" &&
      nextEpisode &&
      effectiveCurrentSeconds > 0 &&
      remainingSeconds <= postPlayWindow,
  );
  const nextEpisodeProgress = showNextEpisodePreview
    ? Math.min(100, Math.max(0, ((postPlayWindow - remainingSeconds) / postPlayWindow) * 100))
    : 0;
  const showRecommendations = Boolean(
    item.type === "episode" &&
      playbackEnded &&
      postPlay !== null &&
      !nextEpisode,
  );
  const hasHDRSource = Boolean(
    item.technical.hdrFormat || session?.plan.sourceHdrFormat,
  );
  const selectedHDRMode: HDRMode = requestedHDRMode;
  const hdrFallbackActive = Boolean(
    hasHDRSource &&
      requestedHDRMode !== "sdr" &&
      session?.plan.toneMap,
  );

  const bufferTarget =
    bufferPhase === "rebuffering"
      ? bufferPolicy.rebufferSeconds
      : bufferPolicy.startupSeconds;
  const shownBufferSeconds = Math.min(
    Math.max(0, Math.floor(bufferedAhead)),
    Math.max(0, Math.ceil(bufferTarget)),
  );
  const startupText =
    loading || !session || bufferPhase === "preparing"
      ? tr('Server kontrollib radu, HDR-i ja codec’eid ning valmistab esituse ette…')
      : !mediaReady
        ? tr('Server loob brauserile esitusvoogu…')
        : bufferPhase === "rebuffering"
          ? tr('Taastan esituspuhvrit {{current}} / {{target}} s…', { current: shownBufferSeconds, target: Math.ceil(bufferPolicy.rebufferSeconds) })
          : tr('Puhverdan esituse algust {{current}} / {{target}} s…', { current: shownBufferSeconds, target: Math.ceil(bufferPolicy.startupSeconds) });
  const activePlan = session?.plan;
  const videoPathLabel =
    activePlan?.videoAction === "copy"
      ? tr('Video muutmata')
      : activePlan?.videoAction === "transcode"
        ? tr('Video transkodeeritud')
        : tr('Helisisu');
  const audioPathLabel = activePlan
    ? activePlan.audioAction === "transcode"
      ? `${item.technical.audioCodec?.toUpperCase() || tr('Heli')} → ${activePlan.audioCodec?.toUpperCase() || 'AAC'}`
      : activePlan.audioCodec
        ? tr('{{codec}} muutmata', { codec: activePlan.audioCodec.toUpperCase() })
        : undefined
    : undefined;
  const qualityStatusTitle = activePlan
    ? `${profileLabel(activePlan.qualityProfile)} · ${videoPathLabel}`
    : tr('Esitusinfo');
  const qualityStatusDetails = activePlan
    ? [
        resolutionLabel(activePlan, item),
        activePlan.videoCodec?.toUpperCase(),
        activePlan.videoAction === "transcode" ? encoderLabel(activePlan) : undefined,
        activePlan.videoAction === "transcode" ? `${tr('Decode:')} ${decoderLabel(activePlan)}` : undefined,
        audioPathLabel,
        activePlan.preserveHdr ? "HDR" : activePlan.toneMap ? "HDR→SDR" : "SDR",
        activePlan.cacheHit ? tr('Optimeeritud cache') : undefined,
      ]
        .filter(Boolean)
        .join(" · ")
    : "";
  const liveCaptionStatusText =
    liveCaptionSnapshot.state === "starting"
      ? `${tr('Käivitan')} · ${liveCaptionLanguageName}`
      : liveCaptionSnapshot.state === "active"
        ? `${liveCaptionLanguageName} · ${liveCaptionSnapshot.usingLocalRecognition ? tr('seadmesisene') : tr("Chrome'i kõnetuvastus")}`
        : liveCaptionSnapshot.error || `${tr('Väljas')} · ${liveCaptionLanguageName}`;

  return (
    <div
      className="player-layer"
      role="dialog"
      aria-modal="true"
      aria-label={tr('Esitan: {{title}}', { title: item.title })}
    >
      <div
        ref={shellRef}
        className={`player-shell ${controlsVisible || !effectivePlaying ? "player-shell--controls" : ""}`}
        onPointerMove={revealControls}
        onPointerDown={revealControls}
      >
        <video
          ref={videoRef}
          className="player-video"
          playsInline
          disableRemotePlayback={false}
          crossOrigin="anonymous"
          poster={item.metadata.backdropUrl || item.metadata.posterUrl}
          onClick={toggleActivePlayback}
          onPlay={() => setPlaying(true)}
          onPlaying={() => setPlaying(true)}
          onPause={() => {
            setPlaying(false);
            if (
              bufferPhaseRef.current !== "rebuffering" &&
              bufferPhaseRef.current !== "startup" &&
              bufferPhaseRef.current !== "preparing"
            )
              updateBufferPhase("paused");
            reportProgress(true);
          }}
          onEnded={() => { void handleEnded(); }}
          onTimeUpdate={(event) => {
            const position = actualPositionMs(
              event.currentTarget,
              sessionRef.current,
            );
            lastPositionMs.current = position;
            setCurrentSeconds(position / 1000);
            reportProgress(false);
          }}
          onDurationChange={(event) => {
            if (
              sessionRef.current?.plan.mode === "direct" &&
              Number.isFinite(event.currentTarget.duration) &&
              event.currentTarget.duration > 0
            ) {
              setDurationSeconds(event.currentTarget.duration);
            }
          }}
        />

        {activeCues.length > 0 ? (
          <div
            className={`player-subtitle-layer ${controlsVisible ? "player-subtitle-layer--controls" : ""} ${liveCaptionEnabled && liveCaptionSnapshot.text ? "player-subtitle-layer--with-live-caption" : ""}`}
            aria-live="off"
            aria-label={tr('Subtiitrid')}
          >
            {activeCues.map((cue, cueIndex) => (
              <div
                className="player-subtitle-cue"
                dir="auto"
                key={`${cue.id ?? cue.startMs}-${cue.endMs}-${cueIndex}`}
              >
                {cue.text.split("\n").map((line, lineIndex) => (
                  <span key={`${lineIndex}-${line}`}>{line || "\u00a0"}</span>
                ))}
              </div>
            ))}
          </div>
        ) : null}

        {liveCaptionEnabled && liveCaptionSnapshot.text ? (
          <div
            className={`player-live-caption-layer ${controlsVisible ? "player-live-caption-layer--controls" : ""}`}
            aria-live="polite"
            aria-label="Live Caption"
          >
            <span dir="auto">{liveCaptionSnapshot.text}</span>
          </div>
        ) : null}

        <PlayerPostPlay
          nextEpisode={nextEpisode}
          showNextEpisodePreview={showNextEpisodePreview}
          playbackEnded={playbackEnded}
          autoplayCancelled={autoplayCancelled}
          remainingSeconds={remainingSeconds}
          nextEpisodeProgress={nextEpisodeProgress}
          showRecommendations={showRecommendations}
          postPlay={postPlay}
          onPlayNext={(next) => { void playPostPlayItem(next) }}
          onCancelAutoplay={cancelAutoplay}
          onPlayRecommendation={(recommendation) => { void playPostPlayItem(recommendation, false) }}
          onClose={closePlayer}
        />

        <div
          className="player-topbar"
          aria-hidden={!controlsVisible && effectivePlaying}
        >
          <div className="player-topbar__identity">
            <span className="player-kicker">
              {item.type === "episode" ? tr('Seriaal') : item.type === "movie" ? tr('Film') : tr('Kodu')}
            </span>
            <strong>
              {item.type === "episode" && item.seriesTitle
                ? item.seriesTitle
                : item.title}
            </strong>
            {item.type === "episode" ? <small>{item.title}</small> : null}
          </div>
          <Button
            compact
            variant="ghost"
            icon="close"
            onClick={closePlayer}
            aria-label={tr('Sulge mängija')}
          />
        </div>

        {error ? (
          <div className="player-message player-message--error">
            <Icon name="alert" size={30} />
            <h2>{tr('Esitus ei käivitunud')}</h2>
            <p>{error}</p>
            <div className="player-message__actions">
              <Button
                variant="primary"
                icon="refresh"
                onClick={() => {
                  recoveryAttempts.current = 0;
                  void startSession({
                    ...preferences,
                    forceTranscode: true,
                    startPositionMs: sessionRef.current
                      ? actualPositionMs(videoRef.current, sessionRef.current)
                      : lastPositionMs.current,
                  });
                }}
              >
                {tr('Proovi universaalset voogu')}
              </Button>
              <Button variant="secondary" onClick={closePlayer}>
                {tr('Sulge')}
              </Button>
            </div>
          </div>
        ) : null}

        {!error && (loading || !mediaReady || startupBuffering) ? (
          <div className="player-startup-note" role="status" aria-live="polite">
            <Spinner label={tr('Puhverdan')} />
            <span>{startupText}</span>
          </div>
        ) : null}

        {!error &&
        selectedTextTrack &&
        (subtitleLoading || subtitleLoadError) ? (
          <div
            className={`player-subtitle-note ${subtitleLoadError ? "player-subtitle-note--error" : ""}`}
            role="status"
            aria-live="polite"
          >
            {subtitleLoading ? (
              <Spinner label={tr('Laadin subtiitreid')} />
            ) : (
              <Icon name="alert" size={17} />
            )}
            <span>
              {subtitleLoadError ||
                tr('Laadin subtiitreid: {{track}}…', { track: trackLabel(selectedTextTrack) })}
            </span>
          </div>
        ) : null}

        {!loading && !error && mediaReady && !startupBuffering && !effectivePlaying ? (
          <button
            className="player-center-play"
            type="button"
            onClick={toggleActivePlayback}
            aria-label={tr('Esita')}
          >
            <Icon name="play" size={42} />
          </button>
        ) : null}

        <PlayerControls
          visible={controlsVisible}
          playing={effectivePlaying}
          currentSeconds={effectiveCurrentSeconds}
          durationSeconds={effectiveDurationSeconds}
          bufferedAhead={activeRemote ? 0 : bufferedAhead}
          seekDraft={seekDraft}
          muted={effectiveMuted}
          volume={effectiveVolume}
          onSeekDraftChange={setSeekDraft}
          onSeek={seekActivePlayback}
          onTogglePlayback={toggleActivePlayback}
          onToggleMuted={toggleActiveMuted}
          onVolumeChange={changeActiveVolume}
          pictureInPictureAvailable={Boolean(capabilities?.supportsPip && !activeRemote)}
          onPictureInPicture={() => { void togglePiP() }}
          onFullscreen={() => { void toggleFullscreen() }}
          trackMenus={(
            <>
              {!activeRemote ? <>
              {options.audioTracks.length > 1 ? (
                <PlayerTrackMenu
                  icon="volume"
                  ariaLabel={tr('Helirada: {{track}}', { track: audioMenuValue })}
                  title={tr('Helirada')}
                  description={tr('Vali faili sisse pakitud helirada. Vahetamine jätkab samast kohast.')}
                  open={audioMenuOpen}
                  active={options.audioTracks.length > 1}
                  onOpenChange={changeAudioMenu}
                  options={audioMenuOptions}
                  onSelect={selectAudioTrack}
                />
              ) : null}
              <PlayerTrackMenu
                icon="captions"
                ariaLabel={tr('Subtiitrid: {{track}}', { track: subtitleMenuValue })}
                title={tr('Subtiitrid')}
                description={tr('Vali olemasolev rada või otsi uus OpenSubtitlesest.')}
                open={subtitleMenuOpen}
                active={Boolean(preferences.subtitleId)}
                onOpenChange={changeSubtitleMenu}
                options={subtitleMenuOptions}
                onSelect={(subtitleID) => selectSubtitle(subtitleID || undefined)}
                footerAction={{
                  title: tr('Otsi OpenSubtitlesest'),
                  description: tr('Lae uus subtiitrifail otse KODU serverisse'),
                  icon: 'download',
                  onClick: () => { void searchSubtitles() },
                }}
              />
              <PlayerTrackMenu
                icon="liveCaptions"
                ariaLabel={tr('Live Caption: {{state}}', { state: liveCaptionEnabled ? tr('sees') : tr('väljas') })}
                title={tr('Live Caption')}
                description={tr('Reaalajas kõnetuvastus playeri aktiivsest helirajast')}
                open={liveCaptionMenuOpen}
                active={liveCaptionEnabled}
                onOpenChange={changeLiveCaptionMenu}
                options={[]}
                onSelect={() => undefined}
                extraContent={(
                  <div className="player-live-caption-control">
                    <div className="player-live-caption-control__copy">
                      <strong>{liveCaptionEnabled ? tr('Live Caption on sees') : tr('Live Caption on väljas')}</strong>
                      <small>{liveCaptionStatusText}</small>
                    </div>
                    <button
                      type="button"
                      className={`player-live-caption-switch ${liveCaptionEnabled ? 'player-live-caption-switch--active' : ''}`}
                      role="switch"
                      aria-checked={liveCaptionEnabled}
                      onClick={toggleLiveCaptions}
                    >
                      <span />
                    </button>
                    <p>{tr("KoduMedia annab video heliraja Chrome'i kõnetuvastusele. Kui seadmesisene keelemudel pole saadaval, võib Chrome kasutada oma veebipõhist teenust.")}</p>
                  </div>
                )}
              />
              </> : null}
              <PlayerCastMenu
                open={castMenuOpen}
                onOpenChange={changeCastMenu}
                devices={casting.devices}
                discovering={casting.discovering}
                connecting={casting.connecting}
                error={casting.error}
                systemSupported={casting.systemSupported}
                systemAvailable={casting.systemAvailable}
                systemConnected={casting.systemConnected}
                google={casting.google}
                googleActive={casting.googleActive}
                dlnaSession={casting.dlnaSession}
                activeRemote={activeRemote}
                onRefresh={() => { void casting.refreshDevices() }}
                onSystemPicker={() => { void casting.openSystemPicker() }}
                onGoogleCast={() => {
                  setCastMenuOpen(false)
                  void casting.startGoogleCast()
                }}
                onDLNA={(device) => {
                  setCastMenuOpen(false)
                  void casting.startDLNA(device)
                }}
                onStop={() => { void casting.stop() }}
              />
            </>
          )}
          diagnostics={diagnosticsOpen && session && !startupBuffering ? (
            <PlayerDiagnostics
              session={session}
              item={item}
              videoPathLabel={videoPathLabel}
              audioPathLabel={audioPathLabel}
              bufferedAhead={bufferedAhead}
              desiredBufferSeconds={bufferPolicy.desiredSeconds}
              runtimeStats={runtimeStats}
              stallCount={stallCount}
              stallDurationMs={stallDurationMs}
              notice={stabilityNotice || hdrFallbackNotice || trExternal(session.plan.encoderFallbackReason) || trExternal(session.plan.reason)}
            />
          ) : null}
          status={session && !startupBuffering ? (
            <PlayerPlaybackStatus
              session={session}
              diagnosticsOpen={diagnosticsOpen}
              qualityStatusTitle={qualityStatusTitle}
              qualityStatusDetails={qualityStatusDetails}
              notice={stabilityNotice || hdrFallbackNotice}
              hasHDRSource={hasHDRSource}
              hdrFallbackActive={hdrFallbackActive}
              selectedHDRMode={selectedHDRMode}
              onToggleDiagnostics={() => setDiagnosticsOpen((value) => !value)}
              onHDRModeChange={changeHDRMode}
            />
          ) : null}
        />

        {subtitlePanel ? (
          <PlayerSubtitleSearchPanel
            results={orderedSubtitleResults}
            searching={searchingSubtitles}
            error={subtitleSearchError}
            downloadingFileId={downloadingSubtitle}
            onClose={() => setSubtitlePanel(false)}
            onDownload={(result) => { void downloadSubtitle(result) }}
            matchLabel={subtitleMatchLabel}
          />
        ) : null}
      </div>
    </div>
  );
}
