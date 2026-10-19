import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ImageResizeError, resizeImage } from '../../../lib/imageResize'
import { renderRoute } from '../../../test/renderRoute'
import type { CurrentUser } from '../../auth/api'
import type { StudentCardState } from '../api'
import { StudentCardPage } from './StudentCardPage'

vi.mock('../../../lib/imageResize', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../../lib/imageResize')>()),
  resizeImage: vi.fn(),
}))

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const none: StudentCardState = { status: 'none', submittedAt: null, reviewedAt: null }
const pending: StudentCardState = {
  status: 'pending',
  submittedAt: '2026-10-08T05:02:00Z',
  reviewedAt: null,
}
const upload = {
  objectKey: 'u1/k1',
  uploadUrl: 'http://localhost:4443/upload/u1/k1?sig=x',
  method: 'PUT',
  headers: {
    'Content-Type': 'image/webp',
    'x-goog-if-generation-match': '0',
    'x-goog-content-length-range': '0,5242880',
  },
  expiresAt: '2026-10-08T05:17:00Z',
}
const me = { id: 'u1', sellerStatus: 'none' } as CurrentUser
const resized = new Blob(['webp'], { type: 'image/webp' })

type Routes = {
  current?: () => Response
  uploads?: () => Response
  put?: () => Response
  submit?: () => Response
}

/** URL とメソッドで応答を返し分ける fetch を用意して画面を開く */
function setup(routes: Routes) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    if (url === '/api/student-cards/current') return (routes.current ?? (() => json(none)))()
    if (url === '/api/student-cards/uploads') return (routes.uploads ?? (() => json(upload, 201)))()
    if (url === upload.uploadUrl && init?.method === 'PUT')
      return (routes.put ?? (() => new Response(null, { status: 200 })))()
    if (url === '/api/student-cards') return (routes.submit ?? (() => json(pending, 201)))()
    throw new Error(`想定していない呼び出し: ${init?.method ?? 'GET'} ${url}`)
  })
  vi.stubGlobal('fetch', fetchMock)
  const user = userEvent.setup()
  const view = renderRoute('/mypage/student-card', <StudentCardPage />, { session: { user: me } })
  return { ...view, user, fetchMock }
}

async function selectImage(user: ReturnType<typeof userEvent.setup>) {
  const file = new File(['raw'], 'card.jpg', { type: 'image/jpeg' })
  await user.upload(screen.getByLabelText('学生証の画像'), file)
}

describe('StudentCardPage', () => {
  beforeEach(() => {
    vi.mocked(resizeImage).mockResolvedValue({ blob: resized, contentType: 'image/webp' })
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it('審査状況を読み込むまでは読み込み中を出す', async () => {
    setup({})
    expect(screen.getByRole('status')).toHaveTextContent('読み込み中…')
    expect(await screen.findByRole('button', { name: '学生証の画像を選ぶ' })).toBeInTheDocument()
  })

  it('画像を選ぶと縮小・URL発行・直接PUT・登録の順に行い、「審査中」を表示する（CARD, STORAGE）', async () => {
    const { user, fetchMock, session } = setup({})
    await screen.findByRole('button', { name: '学生証の画像を選ぶ' })
    await selectImage(user)

    expect(await screen.findByText('審査中')).toBeInTheDocument()
    expect(screen.getByText('管理者の確認待ち')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: '登録した学生証の画像' })).toHaveAttribute(
      'src',
      'blob:preview',
    )
    expect(screen.getByRole('button', { name: '撮り直す' })).toBeEnabled()

    const calls = fetchMock.mock.calls.map(([url]) => url)
    expect(calls).toEqual([
      '/api/student-cards/current',
      '/api/student-cards/uploads',
      upload.uploadUrl,
      '/api/student-cards',
    ])
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/student-cards/uploads',
      expect.objectContaining({ body: JSON.stringify({ contentType: 'image/webp' }) }),
    )
    // 署名に含まれるヘッダーをすべてそのまま付ける
    expect(fetchMock).toHaveBeenCalledWith(
      upload.uploadUrl,
      expect.objectContaining({ method: 'PUT', headers: upload.headers, body: resized }),
    )
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/student-cards',
      expect.objectContaining({ body: JSON.stringify({ objectKey: 'u1/k1' }) }),
    )
    expect(URL.createObjectURL).toHaveBeenCalledWith(resized)
    expect(session.signIn).toHaveBeenCalledWith({ ...me, sellerStatus: 'pending' })
  })

  it('審査中は撮り直せる。再読み込み後は画像を出さない', async () => {
    setup({ current: () => json(pending) })
    expect(await screen.findByText('審査中')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '撮り直す' })).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('差し戻しは理由の案内と撮り直しを出す', async () => {
    setup({
      current: () => json({ ...pending, status: 'rejected', reviewedAt: '2026-10-09T01:30:00Z' }),
    })
    expect(
      await screen.findByText('学生証を確認できませんでした。撮り直して提出してください。'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '撮り直す' })).toBeInTheDocument()
  })

  it('承認済みは撮り直しを出さず、出品へ案内する', async () => {
    setup({
      current: () => json({ ...pending, status: 'approved', reviewedAt: '2026-10-09T01:30:00Z' }),
    })
    expect(await screen.findByRole('link', { name: '出品する' })).toHaveAttribute('href', '/sell')
    expect(screen.queryByRole('button', { name: '撮り直す' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('学生証の画像')).not.toBeInTheDocument()
  })

  it('登録できなかったときはサーバーの message を表示する', async () => {
    const message = '学生証の画像を確認できませんでした。撮り直してください'
    const { user } = setup({
      submit: () => json({ code: 'invalid_student_card_upload', message }, 400),
    })
    await screen.findByRole('button', { name: '学生証の画像を選ぶ' })
    await selectImage(user)
    expect(await screen.findByRole('alert')).toHaveTextContent(message)
    expect(screen.getByRole('button', { name: '学生証の画像を選ぶ' })).toBeEnabled()
  })

  it('PUT に失敗したときは登録せずにエラーを表示する', async () => {
    const { user, fetchMock } = setup({ put: () => new Response(null, { status: 403 }) })
    await screen.findByRole('button', { name: '学生証の画像を選ぶ' })
    await selectImage(user)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      '画像をアップロードできませんでした。もう一度お試しください',
    )
    expect(fetchMock).not.toHaveBeenCalledWith('/api/student-cards', expect.anything())
  })

  it('画像を読み込めなかったときは API を呼ばずにエラーを表示する', async () => {
    vi.mocked(resizeImage).mockRejectedValue(new ImageResizeError('この画像は読み込めませんでした'))
    const { user, fetchMock } = setup({})
    await screen.findByRole('button', { name: '学生証の画像を選ぶ' })
    await selectImage(user)
    expect(await screen.findByRole('alert')).toHaveTextContent('この画像は読み込めませんでした')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('承認済みになっていた（409）ときは審査状況を取り直す', async () => {
    let current = none
    const { user } = setup({
      current: () => json(current),
      uploads: () => {
        current = { ...pending, status: 'approved', reviewedAt: '2026-10-09T01:30:00Z' }
        return json({ code: 'invalid_status_transition', message: 'すでに承認されています' }, 409)
      },
    })
    await screen.findByRole('button', { name: '学生証の画像を選ぶ' })
    await selectImage(user)
    expect(await screen.findByRole('link', { name: '出品する' })).toBeInTheDocument()
  })

  it('審査状況を取得できなかったときは読み込み直せる', async () => {
    let fail = true
    const { user } = setup({
      current: () =>
        fail ? json({ code: 'internal_error', message: 'サーバーでエラー' }, 500) : json(pending),
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('サーバーでエラー')
    fail = false
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByText('審査中')).toBeInTheDocument()
  })
})
