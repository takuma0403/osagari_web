import { formatPrice } from './format'

describe('formatPrice', () => {
  it.each([
    [0, '¥0'],
    [800, '¥800'],
    [1500, '¥1,500'],
    [15000, '¥15,000'],
    [1234567, '¥1,234,567'],
  ])('%d を %s と表示する', (amount, expected) => {
    expect(formatPrice(amount)).toBe(expected)
  })
})
