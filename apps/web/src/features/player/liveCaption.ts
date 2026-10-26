import { getLocale, tr } from '../../i18n/i18n'
export type LiveCaptionState =
  | "idle"
  | "starting"
  | "active"
  | "stopped"
  | "unsupported"
  | "error";

export interface LiveCaptionSnapshot {
  state: LiveCaptionState;
  text: string;
  error?: string;
  usingLocalRecognition?: boolean;
}

export interface LiveCaptionController {
  start(): Promise<void>;
  stop(): void;
  isSupported(): boolean;
}

interface SpeechRecognitionAlternativeLike {
  transcript: string;
  confidence: number;
}

interface SpeechRecognitionResultLike {
  readonly isFinal: boolean;
  readonly length: number;
  [index: number]: SpeechRecognitionAlternativeLike;
}

interface SpeechRecognitionResultListLike {
  readonly length: number;
  [index: number]: SpeechRecognitionResultLike;
}

interface SpeechRecognitionEventLike extends Event {
  readonly resultIndex: number;
  readonly results: SpeechRecognitionResultListLike;
}

interface SpeechRecognitionErrorEventLike extends Event {
  readonly error: string;
  readonly message?: string;
}

interface SpeechRecognitionLike extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  maxAlternatives: number;
  lang: string;
  processLocally?: boolean;
  start(audioTrack?: MediaStreamTrack): void;
  stop(): void;
  abort(): void;
  onstart: ((event: Event) => void) | null;
  onend: ((event: Event) => void) | null;
  onresult: ((event: SpeechRecognitionEventLike) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEventLike) => void) | null;
}

type SpeechRecognitionConstructor = new () => SpeechRecognitionLike;

type CapturableVideo = HTMLVideoElement & {
  captureStream?: () => MediaStream;
  mozCaptureStream?: () => MediaStream;
};

type SpeechWindow = Window & {
  SpeechRecognition?: SpeechRecognitionConstructor;
  webkitSpeechRecognition?: SpeechRecognitionConstructor;
};

const LANGUAGE_MAP: Record<string, string> = {
  et: "et-EE",
  est: "et-EE",
  en: "en-US",
  eng: "en-US",
  fi: "fi-FI",
  fin: "fi-FI",
  sv: "sv-SE",
  swe: "sv-SE",
  de: "de-DE",
  deu: "de-DE",
  ger: "de-DE",
  fr: "fr-FR",
  fra: "fr-FR",
  fre: "fr-FR",
  es: "es-ES",
  spa: "es-ES",
  it: "it-IT",
  ita: "it-IT",
  ru: "ru-RU",
  rus: "ru-RU",
  uk: "uk-UA",
  ukr: "uk-UA",
  pl: "pl-PL",
  pol: "pl-PL",
  lv: "lv-LV",
  lav: "lv-LV",
  lt: "lt-LT",
  lit: "lt-LT",
  da: "da-DK",
  dan: "da-DK",
  no: "nb-NO",
  nor: "nb-NO",
  nl: "nl-NL",
  nld: "nl-NL",
  dut: "nl-NL",
  pt: "pt-PT",
  por: "pt-PT",
  ja: "ja-JP",
  jpn: "ja-JP",
  ko: "ko-KR",
  kor: "ko-KR",
  zh: "zh-CN",
  zho: "zh-CN",
  chi: "zh-CN",
};

function normalizeLanguage(language?: string): string {
  const value = language?.trim();
  if (!value) return navigator.language || "en-US";
  if (/^[a-z]{2,3}-[A-Z]{2}$/.test(value)) return value;
  const key = value.toLowerCase().split(/[-_]/)[0] ?? value.toLowerCase();
  return LANGUAGE_MAP[key] || value;
}

function recognitionConstructor(): SpeechRecognitionConstructor | undefined {
  const speechWindow = window as SpeechWindow;
  if (speechWindow.SpeechRecognition) return speechWindow.SpeechRecognition;

  // Older prefixed implementations may ignore start(audioTrack) and capture
  // the microphone instead. Use them only when the method advertises an input.
  const prefixed = speechWindow.webkitSpeechRecognition;
  if (prefixed && prefixed.prototype?.start?.length > 0) return prefixed;
  return undefined;
}

function captureAudioTrack(video: HTMLVideoElement): MediaStreamTrack | undefined {
  const capturable = video as CapturableVideo;
  const capture = capturable.captureStream ?? capturable.mozCaptureStream;
  if (!capture) return undefined;
  const stream = capture.call(video);
  return stream.getAudioTracks()[0];
}

function userFacingError(code: string, message?: string): string {
  switch (code) {
    case "audio-capture":
      return tr('Playeri helirada ei olnud Live Captioni jaoks saadaval.');
    case "language-not-supported":
      return tr('Chrome ei toeta valitud keele kõnetuvastust.');
    case "not-allowed":
    case "service-not-allowed":
      return tr('Chrome ei lubanud kõnetuvastust käivitada.');
    case "network":
      return tr("Chrome'i kõnetuvastusteenusega ei saadud ühendust.");
    default:
      return message || tr('Live Caption ebaõnnestus ({{error}}).', { error: code || tr('tundmatu viga') });
  }
}

export function createLiveCaptionController(
  video: HTMLVideoElement,
  language: string | undefined,
  onUpdate: (snapshot: LiveCaptionSnapshot) => void,
): LiveCaptionController {
  const Constructor = recognitionConstructor();
  const supported = Boolean(Constructor) && Boolean(
    (video as CapturableVideo).captureStream ||
      (video as CapturableVideo).mozCaptureStream,
  );

  let enabled = false;
  let recognition: SpeechRecognitionLike | null = null;
  let restartTimer: number | null = null;
  let clearTimer: number | null = null;
  let finalLines: string[] = [];
  let onlineFallbackUsed = false;
  let localAttempt = false;

  const emit = (snapshot: LiveCaptionSnapshot) => onUpdate(snapshot);

  const clearTimers = () => {
    if (restartTimer !== null) window.clearTimeout(restartTimer);
    if (clearTimer !== null) window.clearTimeout(clearTimer);
    restartTimer = null;
    clearTimer = null;
  };

  const scheduleTextClear = () => {
    if (clearTimer !== null) window.clearTimeout(clearTimer);
    clearTimer = window.setTimeout(() => {
      if (!enabled) return;
      finalLines = [];
      emit({
        state: "active",
        text: "",
        usingLocalRecognition: localAttempt,
      });
    }, 5_500);
  };

  const startRecognition = (preferLocal: boolean) => {
    if (!enabled || !Constructor) return;

    const audioTrack = captureAudioTrack(video);
    if (!audioTrack || audioTrack.readyState !== "live") {
      emit({
        state: "error",
        text: "",
        error: tr('Playeri helirada ei olnud Live Captioni jaoks saadaval.'),
      });
      enabled = false;
      return;
    }

    const instance = new Constructor();
    recognition = instance;
    instance.continuous = true;
    instance.interimResults = true;
    instance.maxAlternatives = 1;
    instance.lang = normalizeLanguage(language);

    localAttempt = preferLocal && "processLocally" in instance;
    if (localAttempt) instance.processLocally = true;

    instance.onstart = () => {
      if (!enabled) return;
      emit({
        state: "active",
        text: finalLines.join(" "),
        usingLocalRecognition: localAttempt,
      });
    };

    instance.onresult = (event) => {
      if (!enabled) return;
      let interim = "";
      for (let index = event.resultIndex; index < event.results.length; index += 1) {
        const result = event.results[index];
        const transcript = result?.[0]?.transcript?.trim();
        if (!transcript) continue;
        if (result?.isFinal) {
          finalLines.push(transcript);
          finalLines = finalLines.slice(-2);
        } else {
          interim = `${interim} ${transcript}`.trim();
        }
      }
      const text = [...finalLines, interim].filter(Boolean).join(" ").trim();
      emit({
        state: "active",
        text,
        usingLocalRecognition: localAttempt,
      });
      scheduleTextClear();
    };

    instance.onerror = (event) => {
      if (!enabled || event.error === "aborted" || event.error === "no-speech") return;

      if (
        localAttempt &&
        !onlineFallbackUsed &&
        ["language-not-supported", "service-not-allowed", "not-allowed"].includes(
          event.error,
        )
      ) {
        onlineFallbackUsed = true;
        localAttempt = false;
        try {
          instance.abort();
        } catch {
          // The instance may already have ended.
        }
        restartTimer = window.setTimeout(() => startRecognition(false), 150);
        return;
      }

      enabled = false;
      emit({
        state: "error",
        text: "",
        error: userFacingError(event.error, event.message),
        usingLocalRecognition: localAttempt,
      });
    };

    instance.onend = () => {
      if (!enabled) return;
      recognition = null;
      restartTimer = window.setTimeout(
        () => startRecognition(!onlineFallbackUsed),
        220,
      );
    };

    emit({
      state: "starting",
      text: finalLines.join(" "),
      usingLocalRecognition: localAttempt,
    });

    try {
      instance.start(audioTrack);
    } catch (error) {
      if (localAttempt && !onlineFallbackUsed) {
        onlineFallbackUsed = true;
        localAttempt = false;
        restartTimer = window.setTimeout(() => startRecognition(false), 150);
        return;
      }
      enabled = false;
      emit({
        state: "error",
        text: "",
        error:
          error instanceof Error
            ? error.message
            : tr('Live Captioni ei saanud käivitada.'),
      });
    }
  };

  return {
    isSupported: () => supported,
    async start() {
      if (!supported || !Constructor) {
        emit({
          state: "unsupported",
          text: "",
          error:
            tr('See brauser ei toeta video heliraja reaalajas kõnetuvastust.'),
        });
        return;
      }
      if (enabled) return;
      enabled = true;
      onlineFallbackUsed = false;
      finalLines = [];
      clearTimers();
      startRecognition(true);
    },
    stop() {
      enabled = false;
      clearTimers();
      finalLines = [];
      const current = recognition;
      recognition = null;
      if (current) {
        current.onstart = null;
        current.onend = null;
        current.onresult = null;
        current.onerror = null;
        try {
          current.abort();
        } catch {
          // The instance may already have ended.
        }
      }
      emit({ state: "stopped", text: "" });
    },
  };
}

export function liveCaptionLanguageLabel(language?: string): string {
  const normalized = normalizeLanguage(language);
  try {
    return new Intl.DisplayNames([getLocale()], {
      type: "language",
    }).of(normalized) || normalized;
  } catch {
    return normalized;
  }
}
