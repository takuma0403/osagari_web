import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { ItemCard } from './ItemCard'

const props = {
  to: '/items/abc123',
  title: '微分積分学 第3版（書き込み少）',
  price: 1200,
  categoryLabel: '教科書',
  tint: 'textbook',
  campus: '本庄',
  condition: '目立った傷なし',
} as const

describe('ItemCard', () => {
  it('商品詳細へのリンクに、タイトル・価格・カテゴリ・キャンパスと状態を表示する', () => {
    render(
      <MemoryRouter>
        <ItemCard {...props} />
      </MemoryRouter>,
    )
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/items/abc123')
    expect(link).toHaveTextContent('微分積分学 第3版（書き込み少）')
    expect(link).toHaveTextContent('¥1,200')
    expect(link).toHaveTextContent('教科書')
    expect(link).toHaveTextContent('本庄キャンパス · 目立った傷なし')
  })

  it('写真があれば表示し、なければカテゴリ色の地にする', () => {
    const { container, rerender } = render(
      <MemoryRouter>
        <ItemCard {...props} imageUrl="https://example.com/a.jpg" />
      </MemoryRouter>,
    )
    expect(container.querySelector('img')).toHaveAttribute('src', 'https://example.com/a.jpg')

    rerender(
      <MemoryRouter>
        <ItemCard {...props} tint="game" />
      </MemoryRouter>,
    )
    expect(container.querySelector('img')).not.toBeInTheDocument()
    expect(container.querySelector('[class*="game"]')).toBeInTheDocument()
  })
})
