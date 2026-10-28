import {
  approveStudentCard,
  cancelTransactionByAdmin,
  deleteUser,
  getAdminFeeRate,
  listAdminItems,
  listAdminTransactions,
  listAdminUsers,
  suspendItem,
  suspendUser,
  unsuspendUser,
  updateFeeRate,
  updateUserExpiresOn,
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
    ['ユーザーの一覧', () => listAdminUsers({}), 'GET', '/api/admin/users'],
    [
      'ユーザーの検索と絞り込み',
      () => listAdminUsers({ q: '2523', filter: 'suspended' }, 'c2'),
      'GET',
      '/api/admin/users?q=2523&filter=suspended&cursor=c2',
    ],
    [
      'すべてのユーザー（filter を付けない）',
      () => listAdminUsers({ filter: 'all' }),
      'GET',
      '/api/admin/users',
    ],
    ['アカウントの停止', () => suspendUser('u 1'), 'POST', '/api/admin/users/u%201/suspend'],
    ['停止の解除', () => unsuspendUser('u1'), 'POST', '/api/admin/users/u1/unsuspend'],
    [
      '有効期限の変更',
      () => updateUserExpiresOn('u1', '2030-03-31'),
      'PUT',
      '/api/admin/users/u1/expires-on',
    ],
    ['退学による削除', () => deleteUser('u1'), 'POST', '/api/admin/users/u1/delete'],
    ['出品の一覧', () => listAdminItems('c1'), 'GET', '/api/admin/items?cursor=c1'],
    ['出品の停止', () => suspendItem('i1'), 'POST', '/api/admin/items/i1/suspend'],
    ['取引の一覧', () => listAdminTransactions(), 'GET', '/api/admin/transactions'],
    [
      '取引のキャンセル',
      () => cancelTransactionByAdmin('t1'),
      'POST',
      '/api/admin/transactions/t1/cancel',
    ],
    ['手数料率の取得', () => getAdminFeeRate(), 'GET', '/api/admin/settings/fee-rate'],
    ['手数料率の変更', () => updateFeeRate(525), 'PUT', '/api/admin/settings/fee-rate'],
  ])('%s を呼ぶ', async (_, call, method, url) => {
    const fetchMock = vi.fn(async (_url: string, _init?: RequestInit) => json({}))
    vi.stubGlobal('fetch', fetchMock)
    await call()
    expect(fetchMock).toHaveBeenCalledWith(url, expect.anything())
    expect(fetchMock.mock.calls[0][1]?.method ?? 'GET').toBe(method)
  })

  it.each([
    ['有効期限', () => updateUserExpiresOn('u1', '2030-03-31'), { expiresOn: '2030-03-31' }],
    ['手数料率', () => updateFeeRate(525), { feeRateBasisPoints: 525 }],
  ])('%s の変更は JSON の本文で送る', async (_, call, body) => {
    const fetchMock = vi.fn(async (_url: string, _init?: RequestInit) => json({}))
    vi.stubGlobal('fetch', fetchMock)
    await call()
    expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual(body)
  })
})
