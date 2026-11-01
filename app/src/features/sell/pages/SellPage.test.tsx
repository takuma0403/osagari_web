import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { resizeImage } from '../../../lib/imageResize'
import { renderRoute } from '../../../test/renderRoute'
import type { CurrentUser } from '../../auth/api'
import type { OwnItem } from '../api'
import { SellPage } from './SellPage'

vi.mock('../../../lib/imageResize', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../../lib/imageResize')>()),
  resizeImage: vi.fn(),
}))

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const approved = { id: 'u1', sellerStatus: 'approved' } as CurrentUser
const upload = {
  objectKey: 'u1/k1',
  uploadUrl: 'http://localhost:4443/upload/u1/k1?sig=x',
  method: 'PUT',
  headers: { 'Content-Type': 'image/webp' },
  expiresAt: '2026-10-20T05:17:00Z',
}
const item: OwnItem = {
  id: 'i1',
  shareId: 'abc123',
  status: 'draft',
  title: '線形代数入門',
  category: 'textbook',
  condition: 'good',
  price: 1500,
  description: '1年前期で使いました。',
  campus: 'honjo',
  images: [
    {
      objectKey: 'u1/old',
      url: 'http://localhost:4443/u1/old?sig=y',
      expiresAt: '2026-10-20T05:17:00Z',
    },
  ],
  textbook: { courseName: '線形代数学I', isbn: '9784000000000', hasWriting: false },
  furnitureAppliance: null,
  publishedAt: null,
  createdAt: '2026-10-20T05:00:00Z',
  updatedAt: '2026-10-20T05:00:00Z',
}

type Handler = (init?: RequestInit) => Response

/** メソッドと URL で応答を返し分ける fetch を用意して画面を開く */
function setup(
  routes: Record<string, Handler> = {},
  { at = '/sell', user = approved }: { at?: string; user?: CurrentUser } = {},
) {
  const all: Record<string, Handler> = {
    'GET /api/settings/fee-rate': () => json({ feeRateBasisPoints: 500 }),
    'POST /api/items/image-uploads': () => json(upload, 201),
    [`PUT ${upload.uploadUrl}`]: () => new Response(null, { status: 200 }),
    ...routes,
  }
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const handler = all[`${init?.method ?? 'GET'} ${url}`]
    if (!handler) throw new Error(`想定していない呼び出し: ${init?.method ?? 'GET'} ${url}`)
    return handler(init)
  })
  vi.stubGlobal('fetch', fetchMock)
  // 新規は /sell、編集は /sell/:itemId として開く。遷移先は商品ページと（新規なら）編集の画面
  const editing = at !== '/sell'
  const view = renderRoute(editing ? '/sell/:itemId' : '/sell', <SellPage />, {
    at,
    session: { user },
    destinations: editing ? ['/items/:shareId'] : ['/items/:shareId', '/sell/:itemId'],
  })
  return { ...view, user: userEvent.setup(), fetchMock }
}

/** fetch に送った本文（JSON） */
function bodyOf(fetchMock: ReturnType<typeof vi.fn>, key: string) {
  const call = fetchMock.mock.calls.find(([url, init]) => `${init?.method ?? 'GET'} ${url}` === key)
  return call ? JSON.parse(call[1].body as string) : undefined
}

async function fillRequired(user: ReturnType<typeof userEvent.setup>) {
  await user.upload(
    screen.getByLabelText('商品の写真'),
    new File(['raw'], 'a.jpg', { type: 'image/jpeg' }),
  )
  await screen.findByRole('button', { name: '写真1を並べ替え・削除' })
  await user.type(screen.getByLabelText('タイトル'), '線形代数入門')
  await user.selectOptions(screen.getByLabelText('カテゴリ'), '教科書・参考書')
  await user.selectOptions(screen.getByLabelText('状態'), '目立った傷なし')
  await user.type(screen.getByLabelText('価格'), '1550')
  await user.click(screen.getByRole('radio', { name: '本庄' }))
  await user.type(screen.getByLabelText('説明'), '書き込みはありません。')
}

describe('SellPage', () => {
  beforeEach(() => {
    vi.mocked(resizeImage).mockResolvedValue({
      blob: new Blob(['webp'], { type: 'image/webp' }),
      contentType: 'image/webp',
    })
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it.each([
    ['none', '学生証を登録する'],
    ['pending', '審査状況を見る'],
  ] as const)(
    '学生証が承認されていない（%s）ときは学生証の画面へ案内する',
    (sellerStatus, link) => {
      setup({}, { user: { ...approved, sellerStatus } })
      expect(screen.getByRole('link', { name: link })).toHaveAttribute(
        'href',
        '/mypage/student-card',
      )
      expect(screen.queryByLabelText('タイトル')).not.toBeInTheDocument()
    },
  )

  it('価格から手数料を引いた受け取り見込みを出す（1円未満は切り捨て）', async () => {
    const { user } = setup()
    await user.type(screen.getByLabelText('価格'), '1,550円')
    expect(screen.getByLabelText('価格')).toHaveValue('1550')
    expect(await screen.findByText('手数料 5%')).toBeInTheDocument()
    expect(screen.getByText('受け取り見込み ¥1,473')).toBeInTheDocument()
  })

  it('手数料率を取得できないときは見込みを出さずに出品できる', async () => {
    setup({
      'GET /api/settings/fee-rate': () => json({ code: 'internal_error', message: 'x' }, 500),
    })
    expect(await screen.findByLabelText('タイトル')).toBeInTheDocument()
    expect(screen.queryByText(/受け取り見込み/)).not.toBeInTheDocument()
  })

  it('公開するときに足りない項目を示し、送らない', async () => {
    const { user, fetchMock } = setup()
    await user.click(screen.getByRole('button', { name: '公開する' }))
    expect(screen.getByText('写真を1枚以上追加してください')).toBeInTheDocument()
    expect(screen.getByText('タイトルを入力してください')).toBeInTheDocument()
    expect(screen.getByText('カテゴリを選んでください')).toBeInTheDocument()
    expect(screen.getByText('受け渡し希望キャンパスを選んでください')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith('/api/items', expect.anything())
  })

  it('写真を縮小して直接アップロードし、公開したら商品ページへ移る', async () => {
    const { user, fetchMock, router } = setup({
      'POST /api/items': () => json({ ...item, status: 'published' }, 201),
    })
    await fillRequired(user)
    await user.type(screen.getByLabelText('ISBN'), '978-4-00-000000-0')
    await user.click(screen.getByRole('button', { name: '公開する' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/items/abc123'))
    expect(bodyOf(fetchMock, 'POST /api/items/image-uploads')).toEqual({
      contentType: 'image/webp',
    })
    expect(bodyOf(fetchMock, 'POST /api/items')).toEqual({
      title: '線形代数入門',
      category: 'textbook',
      condition: 'good',
      price: 1550,
      description: '書き込みはありません。',
      campus: 'honjo',
      imageObjectKeys: ['u1/k1'],
      textbook: { isbn: '9784000000000', hasWriting: false },
      publish: true,
    })
  })

  it('カテゴリを変えたら、前のカテゴリの追加項目を送らない', async () => {
    const { user, fetchMock } = setup({
      'POST /api/items': () => json({ ...item, status: 'published' }, 201),
    })
    await fillRequired(user)
    await user.type(screen.getByLabelText('授業名'), '線形代数学I')
    await user.selectOptions(screen.getByLabelText('カテゴリ'), '家具・家電')
    await user.type(screen.getByLabelText('サイズ'), '幅60cm')
    await user.click(screen.getByRole('button', { name: '公開する' }))

    await waitFor(() => expect(bodyOf(fetchMock, 'POST /api/items')).toBeDefined())
    const body = bodyOf(fetchMock, 'POST /api/items')
    expect(body.textbook).toBeUndefined()
    expect(body.furnitureAppliance).toEqual({ size: '幅60cm', operationChecked: false })
  })

  it('タイトルだけで下書き保存でき、編集の画面へ移ってトーストを出す', async () => {
    const { user, fetchMock, router } = setup({
      'POST /api/items': () => json({ ...item, images: [] }, 201),
    })
    await user.type(screen.getByLabelText('タイトル'), '線形代数入門')
    await user.click(screen.getByRole('button', { name: '下書き保存' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/sell/i1'))
    expect(router.state.location.state).toEqual({ toast: '下書きを保存しました' })
    expect(bodyOf(fetchMock, 'POST /api/items')).toEqual({
      title: '線形代数入門',
      imageObjectKeys: [],
      publish: false,
    })
  })

  it('写真のアップロードに失敗したら写真を外してエラーを出す', async () => {
    const { user } = setup({
      [`PUT ${upload.uploadUrl}`]: () => new Response(null, { status: 403 }),
    })
    await user.upload(
      screen.getByLabelText('商品の写真'),
      new File(['raw'], 'a.jpg', { type: 'image/jpeg' }),
    )
    expect(
      await screen.findByText('画像をアップロードできませんでした。もう一度お試しください'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^写真1/ })).not.toBeInTheDocument()
  })

  it('サーバーのエラーをそのまま出す', async () => {
    const { user } = setup({
      'POST /api/items': () =>
        json({ code: 'invalid_item_image_upload', message: '写真をもう一度追加してください' }, 400),
    })
    await fillRequired(user)
    await user.click(screen.getByRole('button', { name: '公開する' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('写真をもう一度追加してください')
  })

  describe('編集', () => {
    it('自分の商品を読み込んで入力欄に入れる', async () => {
      setup({ 'GET /api/items/i1': () => json(item) }, { at: '/sell/i1' })
      expect(screen.getByRole('status')).toHaveTextContent('読み込み中…')
      expect(await screen.findByLabelText('タイトル')).toHaveValue('線形代数入門')
      expect(screen.getByRole('heading', { level: 1, name: '出品を編集' })).toBeInTheDocument()
      expect(screen.getByLabelText('カテゴリ')).toHaveValue('textbook')
      expect(screen.getByLabelText('授業名')).toHaveValue('線形代数学I')
      expect(screen.getByLabelText('価格')).toHaveValue('1500')
      expect(screen.getByRole('radio', { name: '本庄' })).toBeChecked()
      expect(screen.getByRole('button', { name: '写真1を並べ替え・削除' })).toBeInTheDocument()
    })

    it('下書きの公開は、編集を保存してから公開する', async () => {
      const { user, fetchMock, router } = setup(
        {
          'GET /api/items/i1': () => json(item),
          'PUT /api/items/i1': () => json(item),
          'POST /api/items/i1/publish': () => json({ ...item, status: 'published' }),
        },
        { at: '/sell/i1' },
      )
      await user.click(await screen.findByRole('button', { name: '公開する' }))
      await waitFor(() => expect(router.state.location.pathname).toBe('/items/abc123'))
      expect(bodyOf(fetchMock, 'PUT /api/items/i1').imageObjectKeys).toEqual(['u1/old'])
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/items/i1/publish',
        expect.objectContaining({ method: 'POST' }),
      )
    })

    it('取り下げた商品は「保存する」で取り下げのまま保存し、トーストを出す', async () => {
      const withdrawn = { ...item, status: 'withdrawn' as const }
      const { user, fetchMock } = setup(
        { 'GET /api/items/i1': () => json(withdrawn), 'PUT /api/items/i1': () => json(withdrawn) },
        { at: '/sell/i1' },
      )
      await user.click(await screen.findByRole('button', { name: '保存する' }))
      expect(await screen.findByText('保存しました')).toBeInTheDocument()
      expect(fetchMock).not.toHaveBeenCalledWith('/api/items/i1/publish', expect.anything())
    })

    it('公開中の商品は「保存する」だけで、すべての共通項目が必要', async () => {
      const published = { ...item, status: 'published' as const }
      const { user, fetchMock } = setup(
        { 'GET /api/items/i1': () => json(published) },
        { at: '/sell/i1' },
      )
      await user.clear(await screen.findByLabelText('説明'))
      expect(screen.queryByRole('button', { name: '公開する' })).not.toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: '保存する' }))
      expect(screen.getByText('説明を入力してください')).toBeInTheDocument()
      expect(fetchMock).not.toHaveBeenCalledWith(
        '/api/items/i1',
        expect.objectContaining({ method: 'PUT' }),
      )
    })

    it('取引中の商品は編集できない', async () => {
      setup({ 'GET /api/items/i1': () => json({ ...item, status: 'in_deal' }) }, { at: '/sell/i1' })
      expect(await screen.findByText('取引中のため編集できません。')).toBeInTheDocument()
      expect(screen.getByLabelText('タイトル')).toBeDisabled()
      expect(screen.queryByRole('button', { name: '保存する' })).not.toBeInTheDocument()
    })

    it('ほかの人の商品・ない商品は読み込めない旨を出す', async () => {
      setup(
        {
          'GET /api/items/i1': () =>
            json({ code: 'not_found', message: '商品が見つかりません' }, 404),
        },
        { at: '/sell/i1' },
      )
      const alert = await screen.findByRole('alert')
      expect(alert).toHaveTextContent('商品を読み込めませんでした')
      expect(within(alert).getByRole('button', { name: 'もう一度読み込む' })).toBeInTheDocument()
    })
  })
})
