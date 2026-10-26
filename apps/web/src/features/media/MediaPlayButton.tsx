import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Button } from '../../shared/components/Button'

interface MediaPlayButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  children: ReactNode
}

export function MediaPlayButton({ children, className = '', ...props }: MediaPlayButtonProps) {
  return (
    <Button
      {...props}
      className={`media-play-button ${className}`.trim()}
      variant="primary"
      icon="play"
    >
      {children}
    </Button>
  )
}
