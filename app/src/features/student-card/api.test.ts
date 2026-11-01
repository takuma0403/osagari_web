import { createStudentCardUpload, getCurrentStudentCard, submitStudentCard } from './api'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

const pending = {
  status: 'pending',
  submittedAt: '2026-10-08T05:02:00Z',
  reviewedAt: null,
}

describe('student-card api', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('アップロードURLの発行に画像の形式を送る', async () => {
    const fetchMock = vi.fn(async () => json({ objectKey: 'u1/k' }, 201))
    vi.stubGlobal('fetch', fetchMock)
    await expect(createStudentCardUpload('image/webp')).resolves.toEqual({ objectKey: 'u1/k' })
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/student-cards/uploads',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ contentType: 'image/webp' }),
      }),
    )
  })

  it('登録にオブジェクト名を送り、審査状況を返す', async () => {
    const fetchMock = vi.fn(async () => json(pending, 201))
    vi.stubGlobal('fetch', fetchMock)
    await expect(submitStudentCard('u1/k')).resolves.toEqual(pending)
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/student-cards',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ objectKey: 'u1/k' }) }),
    )
  })

  it('審査状況を GET で取得する', async () => {
    const fetchMock = vi.fn(async () => json(pending))
    vi.stubGlobal('fetch', fetchMock)
    await expect(getCurrentStudentCard()).resolves.toEqual(pending)
    expect(fetchMock).toHaveBeenCalledWith('/api/student-cards/current', expect.anything())
  })
})
