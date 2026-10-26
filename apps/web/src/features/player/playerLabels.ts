import type { MediaItem, PlaybackPlan } from '../../api/types'
import { tr } from '../../i18n/i18n'

export function profileLabel(value?: string): string {
  switch (value) {
    case 'maximum': return tr('Maksimaalne')
    case 'balanced': return tr('Tasakaalus')
    case 'data-saver': return tr('Andmesäästlik')
    default: return tr('Originaal')
  }
}

export function formatBitrate(value?: number): string {
  if (!value || value <= 0) return '—'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(value >= 10_000_000 ? 1 : 2)} Mb/s`
  return `${Math.round(value / 1000)} kb/s`
}

export function resolutionLabel(plan: PlaybackPlan, item: MediaItem): string {
  const width = plan.targetWidth || item.technical.width
  const height = plan.targetHeight || item.technical.height
  return width && height ? `${width}×${height}` : tr('Algne resolutsioon')
}

export function encoderLabel(plan: PlaybackPlan): string {
  if (plan.videoAction !== 'transcode') return tr('Video muutmata')
  const encoder = (plan.videoEncoder || 'software').toLowerCase()
  const names: Record<string, string> = {
    nvenc: 'NVIDIA NVENC',
    qsv: 'Intel QSV',
    amf: 'AMD AMF',
    software: tr('Tarkvaraline'),
  }
  const parts = [names[encoder] || plan.videoEncoder || tr('Tarkvaraline')]
  if (plan.videoPreset) parts.push(plan.videoPreset)
  if (plan.videoCrf) parts.push(`${encoder === 'software' ? 'CRF' : tr('kvaliteet')} ${plan.videoCrf}`)
  return parts.join(' · ')
}

export function decoderLabel(plan: PlaybackPlan): string {
  switch ((plan.videoDecoder || '').toLowerCase()) {
    case 'cuda': return 'NVIDIA NVDEC / CUDA'
    case 'qsv': return 'Intel QSV'
    case 'd3d11va': return 'D3D11VA'
    default: return tr('Tarkvaraline')
  }
}
