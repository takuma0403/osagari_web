import { campusOptions, categoryOptions, categoryTint, isOptionValue, labelOf } from './itemOptions'

describe('itemOptions', () => {
  it('値から表示名を引く', () => {
    expect(labelOf(categoryOptions, 'furniture_appliance')).toBe('家具・家電')
    expect(labelOf(campusOptions, 'nabeshima')).toBe('鍋島')
  })

  it('選択肢にない文字列は値として認めない', () => {
    expect(isOptionValue(categoryOptions, 'textbook')).toBe(true)
    expect(isOptionValue(categoryOptions, 'game')).toBe(false)
    expect(isOptionValue(categoryOptions, '')).toBe(false)
  })

  it('家具・家電の写真の地は家具の色にする', () => {
    expect(categoryTint).toEqual({
      textbook: 'textbook',
      furniture_appliance: 'furniture',
      game_other: 'game',
    })
  })
})
