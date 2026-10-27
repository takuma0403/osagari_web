import { salesEntryDate } from './salesDisplay'
import { salesEntry } from './test/fixtures'

describe('salesEntryDate', () => {
  it('出金済みなら出金した日を出す', () => {
    expect(salesEntryDate(salesEntry({ paidOutAt: '2026-10-20T03:00:00Z' }))).toBe('10/20')
  })

  it('未出金なら受け渡しが完了した日を出す', () => {
    expect(salesEntryDate(salesEntry())).toBe('10/5')
  })

  it('受け渡し待ちなら取引が成立した日を出す', () => {
    expect(salesEntryDate(salesEntry({ state: 'awaiting_handover', completedAt: null }))).toBe(
      '10/1',
    )
  })
})
