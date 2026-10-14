import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { ItemRow } from './ItemRow'

describe('ItemRow', () => {
  it('商品詳細へのリンクに、タイトル・補足・価格・状態を表示する', () => {
    render(
      <MemoryRouter>
        <ItemRow
          to="/items/abc123"
          title="線形代数入門"
          subtitle="線形代数学I · 理工学部"
          price={1500}
          note="書き込みなし"
          tint="textbook"
        />
      </MemoryRouter>,
    )
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/items/abc123')
    expect(link).toHaveTextContent('線形代数入門')
    expect(link).toHaveTextContent('線形代数学I · 理工学部')
    expect(link).toHaveTextContent('¥1,500')
    expect(screen.getByText('書き込みなし')).toBeInTheDocument()
  })
})
