import { tr } from '../../i18n/i18n'
export interface SubtitleCue {
  id?: string
  startMs: number
  endMs: number
  text: string
}

const timestampPattern = /^(?:(\d+):)?(\d{1,2}):(\d{2})[.,](\d{1,3})$/

export function parseWebVttTimestamp(value: string): number | undefined {
  const match = timestampPattern.exec(value.trim())
  if (!match) return undefined
  const hours = Number(match[1] ?? 0)
  const minutes = Number(match[2])
  const seconds = Number(match[3])
  const milliseconds = Number((match[4] ?? '').padEnd(3, '0'))
  if (![hours, minutes, seconds, milliseconds].every(Number.isFinite)) return undefined
  if (minutes > 59 || seconds > 59) return undefined
  return (((hours * 60 + minutes) * 60) + seconds) * 1000 + milliseconds
}

function decodeEntity(entity: string): string {
  const lower = entity.toLowerCase()
  switch (lower) {
    case '&amp;': return '&'
    case '&lt;': return '<'
    case '&gt;': return '>'
    case '&nbsp;': return '\u00a0'
    case '&lrm;': return '\u200e'
    case '&rlm;': return '\u200f'
    case '&quot;': return '"'
    case '&apos;': return "'"
    default: {
      const decimal = /^&#(\d+);$/.exec(lower)
      if (decimal) {
        const codePoint = Number(decimal[1])
        if (Number.isSafeInteger(codePoint) && codePoint >= 0 && codePoint <= 0x10ffff) {
          return String.fromCodePoint(codePoint)
        }
      }
      const hexadecimal = /^&#x([0-9a-f]+);$/.exec(lower)
      if (hexadecimal) {
        const codePoint = Number.parseInt(hexadecimal[1] ?? '', 16)
        if (Number.isSafeInteger(codePoint) && codePoint >= 0 && codePoint <= 0x10ffff) {
          return String.fromCodePoint(codePoint)
        }
      }
      return entity
    }
  }
}

export function normalizeCueText(value: string): string {
  return value
    .replace(/<br\s*\/?\s*>/gi, '\n')
    .replace(/\{\\[^}]*}/g, '')
    .replace(/<\/?(?:b|i|u|ruby|rt)>/gi, '')
    .replace(/<\/?c(?:\.[^ >]+)*>/gi, '')
    .replace(/<\/?v(?:\s+[^>]*)?>/gi, '')
    .replace(/<\/?lang(?:\s+[^>]*)?>/gi, '')
    .replace(/<[^>]*>/g, '')
    .replace(/&(?:amp|lt|gt|nbsp|lrm|rlm|quot|apos|#\d+|#x[0-9a-f]+);/gi, decodeEntity)
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n[ \t]+/g, '\n')
    .trim()
}

export function parseWebVtt(input: string): SubtitleCue[] {
  const normalized = input.replace(/^\ufeff/, '').replace(/\r\n?/g, '\n')
  const lines = normalized.split('\n')
  if (!lines[0]?.trimStart().startsWith('WEBVTT')) {
    throw new Error(tr('Server ei tagastanud korrektset WebVTT subtiitrifaili.'))
  }

  const cues: SubtitleCue[] = []
  let index = 1

  // Skip optional WEBVTT header metadata until the first empty line.
  while (index < lines.length && lines[index]?.trim() !== '') index += 1

  while (index < lines.length) {
    while (index < lines.length && lines[index]?.trim() === '') index += 1
    if (index >= lines.length) break

    const block: string[] = []
    while (index < lines.length && lines[index]?.trim() !== '') {
      block.push(lines[index] ?? '')
      index += 1
    }
    if (block.length === 0) continue

    const first = block[0]?.trim().toUpperCase() ?? ''
    if (first === 'STYLE' || first === 'REGION' || first.startsWith('NOTE')) continue

    const timingIndex = block.findIndex((line) => line.includes('-->'))
    if (timingIndex < 0) continue
    const timing = block[timingIndex]?.split('-->') ?? []
    const startToken = timing[0]?.trim().split(/\s+/)[0] ?? ''
    const endToken = timing[1]?.trim().split(/\s+/)[0] ?? ''
    const startMs = parseWebVttTimestamp(startToken)
    const endMs = parseWebVttTimestamp(endToken)
    if (startMs === undefined || endMs === undefined || endMs <= startMs) continue

    const text = normalizeCueText(block.slice(timingIndex + 1).join('\n'))
    if (!text) continue
    cues.push({
      ...(timingIndex > 0 && block[0]?.trim() ? { id: block[0].trim() } : {}),
      startMs,
      endMs,
      text,
    })
  }

  return cues.sort((left, right) => left.startMs - right.startMs || left.endMs - right.endMs)
}

export function activeSubtitleCues(cues: readonly SubtitleCue[], positionMs: number): SubtitleCue[] {
  if (cues.length === 0 || !Number.isFinite(positionMs)) return []
  const position = Math.max(0, positionMs)
  return cues.filter((cue) => cue.startMs <= position && cue.endMs > position)
}
