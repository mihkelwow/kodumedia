interface ProgressBarProps {
  value: number
  className?: string
  label?: string
}

/** Shared progress primitive. Visual progress is transform-based to avoid layout work. */
export function ProgressBar({ value, className = '', label }: ProgressBarProps) {
  const normalized = Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0))
  return (
    <span
      className={`scan-status__bar ${className}`.trim()}
      role={label ? 'progressbar' : undefined}
      aria-label={label}
      aria-valuemin={label ? 0 : undefined}
      aria-valuemax={label ? 100 : undefined}
      aria-valuenow={label ? Math.round(normalized) : undefined}
    >
      <span style={{ transform: `scaleX(${normalized / 100})` }} />
    </span>
  )
}
