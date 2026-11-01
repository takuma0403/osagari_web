import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { BottomNav } from './BottomNav'

function renderAt(path: string, placement?: 'bottom' | 'top') {
  render(
    <MemoryRouter initialEntries={[path]}>
      <BottomNav placement={placement} />
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

  it('下部ではロゴとアプリ名を出さない', () => {
    renderAt('/')
    expect(screen.queryByText('佐賀大お下がりフリマ')).not.toBeInTheDocument()
  })

  it('上部（PC）では左にアプリ名を出し、同じ5つのタブを並べる', () => {
    renderAt('/search', 'top')
    expect(screen.getByRole('navigation', { name: 'メインメニュー' })).toHaveTextContent(
      '佐賀大お下がりフリマ',
    )
    expect(screen.getAllByRole('link')).toHaveLength(5)
    expect(screen.getByRole('link', { name: '検索' })).toHaveAttribute('aria-current', 'page')
  })
})
