import { formatCountdown, formatDate, formatPrice } from './format'

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

describe('formatDate', () => {
  it.each([
    ['2030-03-31', '2030年3月31日'],
    ['2032-12-01', '2032年12月1日'],
  ])('%s を %s と表示する', (date, expected) => {
    expect(formatDate(date)).toBe(expected)
  })
})

describe('formatCountdown', () => {
  it.each([
    [600, '10:00'],
    [581, '9:41'],
    [42, '0:42'],
    [0.2, '0:01'],
    [0, '0:00'],
    [-5, '0:00'],
  ])('%d 秒を %s と表示する', (seconds, expected) => {
    expect(formatCountdown(seconds)).toBe(expected)
  })
})
