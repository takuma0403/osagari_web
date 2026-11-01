import { isBookBarcode, isValidIsbn, normalizeIsbn } from './isbn'

describe('normalizeIsbn', () => {
  it('ハイフンと空白を除き、末尾の x を大文字にする', () => {
    expect(normalizeIsbn('978-4-00-000000-0')).toBe('9784000000000')
    expect(normalizeIsbn(' 4-00-000000-x ')).toBe('400000000X')
  })
})

describe('isValidIsbn', () => {
  it.each([
    ['9784000000000', true],
    ['9784000000002', false],
    ['4000000004', true],
    ['4000000001', false],
    ['080442957X', true],
    ['97840000000', false],
    ['978400000000A', false],
  ])('%s は %s', (isbn, valid) => {
    expect(isValidIsbn(isbn)).toBe(valid)
  })
})

describe('isBookBarcode', () => {
  it('978・979 で始まる正しい EAN-13 だけを書籍とみなす', () => {
    expect(isBookBarcode('9784000000000')).toBe(true)
    expect(isBookBarcode('4901234567894')).toBe(false)
    expect(isBookBarcode('9784000000002')).toBe(false)
  })
})
