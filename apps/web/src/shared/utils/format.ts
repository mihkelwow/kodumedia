import { getLocale, tr } from '../../i18n/i18n'

export function formatDuration(milliseconds?: number): string {
  if (!milliseconds) return ''
  const minutes = Math.round(milliseconds / 60000)
  const hours = Math.floor(minutes / 60)
  const remainder = minutes % 60
  return hours > 0 ? tr('{{hours}} h {{minutes}} min', { hours, minutes: remainder }) : tr('{{minutes}} min', { minutes })
}

export function formatBytes(bytes?: number): string {
  if (!bytes) return ''
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(value >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`
}

export function formatDate(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(getLocale(), { day: 'numeric', month: 'long', year: 'numeric' }).format(date)
}

export function formatVotes(votes?: number): string {
  if (!votes) return ''
  return new Intl.NumberFormat(getLocale(), { notation: votes > 9999 ? 'compact' : 'standard', maximumFractionDigits: 1 }).format(votes)
}
