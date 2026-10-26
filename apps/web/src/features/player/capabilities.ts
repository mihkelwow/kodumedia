import type {
  ClientCapabilities,
  MediaItem,
  VideoDecodeCapability,
} from '../../api/types'

type TechnicalInput = MediaItem['technical']

type DecodingInfoResult = {
  supported?: boolean
  smooth?: boolean
  powerEfficient?: boolean
}

type MediaCapabilitiesLike = {
  decodingInfo?: (configuration: unknown) => Promise<DecodingInfoResult>
}

type DecodeProbe = Omit<VideoDecodeCapability, 'supported' | 'smooth' | 'powerEfficient'>

const capabilityCache = new Map<string, Promise<ClientCapabilities>>()
const maximumCachedSources = 32

function mediaElement(mime: string): HTMLMediaElement {
  return document.createElement(mime.startsWith('audio/') ? 'audio' : 'video')
}

function canPlay(mime: string): CanPlayTypeResult {
  return mediaElement(mime).canPlayType(mime)
}

function mseSupports(mime: string): boolean {
  return typeof MediaSource !== 'undefined' && MediaSource.isTypeSupported(mime)
}

function decoderAdvertised(mime: string): boolean {
  return mseSupports(mime) || canPlay(mime) !== ''
}

const hdrDisplayQueries = [
  '(dynamic-range: high)',
  '(video-dynamic-range: high)',
  '(color-gamut: rec2020)',
] as const

function hasHDRDisplaySignal(): boolean {
  if (typeof window.matchMedia !== 'function') return false
  return hdrDisplayQueries.some((query) => window.matchMedia(query).matches)
}

function displayCapabilitySignature(): string {
  const screenWidth = window.screen?.width ?? 0
  const screenHeight = window.screen?.height ?? 0
  const colorDepth = window.screen?.colorDepth ?? 0
  const pixelDepth = window.screen?.pixelDepth ?? 0
  const devicePixelRatio = window.devicePixelRatio || 1
  const hdrSignals = typeof window.matchMedia === 'function'
    ? hdrDisplayQueries.map((query) => (window.matchMedia(query).matches ? '1' : '0')).join('')
    : '000'

  return [
    screenWidth,
    screenHeight,
    colorDepth,
    pixelDepth,
    devicePixelRatio.toFixed(3),
    hdrSignals,
  ].join('x')
}

export function clearClientCapabilityCache(): void {
  capabilityCache.clear()
}

export function normalizeVideoCodec(codec?: string): string {
  const value = (codec ?? '').trim().toLowerCase()
  switch (value) {
    case 'h265':
      return 'hevc'
    case 'avc':
    case 'avc1':
      return 'h264'
    case 'vp09':
      return 'vp9'
    default:
      return value
  }
}

function h264CodecString(profile?: string, level?: number): string {
  const normalized = (profile ?? '').toLowerCase()
  let profileHex = '64'
  if (normalized.includes('baseline')) profileHex = '42'
  else if (normalized.includes('main')) profileHex = '4D'

  const normalizedLevel = level && level > 0 ? Math.min(255, level) : 40
  return `avc1.${profileHex}00${normalizedLevel.toString(16).padStart(2, '0').toUpperCase()}`
}

export function codecContentType(
  codec: string,
  bitDepth = 8,
  profile?: string,
  level?: number,
): string | undefined {
  switch (normalizeVideoCodec(codec)) {
    case 'h264':
      return `video/mp4; codecs="${h264CodecString(profile, level)}"`
    case 'hevc':
      return bitDepth > 8
        ? 'video/mp4; codecs="hvc1.2.4.L153.B0"'
        : 'video/mp4; codecs="hvc1.1.6.L153.B0"'
    case 'av1':
      return bitDepth > 8
        ? 'video/mp4; codecs="av01.0.12M.10"'
        : 'video/mp4; codecs="av01.0.08M.08"'
    case 'vp9':
      return bitDepth > 8
        ? 'video/webm; codecs="vp09.02.51.10"'
        : 'video/webm; codecs="vp09.00.51.08"'
    default:
      return undefined
  }
}

function baselineCapabilities(): ClientCapabilities {
  const supportedMimes: string[] = []
  const videoCodecs: string[] = []
  const audioCodecs: string[] = []

  const h264 = 'video/mp4; codecs="avc1.42E01E, mp4a.40.2"'
  const hevcCandidates = [
    'video/mp4; codecs="hvc1.1.6.L120.B0"',
    'video/mp4; codecs="hev1.1.6.L120.B0"',
    'video/mp4; codecs="hvc1.2.4.L153.B0"',
  ]
  const av1 = 'video/mp4; codecs="av01.0.05M.08, mp4a.40.2"'
  const vp9 = 'video/webm; codecs="vp09.00.10.08, opus"'

  if (decoderAdvertised(h264)) {
    supportedMimes.push(h264, 'video/mp4', 'audio/mp4')
    videoCodecs.push('h264')
    audioCodecs.push('aac')
  }

  const supportedHEVC = hevcCandidates.filter(decoderAdvertised)
  if (supportedHEVC.length > 0) {
    supportedMimes.push(...supportedHEVC, 'video/mp4')
    videoCodecs.push('hevc')
  }

  if (decoderAdvertised(av1)) {
    supportedMimes.push(av1, 'video/mp4')
    videoCodecs.push('av1')
  }

  if (decoderAdvertised(vp9)) {
    supportedMimes.push(vp9, 'video/webm', 'audio/webm')
    videoCodecs.push('vp9')
    audioCodecs.push('opus')
  }

  if (canPlay('audio/mpeg') !== '') {
    supportedMimes.push('audio/mpeg')
    audioCodecs.push('mp3')
  }
  if (canPlay('audio/flac') !== '') {
    supportedMimes.push('audio/flac')
    audioCodecs.push('flac')
  }
  if (canPlay('audio/ogg; codecs="opus"') !== '') {
    supportedMimes.push('audio/ogg')
    audioCodecs.push('opus')
  }
  if (canPlay('audio/wav') !== '') supportedMimes.push('audio/wav')

  const screenWidth = Math.max(
    1280,
    Math.round((window.screen?.width ?? 1280) * window.devicePixelRatio),
  )
  const screenHeight = Math.max(
    720,
    Math.round((window.screen?.height ?? 720) * window.devicePixelRatio),
  )
  const hdr = hasHDRDisplaySignal()

  return {
    clientName: 'web',
    clientVersion: navigator.userAgent,
    supportedMimes: [...new Set(supportedMimes)],
    videoCodecs: [...new Set(videoCodecs)],
    audioCodecs: [...new Set(audioCodecs)],
    subtitleFormats: ['webvtt'],
    hdrFormats: hdr ? ['hdr10', 'hdr10plus', 'hlg'] : [],
    supportsMse: typeof MediaSource !== 'undefined',
    supportsHls:
      canPlay('application/vnd.apple.mpegurl') !== '' ||
      typeof MediaSource !== 'undefined',
    supportsPip:
      'pictureInPictureEnabled' in document &&
      document.pictureInPictureEnabled,
    maximumWidth: screenWidth,
    maximumHeight: screenHeight,
    maximumAudioChannels: 2,
    decodeCapabilities: [],
  }
}

async function probeDecodeCapability(probe: DecodeProbe): Promise<VideoDecodeCapability> {
  const fallbackSupported =
    decoderAdvertised(probe.contentType ?? '') &&
    (!probe.hdrFormat || hasHDRDisplaySignal())
  const mediaCapabilities = (
    navigator as Navigator & { mediaCapabilities?: MediaCapabilitiesLike }
  ).mediaCapabilities

  if (!mediaCapabilities?.decodingInfo) {
    return {
      ...probe,
      supported: fallbackSupported,
      smooth: fallbackSupported,
      powerEfficient: false,
    }
  }

  try {
    const result = await mediaCapabilities.decodingInfo({
      type: 'media-source',
      video: {
        contentType: probe.contentType,
        width: probe.width,
        height: probe.height,
        bitrate: probe.bitrate,
        framerate: probe.frameRate,
        ...(probe.hdrFormat
          ? {
              colorGamut: 'rec2020',
              transferFunction: probe.hdrFormat === 'hlg' ? 'hlg' : 'pq',
              hdrMetadataType: 'smpteSt2086',
            }
          : {}),
      },
    })
    return {
      ...probe,
      supported: Boolean(result.supported),
      smooth: Boolean(result.smooth),
      powerEfficient: Boolean(result.powerEfficient),
    }
  } catch {
    return {
      ...probe,
      supported: fallbackSupported,
      smooth: fallbackSupported,
      powerEfficient: false,
    }
  }
}

function uniqueProbes(values: DecodeProbe[]): DecodeProbe[] {
  const seen = new Set<string>()
  return values.filter((value) => {
    const key = [
      value.codec,
      value.codecProfile,
      value.codecLevel,
      value.source,
      value.contentType,
      value.width,
      value.height,
      value.frameRate,
      value.bitrate,
      value.bitDepth,
      value.hdrFormat,
    ].join('|')
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

function scaledSize(width: number, height: number, maximumWidth: number, maximumHeight: number) {
  if (width <= maximumWidth && height <= maximumHeight) return { width, height }
  const ratio = Math.min(maximumWidth / width, maximumHeight / height)
  return {
    width: Math.max(2, Math.floor((width * ratio) / 2) * 2),
    height: Math.max(2, Math.floor((height * ratio) / 2) * 2),
  }
}

function capabilityKey(technical?: TechnicalInput): string {
  return [
    displayCapabilitySignature(),
    normalizeVideoCodec(technical?.videoCodec),
    technical?.videoProfile ?? '',
    technical?.videoLevel ?? 0,
    technical?.width ?? 0,
    technical?.height ?? 0,
    technical?.frameRate ?? 0,
    technical?.bitrate ?? 0,
    technical?.bitDepth ?? 0,
    technical?.hdrFormat ?? '',
  ].join('|')
}

async function detectUncached(technical?: TechnicalInput): Promise<ClientCapabilities> {
  const capabilities = baselineCapabilities()
  const sourceWidth = Math.max(320, technical?.width ?? 1920)
  const sourceHeight = Math.max(180, technical?.height ?? 1080)
  const sourceFrameRate = Math.max(1, technical?.frameRate ?? 24)
  const sourceBitrate = Math.max(
    1_000_000,
    technical?.bitrate ?? (sourceWidth >= 3000 ? 25_000_000 : 8_000_000),
  )
  const sourceBitDepth = Math.max(8, technical?.bitDepth ?? 8)
  const sourceCodec = normalizeVideoCodec(technical?.videoCodec)
  const probes: DecodeProbe[] = []

  const sourceContentType = sourceCodec
    ? codecContentType(
        sourceCodec,
        sourceBitDepth,
        technical?.videoProfile,
        technical?.videoLevel,
      )
    : undefined
  if (sourceCodec && sourceContentType) {
    probes.push({
      codec: sourceCodec,
      codecProfile: technical?.videoProfile,
      codecLevel: technical?.videoLevel,
      source: true,
      contentType: sourceContentType,
      width: sourceWidth,
      height: sourceHeight,
      frameRate: sourceFrameRate,
      bitrate: sourceBitrate,
      bitDepth: sourceBitDepth,
      hdrFormat: technical?.hdrFormat,
    })
  }

  const outputSizes = [
    { width: sourceWidth, height: sourceHeight, bitrate: sourceBitrate },
    { ...scaledSize(sourceWidth, sourceHeight, 1920, 1080), bitrate: 10_000_000 },
    { ...scaledSize(sourceWidth, sourceHeight, 1280, 720), bitrate: 5_000_000 },
    { ...scaledSize(sourceWidth, sourceHeight, 854, 480), bitrate: 2_500_000 },
  ]

  for (const size of outputSizes) {
    for (const codec of ['h264', 'hevc', 'av1'] as const) {
      const bitDepth = codec === 'h264' ? 8 : sourceBitDepth
      const contentType = codecContentType(codec, bitDepth)
      if (!contentType) continue
      probes.push({
        codec,
        source: false,
        contentType,
        width: size.width,
        height: size.height,
        frameRate: sourceFrameRate,
        bitrate: size.bitrate,
        bitDepth,
        hdrFormat: codec === 'h264' ? undefined : technical?.hdrFormat,
      })
    }
  }

  const decodeCapabilities = await Promise.all(
    uniqueProbes(probes).map(probeDecodeCapability),
  )
  const supportedOutput = decodeCapabilities.filter(
    (value) => value.supported && value.smooth && !value.source,
  )

  capabilities.videoCodecs = [
    ...new Set([
      ...capabilities.videoCodecs,
      ...decodeCapabilities
        .filter((value) => value.supported && value.smooth)
        .map((value) => value.codec),
    ]),
  ]
  if (supportedOutput.length > 0) {
    capabilities.maximumWidth = Math.max(
      capabilities.maximumWidth ?? 0,
      ...supportedOutput.map((value) => value.width ?? 0),
    )
    capabilities.maximumHeight = Math.max(
      capabilities.maximumHeight ?? 0,
      ...supportedOutput.map((value) => value.height ?? 0),
    )
  }
  capabilities.decodeCapabilities = decodeCapabilities
  return capabilities
}

export async function detectClientCapabilities(
  technical?: TechnicalInput,
): Promise<ClientCapabilities> {
  const key = capabilityKey(technical)
  const existing = capabilityCache.get(key)
  if (existing) return existing

  const pending = detectUncached(technical).catch((error) => {
    capabilityCache.delete(key)
    throw error
  })
  capabilityCache.set(key, pending)
  if (capabilityCache.size > maximumCachedSources) {
    const oldest = capabilityCache.keys().next().value as string | undefined
    if (oldest && oldest !== key) capabilityCache.delete(oldest)
  }
  return pending
}
