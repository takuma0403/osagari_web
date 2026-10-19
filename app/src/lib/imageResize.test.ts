import { fitWithin, isImageResizeError, resizeImage } from './imageResize'

describe('fitWithin', () => {
  it('横長は幅を1600pxにして縦横比を保つ', () => {
    expect(fitWithin(4032, 3024)).toEqual({ width: 1600, height: 1200 })
  })

  it('縦長は高さを1600pxにして縦横比を保つ', () => {
    expect(fitWithin(3024, 4032)).toEqual({ width: 1200, height: 1600 })
  })

  it('長辺が上限以下なら拡大しない', () => {
    expect(fitWithin(800, 600)).toEqual({ width: 800, height: 600 })
  })

  it('上限を指定できる', () => {
    expect(fitWithin(1000, 500, 400)).toEqual({ width: 400, height: 200 })
  })
})

describe('resizeImage', () => {
  const ctx = { fillStyle: '', fillRect: vi.fn(), drawImage: vi.fn() }
  const close = vi.fn()

  /** canvas.toBlob が返す Blob を、要求された形式と品質から決める */
  function mockToBlob(make: (type: string, quality: number) => Blob | null) {
    return vi
      .spyOn(HTMLCanvasElement.prototype, 'toBlob')
      .mockImplementation((callback, type, quality) =>
        callback(make(type ?? 'image/png', quality ?? 1)),
      )
  }

  beforeEach(() => {
    vi.stubGlobal(
      'createImageBitmap',
      vi.fn(async () => ({ width: 4032, height: 3024, close })),
    )
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      ctx as unknown as CanvasRenderingContext2D,
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('長辺1600pxに縮小して WebP で書き出す', async () => {
    mockToBlob((type) => new Blob(['x'], { type }))
    const result = await resizeImage(new Blob(['raw']))
    expect(result.contentType).toBe('image/webp')
    expect(result.blob.type).toBe('image/webp')
    expect(ctx.drawImage).toHaveBeenCalledWith(expect.anything(), 0, 0, 1600, 1200)
    expect(close).toHaveBeenCalled()
  })

  it('WebP で書き出せないブラウザでは JPEG にする', async () => {
    mockToBlob((type) => new Blob(['x'], { type: type === 'image/webp' ? 'image/png' : type }))
    const result = await resizeImage(new Blob(['raw']))
    expect(result.contentType).toBe('image/jpeg')
    expect(result.blob.type).toBe('image/jpeg')
  })

  it('上限を超えたら品質を下げて圧縮し直す', async () => {
    const toBlob = mockToBlob(
      (type, quality) => new Blob([quality > 0.6 ? 'x'.repeat(20) : 'x'], { type }),
    )
    const result = await resizeImage(new Blob(['raw']), { maxBytes: 10 })
    expect(result.blob.size).toBe(1)
    expect(toBlob).toHaveBeenCalledTimes(3)
  })

  it('最低の品質でも上限を超えるときは ImageResizeError を投げる', async () => {
    mockToBlob((type) => new Blob(['xx'], { type }))
    const err = await resizeImage(new Blob(['raw']), { maxBytes: 1 }).catch((e: unknown) => e)
    expect(isImageResizeError(err)).toBe(true)
    expect(close).toHaveBeenCalled()
  })

  it('画像として読めないときは ImageResizeError を投げる', async () => {
    vi.stubGlobal(
      'createImageBitmap',
      vi.fn(async () => {
        throw new DOMException('decode', 'InvalidStateError')
      }),
    )
    const err = await resizeImage(new Blob(['heic'])).catch((e: unknown) => e)
    expect(isImageResizeError(err)).toBe(true)
    expect((err as Error).message).toBe(
      'この画像は読み込めませんでした。JPEG・PNG・WebP の画像を選んでください',
    )
  })
})
