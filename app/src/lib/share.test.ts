import { copyText, shareText } from './share'

describe('shareText', () => {
  it('「【お下がり】商品名 ¥価格（状態） URL」の文にする', () => {
    expect(
      shareText({
        title: '線形代数入門',
        price: 1500,
        note: '書き込みなし',
        url: 'https://example.com/items/abc123',
      }),
    ).toBe('【お下がり】線形代数入門 ¥1,500（書き込みなし） https://example.com/items/abc123')
  })
})

describe('copyText', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('クリップボードに書き込む', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    await copyText('abc')
    expect(writeText).toHaveBeenCalledWith('abc')
  })

  it('クリップボードが使えなければ reject する', async () => {
    vi.stubGlobal('navigator', {})
    await expect(copyText('abc')).rejects.toThrow()
  })
})
