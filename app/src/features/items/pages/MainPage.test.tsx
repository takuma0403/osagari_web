import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json, page, stubItemsFetch, summary } from '../test/fixtures'
import { MainPage } from './MainPage'

function setup(at = '/') {
  const view = renderRoute('/', <MainPage />, { at, destinations: ['/search'] })
  return { ...view, user: userEvent.setup() }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('MainPage', () => {
  it('公開中の商品を新着順にカードで並べ、商品ページへリンクする', async () => {
    stubItemsFetch({
      '/api/items': () =>
        json(
          page([
            summary(),
            summary({
              shareId: 'def456',
              title: 'ローテーブル 折りたたみ',
              category: 'furniture_appliance',
              price: 800,
              campus: 'nabeshima',
              condition: 'new_like',
            }),
          ]),
        ),
    })
    setup()
    expect(screen.getByRole('heading', { level: 1, name: '新着の出品' })).toBeInTheDocument()
    const table = await screen.findByRole('link', { name: /ローテーブル 折りたたみ/ })
    expect(table).toHaveAttribute('href', '/items/def456')
    expect(table).toHaveTextContent('家具・家電')
    expect(table).toHaveTextContent('¥800')
    expect(table).toHaveTextContent('鍋島キャンパス · 新品同様')
    expect(screen.getByRole('link', { name: /線形代数入門/ })).toHaveAttribute(
      'href',
      '/items/abc123',
    )
  })

  it('カテゴリを選ぶと、そのカテゴリだけを読み込み直し、URL に残す', async () => {
    const fetchMock = stubItemsFetch({
      '/api/items': () => json(page([summary()])),
      '/api/items?category=furniture_appliance': () =>
        json(
          page([
            summary({ shareId: 'def456', title: '冷蔵庫 2ドア', category: 'furniture_appliance' }),
          ]),
        ),
    })
    const { user, router } = setup()
    await screen.findByRole('link', { name: /線形代数入門/ })
    const categories = screen.getByRole('group', { name: 'カテゴリ' })
    expect(within(categories).getByRole('button', { name: 'すべて' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )

    await user.click(within(categories).getByRole('button', { name: '家具・家電' }))
    expect(await screen.findByRole('link', { name: /冷蔵庫 2ドア/ })).toBeInTheDocument()
    expect(within(categories).getByRole('button', { name: '家具・家電' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(router.state.location.search).toBe('?category=furniture_appliance')
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/items?category=furniture_appliance',
      expect.anything(),
    )
  })

  it('URL のカテゴリで開くと、そのカテゴリを選んだ状態にする', async () => {
    stubItemsFetch({ '/api/items?category=game_other': () => json(page([])) })
    setup('/?category=game_other')
    expect(await screen.findByText('まだ出品がありません')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'ゲーム・その他' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  it('検索欄で確定すると、選んでいるカテゴリを引き継いで検索結果へ移る', async () => {
    stubItemsFetch({ '/api/items?category=textbook': () => json(page([])) })
    const { user, router } = setup('/?category=textbook')
    await user.type(screen.getByRole('searchbox', { name: 'キーワード' }), ' 線形代数 {Enter}')
    expect(router.state.location.pathname).toBe('/search')
    expect(new URLSearchParams(router.state.location.search).get('keyword')).toBe('線形代数')
    expect(new URLSearchParams(router.state.location.search).get('category')).toBe('textbook')
  })

  it('続きがあれば「もっと見る」で足す', async () => {
    stubItemsFetch({
      '/api/items': () => json(page([summary()], 'c1', 2)),
      '/api/items?cursor=c1': () =>
        json(page([summary({ shareId: 'def456', title: '解剖学講義' })], null, 2)),
    })
    const { user } = setup()
    await user.click(await screen.findByRole('button', { name: 'もっと見る' }))
    expect(await screen.findByRole('link', { name: /解剖学講義/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /線形代数入門/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'もっと見る' })).not.toBeInTheDocument()
  })

  it('読み込めなかったらエラーを出し、もう一度読み込める', async () => {
    let fail = true
    stubItemsFetch({
      '/api/items': () =>
        fail
          ? json({ code: 'internal', message: 'しばらくしてからもう一度お試しください' }, 500)
          : json(page([summary()])),
    })
    const { user } = setup()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'しばらくしてからもう一度お試しください',
    )
    fail = false
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByRole('link', { name: /線形代数入門/ })).toBeInTheDocument()
  })
})
