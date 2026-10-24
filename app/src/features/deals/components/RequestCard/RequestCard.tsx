import { Avatar } from '../../../../components/Avatar/Avatar'
import { Button } from '../../../../components/Button/Button'
import { formatRemaining, formatShortDateTime } from '../../../../lib/format'
import type { PurchaseRequest } from '../../api'
import styles from './RequestCard.module.css'

type Props = {
  request: PurchaseRequest
  /** 期限までの残りを計る基準の時刻（ミリ秒） */
  now: number
  onApprove: () => void
  onReject: () => void
  /** 承認・却下の送信中など、押せないとき */
  disabled?: boolean
}

/**
 * 出品者に見せる購入希望1件（DEAL-1、DealRequests）。購入者は学部・入学年度と取引完了の回数だけを出し、学籍番号は出さない。
 * 承認期限（購入希望から48時間）までの残りを出す。
 */
export function RequestCard({ request, now, onApprove, onReject, disabled }: Props) {
  const { buyer } = request
  return (
    <section className={styles.card} aria-label={`${buyer.departmentName ?? '学部不明'}の購入希望`}>
      <div className={styles.buyer}>
        <Avatar departmentName={buyer.departmentName} />
        <div className={styles.buyerText}>
          <span className={styles.name}>
            {buyer.departmentName ?? '学部不明'} · {buyer.admissionYear}年度入学
          </span>
          <span className={styles.meta}>
            {formatShortDateTime(request.requestedAt)}に希望 · 取引完了 {buyer.completedDealCount}回
          </span>
        </div>
      </div>
      <div className={styles.footer}>
        <span className={styles.deadline}>
          期限まで残り {formatRemaining(request.approvalDeadline, now)}
        </span>
        <Button variant="text" size="sm" onClick={onReject} disabled={disabled}>
          見送る
        </Button>
        <Button size="sm" onClick={onApprove} disabled={disabled}>
          この人に決める
        </Button>
      </div>
    </section>
  )
}
