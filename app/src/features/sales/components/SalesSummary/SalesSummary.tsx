import { Button } from '../../../../components/Button/Button'
import { formatPrice } from '../../../../lib/format'
import styles from './SalesSummary.module.css'

type Props = {
  /** 出金できる売上（円） */
  availableAmount: number
  /** 受け渡し待ちの売上（円） */
  awaitingHandoverAmount: number
  /** 出金済みの売上（円） */
  paidOutAmount: number
  /** 「出金を申請する」 */
  onPayout: () => void
  /** 申請中など、押せないとき。出金できる売上が 0 円のときも押せない */
  payoutDisabled?: boolean
}

/**
 * 売上管理（Sales）の上の集計（PAY-5）。出金できる売上・受け渡し待ち・出金済みと、出金の申請ボタン。
 * 金額はサーバーの集計をそのまま出す。受け取り先の登録はフェーズ1では作らないので「未登録」の固定表示（ADR 0029）。
 */
export function SalesSummary({
  availableAmount,
  awaitingHandoverAmount,
  paidOutAmount,
  onPayout,
  payoutDisabled,
}: Props) {
  return (
    <section aria-label="売上の集計" className={styles.summary}>
      <dl className={styles.amounts}>
        <div className={styles.available}>
          <dt>出金できる売上</dt>
          <dd className={styles.availableAmount}>{formatPrice(availableAmount)}</dd>
        </div>
        <div className={styles.sub}>
          <dt>受け渡し待ち</dt>
          <dd className={styles.subAmount}>{formatPrice(awaitingHandoverAmount)}</dd>
        </div>
        <div className={styles.sub}>
          <dt>出金済み</dt>
          <dd className={styles.subAmount}>{formatPrice(paidOutAmount)}</dd>
        </div>
      </dl>
      <dl className={styles.destination}>
        <dt>受け取り先</dt>
        <dd>[銀行口座 未登録]</dd>
      </dl>
      <Button
        size="md"
        fullWidth
        onClick={onPayout}
        disabled={payoutDisabled || availableAmount <= 0}
      >
        出金を申請する
      </Button>
    </section>
  )
}
