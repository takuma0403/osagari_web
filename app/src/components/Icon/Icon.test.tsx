import { render } from '@testing-library/react'
import { Icon } from './Icon'
import { iconNames } from './iconPaths'

describe('Icon', () => {
  it('24×24 の線画を、読み上げない装飾として描く', () => {
    const { container } = render(<Icon name="search" size={18} />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('viewBox', '0 0 24 24')
    expect(svg).toHaveAttribute('width', '18')
    expect(svg).toHaveAttribute('aria-hidden', 'true')
    expect(svg).toHaveAttribute('stroke', 'currentColor')
  })

  it.each(iconNames)('%s を描ける', (name) => {
    const { container } = render(<Icon name={name} />)
    expect(container.querySelector('svg')?.childElementCount).toBeGreaterThan(0)
  })
})
