import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json, page, stubItemsFetch, summary } from '../test/fixtures'
import { SearchPage } from './SearchPage'

function setup(at = '/search') {
  const view = renderRoute('/search', <SearchPage />, { at, destinations: ['/'] })
  return { ...view, user: userEvent.setup() }
}

const query = (search: string) => Object.fromEntries(new URLSearchParams(search))

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('SearchPage', () => {
  it('URL の条件で検索し、件数と結果を行で並べる', async () => {
    stubItemsFetch({
      '/api/items?keyword=%E7%B7%9A%E5%BD%A2%E4%BB%A3%E6%95%B0&category=textbook': () =>
        json(page([summary({ hasWriting: true })], 'c1', 12)),
    })
    setup('/search?keyword=線形代数&category=textbook')
    expect(screen.getByRole('heading', { level: 1, name: '検索結果' })).toBeInTheDocument()
    expect(screen.getByRole('searchbox', { name: 'キーワード' })).toHaveValue('線形代数')
    const row = await screen.findByRole('link', { name: /線形代数入門/ })
    expect(row).toHaveAttribute('href', '/items/abc123')
    expect(row).toHaveTextContent('線形代数学I')
    expect(row).toHaveTextContent('書き込みあり')
    expect(screen.getByText('12件 · 新着順')).toBeInTheDocument()
  })

  it('条件なしで開くと公開中の商品をすべて出し、0件ならそう伝える', async () => {
    stubItemsFetch({ '/api/items': () => json(page([])) })
    setup()
    expect(await screen.findByText('条件に合う商品が見つかりませんでした')).toBeInTheDocument()
    expect(screen.getByText('0件 · 新着順')).toBeInTheDocument()
  })

  it('キーワードを変えて確定すると、ほかの条件を残したまま検索し直す', async () => {
    stubItemsFetch({
      '/api/items?campus=honjo': () => json(page([])),
      '/api/items?keyword=%E8%A7%A3%E5%89%96%E5%AD%A6&campus=honjo': () => json(page([])),
    })
    const { user, router } = setup('/search?campus=honjo')
    await screen.findByText('0件 · 新着順')
    await user.type(screen.getByRole('searchbox', { name: 'キーワード' }), '解剖学{Enter}')
    expect(query(router.state.location.search)).toEqual({ keyword: '解剖学', campus: 'honjo' })
  })

  it('絞り込み条件をチップで出し、×で外せる', async () => {
    stubItemsFetch({
      '/api/items?category=textbook&priceMax=2000&campus=honjo': () => json(page([])),
      '/api/items?category=textbook&campus=honjo': () => json(page([])),
    })
    const { user, router } = setup('/search?category=textbook&campus=honjo&priceMax=2000')
    await screen.findByText('0件 · 新着順')
    expect(screen.getByRole('button', { name: '教科書・参考書を外す' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '本庄を外す' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '〜¥2,000を外す' }))
    expect(query(router.state.location.search)).toEqual({ category: 'textbook', campus: 'honjo' })
  })

  it('絞り込みのシートで選んだ条件で検索し直す（キーワードは残す）', async () => {
    stubItemsFetch({
      '/api/items?keyword=%E7%B7%9A%E5%BD%A2&category=textbook': () => json(page([])),
      '/api/items?keyword=%E7%B7%9A%E5%BD%A2&category=textbook&priceMin=500&campus=nabeshima': () =>
        json(page([])),
    })
    const { user, router } = setup('/search?keyword=線形&category=textbook')
    await screen.findByText('0件 · 新着順')

    await user.click(screen.getByRole('button', { name: '絞り込み' }))
    const sheet = screen.getByRole('dialog', { name: '絞り込み' })
    // いまの条件を選んだ状態で開く
    expect(within(sheet).getByRole('button', { name: '教科書・参考書' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await user.type(within(sheet).getByLabelText('価格の下限'), '500')
    await user.click(within(sheet).getByRole('button', { name: '鍋島' }))
    await user.click(within(sheet).getByRole('button', { name: 'この条件で検索' }))

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(query(router.state.location.search)).toEqual({
      keyword: '線形',
      category: 'textbook',
      priceMin: '500',
      campus: 'nabeshima',
    })
  })

  it('サーバーが条件を受け付けなかったらエラーを出す', async () => {
    stubItemsFetch({
      '/api/items?priceMin=3000&priceMax=2000': () =>
        json({ code: 'bad_request', message: '価格の下限は上限以下にしてください' }, 400),
    })
    setup('/search?priceMin=3000&priceMax=2000')
    expect(await screen.findByRole('alert')).toHaveTextContent('価格の下限は上限以下にしてください')
  })

  it('戻るボタンで商品一覧へ戻る', async () => {
    stubItemsFetch({ '/api/items': () => json(page([])) })
    setup()
    expect(screen.getByRole('link', { name: '戻る' })).toHaveAttribute('href', '/')
  })
})
