import { normalizeStudentId, toUniversityEmail } from './studentId'

describe('normalizeStudentId', () => {
  it.each([
    ['26238001', '26238001'],
    ['２６２３８００１', '26238001'],
    ['26238001@edu.cc.saga-u.ac.jp', '26238001'],
    ['2623-8001', '26238001'],
    ['262380019', '26238001'],
    ['abc', ''],
  ])('%s → %s', (input, expected) => {
    expect(normalizeStudentId(input)).toBe(expected)
  })
})

describe('toUniversityEmail', () => {
  it('学籍番号に大学のドメインを続ける', () => {
    expect(toUniversityEmail('26238001')).toBe('26238001@edu.cc.saga-u.ac.jp')
  })
})
