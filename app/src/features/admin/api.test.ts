import {
  approveStudentCard,
  createStudentCardView,
  getAdminSession,
  listPendingStudentCards,
  listStudentCardViews,
  rejectStudentCard,
} from './api'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

describe('admin api', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('管理者なら 204 で成功する', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(null, { status: 204 })),
    )
    await expect(getAdminSession()).resolves.toBeUndefined()
  })

  it('管理者でなければ not_found で失敗する', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => json({ code: 'not_found', message: '見つかりません' }, 404)),
    )
    await expect(getAdminSession()).rejects.toMatchObject({ status: 404, code: 'not_found' })
  })

  it.each([
    ['審査待ちの一覧', () => listPendingStudentCards(), 'GET', '/api/admin/student-cards'],
    [
      '閲覧URLの発行',
      () => createStudentCardView('c 1'),
      'POST',
      '/api/admin/student-cards/c%201/views',
    ],
    ['閲覧記録', () => listStudentCardViews('c1'), 'GET', '/api/admin/student-cards/c1/views'],
    ['承認', () => approveStudentCard('c1'), 'POST', '/api/admin/student-cards/c1/approve'],
    ['差し戻し', () => rejectStudentCard('c1'), 'POST', '/api/admin/student-cards/c1/reject'],
  ])('%s を呼ぶ', async (_, call, method, url) => {
    const fetchMock = vi.fn(async (_url: string, _init?: RequestInit) => json({}))
    vi.stubGlobal('fetch', fetchMock)
    await call()
    expect(fetchMock).toHaveBeenCalledWith(url, expect.anything())
    expect(fetchMock.mock.calls[0][1]?.method ?? 'GET').toBe(method)
  })
})
