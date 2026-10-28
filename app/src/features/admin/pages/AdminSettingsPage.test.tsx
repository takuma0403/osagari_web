import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { AdminLayout } from '../../../layouts/AdminLayout/AdminLayout'
import type { AdminFeeRate } from '../api'
import { AdminSettingsPage } from './AdminSettingsPage'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const current: AdminFeeRate = {
  feeRateBasisPoints: 500,
  updatedAt: '2026-10-01T05:30:00Z',
  updatedByStudentNumber: '99999999',
}

type Handler = (method: string, init?: RequestInit) => Response | undefined

function setup(handler: Handler = () => undefined) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    if (url !== '/api/admin/settings/fee-rate')
      throw new Error(`想定していない呼び出し: ${method} ${url}`)
    return handler(method, init) ?? json(current)
  })
  vi.stubGlobal('fetch', fetchMock)
  const router = createMemoryRouter(
    [
      {
        element: <AdminLayout />,
        children: [{ path: '/admin/settings', element: <AdminSettingsPage /> }],
      },
    ],
    { initialEntries: ['/admin/settings'] },
  )
  render(<RouterProvider router={router} />)
  return { fetchMock, user: userEvent.setup() }
}

const puts = (fetchMock: ReturnType<typeof setup>['fetchMock']) =>
  fetchMock.mock.calls.filter(([, init]) => init?.method === 'PUT')

describe('AdminSettingsPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('現在の料率と最終更新を出す', async () => {
    setup()
    expect(screen.getByText('読み込み中…')).toBeInTheDocument()
    expect(await screen.findByText('5%')).toBeInTheDocument()
    expect(screen.getByText(/ · 99999999$/)).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: '設定' })).toBeInTheDocument()
    expect(
      screen.getByText('フェーズ1の決済はモックです。実際のお金は動きません。'),
    ).toBeInTheDocument()
  })

  it('管理者がまだ変えていなければ初期値と出す', async () => {
    setup((method) =>
      method === 'GET' ? json({ ...current, updatedByStudentNumber: null }) : undefined,
    )
    expect(await screen.findByText(/ · 初期値$/)).toBeInTheDocument()
  })

  it('入力した % をベーシスポイントにして保存する', async () => {
    const { fetchMock, user } = setup((method) =>
      method === 'PUT'
        ? json({ ...current, feeRateBasisPoints: 525, updatedAt: '2026-10-29T01:00:00Z' })
        : undefined,
    )
    await user.type(await screen.findByLabelText('新しい料率'), '5.25')
    await user.click(screen.getByRole('button', { name: '保存する' }))
    expect(await screen.findByText('手数料率を5.25%に変更しました')).toBeInTheDocument()
    expect(JSON.parse(String(puts(fetchMock)[0][1]?.body))).toEqual({ feeRateBasisPoints: 525 })
    expect(screen.getByText('5.25%')).toBeInTheDocument()
    expect(screen.getByLabelText('新しい料率')).toHaveValue('')
  })

  it.each(['120', '5.255', 'abc', ''])('%s は送らずにエラーを出す', async (value) => {
    const { fetchMock, user } = setup()
    const input = await screen.findByLabelText('新しい料率')
    if (value) await user.type(input, value)
    await user.click(screen.getByRole('button', { name: '保存する' }))
    expect(screen.getByRole('alert')).toHaveTextContent(
      '0〜100 の範囲で、小数点以下2桁までで入力してください。',
    )
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(puts(fetchMock)).toHaveLength(0)
  })

  it('保存に失敗したらサーバーの文を出す', async () => {
    const { user } = setup((method) =>
      method === 'PUT'
        ? json({ code: 'bad_request', message: '手数料率が範囲外です' }, 400)
        : undefined,
    )
    await user.type(await screen.findByLabelText('新しい料率'), '5')
    await user.click(screen.getByRole('button', { name: '保存する' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('手数料率が範囲外です')
  })

  it('読み込みに失敗したら、読み直せる', async () => {
    let failed = false
    const { user } = setup((method) => {
      if (method === 'GET' && !failed) {
        failed = true
        return json({ code: 'internal', message: 'エラーが起きました' }, 500)
      }
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('エラーが起きました')
    await user.click(screen.getByRole('button', { name: 'もう一度読み込む' }))
    expect(await screen.findByText('5%')).toBeInTheDocument()
  })
})
