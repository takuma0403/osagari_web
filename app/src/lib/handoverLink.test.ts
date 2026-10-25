import { buildHandoverUrl, parseHandoverFragment, parseHandoverQr } from './handoverLink'

describe('buildHandoverUrl', () => {
  it('QR は #q=、完了リンクは #t= にトークンを載せる', () => {
    expect(buildHandoverUrl('https://example.com', 'tx1', 'qr', 'abc')).toBe(
      'https://example.com/deals/tx1/complete#q=abc',
    )
    expect(buildHandoverUrl('https://example.com', 'tx1', 'link', 'abc')).toBe(
      'https://example.com/deals/tx1/complete#t=abc',
    )
  })
})

describe('parseHandoverFragment', () => {
  it('フラグメントの名前で種類を決める', () => {
    expect(parseHandoverFragment('#q=abc')).toEqual({ kind: 'qr', token: 'abc' })
    expect(parseHandoverFragment('#t=a%2Bb')).toEqual({ kind: 'link', token: 'a+b' })
  })

  it('トークンがなければ null', () => {
    expect(parseHandoverFragment('')).toBeNull()
    expect(parseHandoverFragment('#t=')).toBeNull()
    expect(parseHandoverFragment('#x=abc')).toBeNull()
  })
})

describe('parseHandoverQr', () => {
  it('この取引の受け渡しQRならトークンを返す', () => {
    const text = buildHandoverUrl('https://example.com', 'tx1', 'qr', 'abc')
    expect(parseHandoverQr(text, 'tx1')).toBe('abc')
  })

  it('別の取引・完了リンク・URL でないものは null', () => {
    expect(parseHandoverQr('https://example.com/deals/tx2/complete#q=abc', 'tx1')).toBeNull()
    expect(parseHandoverQr('https://example.com/deals/tx1/complete#t=abc', 'tx1')).toBeNull()
    expect(parseHandoverQr('9784000000000', 'tx1')).toBeNull()
  })
})
