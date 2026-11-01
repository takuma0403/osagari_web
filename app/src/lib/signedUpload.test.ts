import { isApiError } from '../api/client'
import { uploadToSignedUrl, type SignedUploadTarget } from './signedUpload'

const target: SignedUploadTarget = {
  uploadUrl: 'http://localhost:4443/student-cards/u1/abc?X-Goog-Signature=x',
  method: 'PUT',
  headers: {
    'Content-Type': 'image/webp',
    'x-goog-if-generation-match': '0',
    'x-goog-content-length-range': '0,5242880',
  },
}

function mockFetch(impl: () => Promise<Response>) {
  const fn = vi.fn(impl)
  vi.stubGlobal('fetch', fn)
  return fn
}

describe('uploadToSignedUrl', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('発行されたメソッドとヘッダーをすべてそのまま付けて送る（STORAGE）', async () => {
    const fetchMock = mockFetch(async () => new Response(null, { status: 200 }))
    const blob = new Blob(['x'], { type: 'image/webp' })
    await uploadToSignedUrl(target, blob)
    expect(fetchMock).toHaveBeenCalledWith(target.uploadUrl, {
      method: 'PUT',
      headers: target.headers,
      body: blob,
      credentials: 'omit',
    })
  })

  it('2xx 以外は upload_failed の ApiError を投げる', async () => {
    mockFetch(async () => new Response('<Error/>', { status: 412 }))
    const err = await uploadToSignedUrl(target, new Blob(['x'])).catch((e: unknown) => e)
    expect(isApiError(err)).toBe(true)
    expect(err).toMatchObject({
      status: 412,
      code: 'upload_failed',
      message: '画像をアップロードできませんでした。もう一度お試しください',
    })
  })

  it('通信できなかったときは network_error の ApiError を投げる', async () => {
    mockFetch(async () => {
      throw new TypeError('Failed to fetch')
    })
    await expect(uploadToSignedUrl(target, new Blob(['x']))).rejects.toMatchObject({
      status: 0,
      code: 'network_error',
    })
  })
})
