import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { BottomNav } from './BottomNav'

function renderAt(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <BottomNav />
    </MemoryRouter>,
  )
}

describe('BottomNav', () => {
  it('5つのタブを、それぞれの画面へのリンクとして並べる', () => {
    renderAt('/')
    const nav = screen.getByRole('navigation', { name: 'メインメニュー' })
    expect(nav).toBeInTheDocument()
    expect(screen.getAllByRole('link').map((a) => [a.textContent, a.getAttribute('href')])).toEqual(
      [
        ['ホーム', '/'],
        ['検索', '/search'],
        ['出品', '/sell'],
        ['取引', '/deals'],
        ['マイページ', '/mypage'],
      ],
    )
  })

  it('いまの画面のタブだけに aria-current="page" を付ける', () => {
    renderAt('/deals')
    expect(screen.getByRole('link', { name: '取引' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'ホーム' })).not.toHaveAttribute('aria-current')
  })

  it('取引詳細など、タブの下の画面でもそのタブを選択中にする', () => {
    renderAt('/mypage/sales')
    expect(screen.getByRole('link', { name: 'マイページ' })).toHaveAttribute('aria-current', 'page')
  })
})
