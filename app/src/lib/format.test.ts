import {
  formatCountdown,
  formatCountdownHours,
  formatDate,
  formatHandoverCode,
  formatPrice,
  formatRemaining,
  formatShortDateTime,
} from './format'

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

describe('formatShortDateTime', () => {
  // 端末のタイムゾーンによらず確かめられるよう、ローカル時刻で作った日時を渡す
  it.each([
    [new Date(2026, 9, 8, 14, 2), '10/8 14:02'],
    [new Date(2026, 0, 1, 9, 0), '1/1 09:00'],
  ])('%s を %s と表示する', (date, expected) => {
    expect(formatShortDateTime(date.toISOString())).toBe(expected)
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

describe('formatRemaining', () => {
  const now = Date.parse('2026-10-08T00:00:00Z')
  const after = (ms: number) => new Date(now + ms).toISOString()
  const hour = 60 * 60 * 1000
  it.each([
    [48 * hour, '48時間'],
    [24 * hour + 59 * 60 * 1000, '24時間'],
    [hour, '1時間'],
    [hour - 1, '59分'],
    [42 * 60 * 1000, '42分'],
    [30 * 1000, '0分'],
    [-hour, '0分'],
  ])('残り %d ミリ秒を %s と表示する', (ms, expected) => {
    expect(formatRemaining(after(ms), now)).toBe(expected)
  })
})

describe('formatCountdownHours', () => {
  it.each([
    [24 * 3600, '24:00'],
    [23 * 3600 + 57 * 60 + 1, '23:58'],
    [59, '0:01'],
    [-5, '0:00'],
  ])('%d 秒は %s', (seconds, expected) => {
    expect(formatCountdownHours(seconds)).toBe(expected)
  })
})

describe('formatHandoverCode', () => {
  it('6桁を3桁ずつに区切る', () => {
    expect(formatHandoverCode('381902')).toBe('381 902')
  })
})
