import { estimateProceeds, formatFeeRate } from './fee'

describe('estimateProceeds', () => {
  it.each([
    [1500, 500, 75, 1425],
    [1550, 500, 77, 1473], // 1円未満は切り捨て
    [1000, 525, 52, 948],
    [1500, 0, 0, 1500],
    [100000, 999, 9990, 90010],
  ])('¥%i・%ibp は手数料 ¥%i、受け取り ¥%i', (price, bp, fee, proceeds) => {
    expect(estimateProceeds(price, bp)).toEqual({ fee, proceeds })
  })
})

describe('formatFeeRate', () => {
  it.each([
    [500, '5%'],
    [525, '5.25%'],
    [1050, '10.5%'],
    [0, '0%'],
  ])('%ibp を %s にする', (bp, text) => {
    expect(formatFeeRate(bp)).toBe(text)
  })
})
