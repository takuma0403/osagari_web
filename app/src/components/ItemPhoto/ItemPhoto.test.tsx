import { render } from '@testing-library/react'
import { ItemPhoto } from './ItemPhoto'

describe('ItemPhoto', () => {
  it('写真があれば、装飾画像として遅延読み込みで表示する', () => {
    const { container } = render(<ItemPhoto imageUrl="https://example.com/a.jpg" tint="textbook" />)
    const img = container.querySelector('img')
    expect(img).toHaveAttribute('src', 'https://example.com/a.jpg')
    expect(img).toHaveAttribute('alt', '')
    expect(img).toHaveAttribute('loading', 'lazy')
  })

  it.each(['textbook', 'appliance', 'furniture', 'game'] as const)(
    '写真がなければ %s の色の地にする',
    (tint) => {
      const { container } = render(<ItemPhoto tint={tint} />)
      expect(container.querySelector('img')).not.toBeInTheDocument()
      expect((container.firstElementChild as HTMLElement).className).toMatch(tint)
    },
  )
})
