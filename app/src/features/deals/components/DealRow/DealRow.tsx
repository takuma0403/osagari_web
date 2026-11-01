import { Link } from 'react-router'
import { Badge, type BadgeTone } from '../../../../components/Badge/Badge'
import { ItemPhoto } from '../../../../components/ItemPhoto/ItemPhoto'
import { formatPrice } from '../../../../lib/format'
import { categoryTint } from '../../../../lib/itemOptions'
import type { TransactionListItem } from '../../api'
import styles from './DealRow.module.css'

type Props = {
  /** 取引詳細か購入希望の一覧への行き先 */
  to: string
  item: TransactionListItem
  /** 取引の金額（円の整数） */
  price: number
  /** 状態のバッジ */
  status: string
  tone: BadgeTone
  /** バッジの右の補足 */
  note: string
}

/** 取引一覧（DealList）の1行。72pxの写真＋商品名＋金額＋状態のバッジと補足 */
export function DealRow({ to, item, price, status, tone, note }: Props) {
  return (
    <Link to={to} className={styles.row}>
      <ItemPhoto
        imageUrl={item.thumbnail?.url}
        tint={categoryTint[item.category]}
        className={styles.photo}
      />
      <div className={styles.body}>
        <div className={styles.title}>{item.title}</div>
        <div className={styles.price}>{formatPrice(price)}</div>
        <div className={styles.status}>
          <Badge tone={tone}>{status}</Badge>
          <span className={styles.note}>{note}</span>
        </div>
      </div>
    </Link>
  )
}
