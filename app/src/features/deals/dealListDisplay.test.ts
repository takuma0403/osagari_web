import { dealRowDisplay } from './dealListDisplay'
import { transactionListEntry } from './test/fixtures'

const now = Date.parse('2026-10-08T03:40:00Z')

describe('dealRowDisplay', () => {
  it('購入側の購入希望は承認待ちと期限までの残りを出し、取引詳細へ進む', () => {
    expect(dealRowDisplay(transactionListEntry(), 'buyer', now)).toEqual({
      to: '/deals/tx1',
      status: '承認待ち',
      tone: 'warning',
      note: 'あと24時間で期限切れ',
    })
  })

  it.each([
    ['期限を過ぎた', '2026-10-08T03:00:00Z'],
    ['残り1分未満の', '2026-10-08T03:40:30Z'],
  ])('%s購入希望は、期限切れになるまで「まもなく期限切れ」と出す', (_, approvalDeadline) => {
    const entry = transactionListEntry({ approvalDeadline })
    expect(dealRowDisplay(entry, 'buyer', now).note).toBe('まもなく期限切れ')
  })

  it('残りちょうど1分なら「あと1分で期限切れ」と出す', () => {
    const entry = transactionListEntry({ approvalDeadline: '2026-10-08T03:41:00Z' })
    expect(dealRowDisplay(entry, 'buyer', now).note).toBe('あと1分で期限切れ')
  })

  it('出品側の商品ごとの購入希望は件数を出し、購入希望の一覧へ進む', () => {
    const entry = transactionListEntry({
      transactionId: null,
      itemId: 'i1',
      requestCount: 3,
      approvalDeadline: null,
    })
    expect(dealRowDisplay(entry, 'seller', now)).toEqual({
      to: '/sell/i1/requests',
      status: '購入希望 3件',
      tone: 'warning',
      note: '相手を選んでください',
    })
  })

  it.each([
    ['buyer', '受け渡しの日時をTeamsで相談'],
    ['seller', '受け渡しでQRを表示'],
  ] as const)('取引中（%s）は「%s」', (role, note) => {
    const entry = transactionListEntry({ status: 'in_deal', approvalDeadline: null })
    expect(dealRowDisplay(entry, role, now)).toEqual({
      to: '/deals/tx1',
      status: '取引中',
      tone: 'primary',
      note,
    })
  })

  it('完了は、購入側なら受け渡しの日、出品側なら売上に計上済みと出す', () => {
    const entry = transactionListEntry({
      status: 'completed',
      approvalDeadline: null,
      // 端末のタイムゾーンによらず確かめられるよう、ローカル時刻で作る
      completedAt: new Date(2026, 8, 28, 12, 0).toISOString(),
    })
    expect(dealRowDisplay(entry, 'buyer', now)).toMatchObject({
      status: '完了',
      tone: 'neutral',
      note: '9/28 受け渡し',
    })
    expect(dealRowDisplay(entry, 'seller', now).note).toBe('売上に計上済み')
  })

  it.each([
    ['cancelled', 'キャンセル', '双方の同意で取り消し'],
    ['expired', '期限切れ', '枠を解放しました'],
    ['rejected', '却下', '枠を解放しました'],
  ] as const)('%s は「%s」「%s」', (status, label, note) => {
    const entry = transactionListEntry({ status, approvalDeadline: null })
    expect(dealRowDisplay(entry, 'buyer', now)).toEqual({
      to: '/deals/tx1',
      status: label,
      tone: 'neutral',
      note,
    })
  })
})
