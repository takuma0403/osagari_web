import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderRoute } from '../../../test/renderRoute'
import { json, publicItem, stubItemsFetch } from '../test/fixtures'
import { DetailPage } from './DetailPage'

function setup(at = '/items/abc123', state?: unknown) {
  const view = renderRoute('/items/:shareId', <DetailPage />, {
    at,
    state,
    destinations: ['/', '/items/:shareId/request', '/sell/:itemId', '/sell/:itemId/requests'],
  })
  // user-event はクリップボードを差し替えるので、コピーした文は navigator.clipboard.readText で読む
  return { ...view, user: userEvent.setup() }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('DetailPage', () => {
  it('商品の情報と出品者の学部・入学年度を出し、学籍番号は出さない', async () => {
    const fetchMock = stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem()) })
    setup()
    expect(
      await screen.findByRole('heading', { level: 1, name: '線形代数入門' }),
    ).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/items/shared/abc123', expect.anything())
    expect(screen.getByText('教科書・参考書')).toBeInTheDocument()
    expect(screen.getByText('¥1,500')).toBeInTheDocument()
    expect(screen.getByText('線形代数学I')).toBeInTheDocument()
    expect(screen.getByText('佐賀 太郎')).toBeInTheDocument()
    expect(screen.getByText('9784000000000')).toBeInTheDocument()
    expect(screen.getByText('目立った傷なし')).toBeInTheDocument()
    expect(screen.getByText('本庄キャンパス')).toBeInTheDocument()
    expect(screen.getByText('1年前期で使いました。書き込みはありません。')).toBeInTheDocument()
    expect(screen.getByText('理工学部 · 2025年度入学')).toBeInTheDocument()
    expect(screen.getByText('学籍番号は取引成立後に表示されます')).toBeInTheDocument()
    expect(document.body).not.toHaveTextContent('@edu.cc.saga-u.ac.jp')
  })

  it('家具・家電は値のある追加項目だけを出す', async () => {
    stubItemsFetch({
      '/api/items/shared/abc123': () =>
        json(
          publicItem({
            category: 'furniture_appliance',
            textbook: null,
            furnitureAppliance: { size: '幅60cm', operationChecked: true },
          }),
        ),
    })
    setup()
    expect(await screen.findByText('幅60cm')).toBeInTheDocument()
    expect(screen.getByText('確認済み')).toBeInTheDocument()
    expect(screen.queryByText('授業名')).not.toBeInTheDocument()
    expect(screen.queryByText('ISBN')).not.toBeInTheDocument()
  })

  it('写真を何枚目か分かる alt 付きで出す', async () => {
    const image = (n: number) => ({
      objectKey: `u1/k${n}`,
      url: `http://localhost:4443/u1/k${n}`,
      expiresAt: '2026-10-22T05:00:00Z',
    })
    stubItemsFetch({
      '/api/items/shared/abc123': () => json(publicItem({ images: [image(1), image(2)] })),
    })
    setup()
    expect(await screen.findByAltText('商品の写真 1 / 2')).toHaveAttribute(
      'src',
      'http://localhost:4443/u1/k1',
    )
    expect(screen.getByAltText('商品の写真 2 / 2')).toBeInTheDocument()
  })

  it('購入希望を出す画面へ進める', async () => {
    stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem()) })
    setup()
    expect(await screen.findByRole('link', { name: '購入希望を出す' })).toHaveAttribute(
      'href',
      '/items/abc123/request',
    )
  })

  it('自分の出品では購入希望の代わりに編集画面へ進める', async () => {
    stubItemsFetch({
      '/api/items/shared/abc123': () => json(publicItem({ isOwn: true, itemId: 'item1' })),
    })
    const { user, router } = setup()
    const edit = await screen.findByRole('link', { name: '出品を編集する' })
    expect(edit).toHaveAttribute('href', '/sell/item1')
    expect(screen.queryByRole('link', { name: '購入希望を出す' })).not.toBeInTheDocument()

    await user.click(edit)
    expect(router.state.location.pathname).toBe('/sell/item1')
  })

  it('購入希望ありの自分の出品では、編集の代わりに購入希望の一覧へ進める', async () => {
    stubItemsFetch({
      '/api/items/shared/abc123': () =>
        json(publicItem({ isOwn: true, itemId: 'item1', status: 'requested' })),
    })
    setup()
    expect(await screen.findByRole('link', { name: '購入希望を見る' })).toHaveAttribute(
      'href',
      '/sell/item1/requests',
    )
    expect(screen.queryByRole('link', { name: '出品を編集する' })).not.toBeInTheDocument()
  })

  it('購入希望を送って戻ってきたらトーストで知らせる', async () => {
    stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem()) })
    setup('/items/abc123', { requestSent: true })
    expect(await screen.findByText('購入希望を送りました')).toBeInTheDocument()
  })

  it('自分の出品でも商品IDがなければ押せない', async () => {
    stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem({ isOwn: true })) })
    setup()
    expect(await screen.findByRole('button', { name: '自分の出品です' })).toBeDisabled()
    expect(screen.queryByRole('link', { name: '出品を編集する' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: '購入希望を出す' })).not.toBeInTheDocument()
  })

  it('売り切れの自分の出品は編集へ進めない', async () => {
    stubItemsFetch({
      '/api/items/shared/abc123': () =>
        json(publicItem({ isOwn: true, itemId: 'item1', status: 'in_deal' })),
    })
    setup()
    expect(await screen.findByRole('button', { name: '売り切れ' })).toBeDisabled()
    expect(screen.queryByRole('link', { name: '出品を編集する' })).not.toBeInTheDocument()
  })

  it.each(['in_deal', 'completed'] as const)(
    '%s の商品は売り切れとして出し、購入希望を出せない',
    async (status) => {
      stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem({ status })) })
      setup()
      expect(await screen.findByRole('button', { name: '売り切れ' })).toBeDisabled()
      expect(screen.queryByRole('link', { name: '購入希望を出す' })).not.toBeInTheDocument()
    },
  )

  it('共有シートで、商品名・価格・状態と URL をまとめた文をコピーする', async () => {
    stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem()) })
    const { user } = setup()
    await user.click(await screen.findByRole('button', { name: 'リンクをコピー' }))
    const sheet = screen.getByRole('dialog', { name: 'リンクをコピー' })
    expect(
      within(sheet).getByText('リンクと、商品の簡単な説明のテキストがコピーされます。'),
    ).toBeInTheDocument()
    expect(within(sheet).queryByRole('button', { name: /ほかのアプリで共有/ })).toBeNull()

    await user.click(within(sheet).getByRole('button', { name: 'リンクをコピー' }))
    expect(await screen.findByRole('status')).toHaveTextContent('リンクをコピーしました')
    expect(await navigator.clipboard.readText()).toBe(
      `【お下がり】線形代数入門 ¥1,500（書き込みなし） ${window.location.origin}/items/abc123`,
    )

    await user.click(within(sheet).getByRole('button', { name: '閉じる' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('コピーできなければシートにエラーを出す', async () => {
    stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem()) })
    const { user } = setup()
    vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(new Error('denied'))
    await user.click(await screen.findByRole('button', { name: 'リンクをコピー' }))
    const sheet = screen.getByRole('dialog', { name: 'リンクをコピー' })
    await user.click(within(sheet).getByRole('button', { name: 'リンクをコピー' }))
    expect(await within(sheet).findByRole('alert')).toHaveTextContent('コピーできませんでした')
  })

  it('見つからない商品は「商品が見つかりません」を出す', async () => {
    stubItemsFetch({
      '/api/items/shared/abc123': () => json({ code: 'not_found', message: '見つかりません' }, 404),
    })
    setup()
    expect(
      await screen.findByRole('heading', { level: 1, name: '商品が見つかりません' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '商品一覧へ戻る' })).toHaveAttribute('href', '/')
  })

  it('読み込めなければエラーを出し、読み込み直せる', async () => {
    let calls = 0
    stubItemsFetch({
      '/api/items/shared/abc123': () =>
        ++calls === 1
          ? json({ code: 'internal_error', message: 'サーバーでエラーが発生しました' }, 500)
          : json(publicItem()),
    })
    const { user } = setup()
    expect(await screen.findByRole('alert')).toHaveTextContent('サーバーでエラーが発生しました')
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(
      await screen.findByRole('heading', { level: 1, name: '線形代数入門' }),
    ).toBeInTheDocument()
  })

  it('共有リンクから直接開いたときは、戻るで商品一覧へ移る', async () => {
    stubItemsFetch({ '/api/items/shared/abc123': () => json(publicItem()) })
    const { user, router } = setup()
    await screen.findByRole('heading', { level: 1, name: '線形代数入門' })
    await user.click(screen.getByRole('button', { name: '戻る' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
  })
})
