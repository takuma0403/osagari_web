import { ApiError, isApiError, request } from './client'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

function mockFetch(impl: () => Promise<Response>) {
  const fn = vi.fn(impl)
  vi.stubGlobal('fetch', fn)
  return fn
}

async function catchError(p: Promise<unknown>): Promise<ApiError> {
  const err = await p.catch((e: unknown) => e)
  if (!isApiError(err)) throw new Error(`ApiError ではありません: ${String(err)}`)
  return err
}

describe('request', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('/api を前置し、同一オリジンの Cookie と Accept を付けて呼ぶ', async () => {
    const fetchMock = mockFetch(async () => json({ status: 'ok' }))
    await expect(request('/health')).resolves.toEqual({ status: 'ok' })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/health',
      expect.objectContaining({
        credentials: 'same-origin',
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    )
  })

  it('本文のない成功（204）は undefined を返す', async () => {
    mockFetch(async () => new Response(null, { status: 204 }))
    await expect(request('/x', { method: 'DELETE' })).resolves.toBeUndefined()
  })

  it('Error スキーマの応答は code と message をそのまま使う', async () => {
    mockFetch(async () =>
      json({ code: 'invalid_status_transition', message: 'この取引はすでに完了しています' }, 409),
    )
    const err = await catchError(request('/deals/1/complete', { method: 'POST' }))
    expect(err).toMatchObject({
      status: 409,
      code: 'invalid_status_transition',
      message: 'この取引はすでに完了しています',
    })
  })

  it.each([
    [401, 'unauthorized'],
    [404, 'not_found'],
    [413, 'bad_request'],
    [429, 'too_many_requests'],
    [502, 'internal_error'],
  ])('本文が Error スキーマでない %d は %s として扱う', async (status, code) => {
    mockFetch(async () => new Response('<html>Bad Gateway</html>', { status }))
    const err = await catchError(request('/x'))
    expect(err.status).toBe(status)
    expect(err.code).toBe(code)
    expect(err.message).not.toBe('')
  })

  it('通信に失敗したときは status 0 の network_error を投げる', async () => {
    mockFetch(async () => {
      throw new TypeError('Failed to fetch')
    })
    const err = await catchError(request('/x'))
    expect(err).toMatchObject({ status: 0, code: 'network_error' })
  })

  it('中断（AbortError）は ApiError に変えずにそのまま投げる', async () => {
    mockFetch(async () => {
      throw new DOMException('aborted', 'AbortError')
    })
    await expect(request('/x')).rejects.toMatchObject({ name: 'AbortError' })
  })
})
