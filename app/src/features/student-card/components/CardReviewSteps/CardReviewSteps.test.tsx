import { render, screen, within } from '@testing-library/react'
import { formatShortDateTime } from '../../../../lib/format'
import type { StudentCardState } from '../../api'
import { CardReviewSteps } from './CardReviewSteps'

const submittedAt = '2026-10-08T05:02:00Z'
const reviewedAt = '2026-10-09T01:30:00Z'

function steps(card: StudentCardState) {
  render(<CardReviewSteps card={card} />)
  return within(screen.getByRole('list', { name: '審査の進み具合' })).getAllByRole('listitem')
}

describe('CardReviewSteps', () => {
  it('未登録はアップロードがいまの段階', () => {
    const [upload, review, sell] = steps({ status: 'none', submittedAt: null, reviewedAt: null })
    expect(upload).toHaveAttribute('aria-current', 'step')
    expect(review).toHaveTextContent('管理者の確認（未完了）')
    expect(sell).toHaveTextContent('出品できるようになる（未完了）')
  })

  it('審査中は提出日時と「審査中」を出す', () => {
    const [upload, review] = steps({ status: 'pending', submittedAt, reviewedAt: null })
    expect(upload).toHaveTextContent('画像をアップロード（完了）')
    expect(upload).toHaveTextContent(formatShortDateTime(submittedAt))
    expect(review).toHaveAttribute('aria-current', 'step')
    expect(review).toHaveTextContent('管理者の確認待ち')
    expect(review).toHaveTextContent('審査中')
  })

  it('差し戻しは「差し戻し」を出す', () => {
    const [, review, sell] = steps({ status: 'rejected', submittedAt, reviewedAt })
    expect(review).toHaveTextContent('差し戻し')
    expect(review).not.toHaveAttribute('aria-current')
    expect(sell).toHaveTextContent('（未完了）')
  })

  it('承認済みはすべて完了で、審査日時を出す', () => {
    const items = steps({ status: 'approved', submittedAt, reviewedAt })
    items.forEach((item) => expect(item).toHaveTextContent('（完了）'))
    expect(items[1]).toHaveTextContent(formatShortDateTime(reviewedAt))
  })
})
