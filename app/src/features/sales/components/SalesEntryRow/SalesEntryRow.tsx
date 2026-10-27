import { Link } from 'react-router'
import { formatFeeRate } from '../../../../lib/fee'
import { formatPrice } from '../../../../lib/format'
import type { SalesEntry } from '../../api'
import { salesEntryDate, salesStateLabels } from '../../salesDisplay'
import styles from './SalesEntryRow.module.css'

type Props = {
  entry: SalesEntry
}

/**
 * 売上の明細の1行（PAY-5、Sales）。商品名と売上、日付・状態、金額と手数料（料率）。押すと商品詳細を開く（ADR 0032）。
 * 料率・手数料・売上は取引成立時に記録した値をそのまま出す（PAY-2、ADR 0003）。
 */
export function SalesEntryRow({ entry }: Props) {
  return (
    <Link to={`/items/${entry.item.shareId}`} className={styles.row}>
      <div className={styles.line}>
        <span className={styles.title}>{entry.item.title}</span>
        <span className={styles.proceeds}>{formatPrice(entry.sellerProceeds)}</span>
      </div>
      <div className={styles.detail}>
        <span>
          {salesEntryDate(entry)} · {salesStateLabels[entry.state]}
        </span>
        <span className={styles.breakdown}>
          {formatPrice(entry.price)} − 手数料（{formatFeeRate(entry.feeRateBasisPoints)}）{' '}
          {formatPrice(entry.fee)}
        </span>
      </div>
    </Link>
  )
}
