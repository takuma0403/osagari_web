import { normalizeStudentId } from './studentId'

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
