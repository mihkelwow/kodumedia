import type { AnchorHTMLAttributes, MouseEvent } from 'react'

interface AppLinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'href'> {
  href: string
  onNavigate: (path: string) => void
}

function shouldUseBrowserNavigation(event: MouseEvent<HTMLAnchorElement>): boolean {
  return (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey ||
    event.currentTarget.target === '_blank'
  )
}

export function AppLink({ href, onNavigate, onClick, ...props }: AppLinkProps) {
  const handleClick = (event: MouseEvent<HTMLAnchorElement>) => {
    onClick?.(event)
    if (shouldUseBrowserNavigation(event)) return

    event.preventDefault()
    onNavigate(href)
  }

  return <a {...props} href={href} onClick={handleClick} />
}
