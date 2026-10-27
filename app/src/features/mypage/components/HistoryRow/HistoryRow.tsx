import { Link } from 'react-router'
import { Badge, type BadgeTone } from '../../../../components/Badge/Badge'
import { ItemPhoto, type PhotoTint } from '../../../../components/ItemPhoto/ItemPhoto'
import { formatPrice } from '../../../../lib/format'
import styles from './HistoryRow.module.css'

type Props = {
  to: string
  title: string
  /** 価格（円の整数）。下書きで未入力なら null */
  price: number | null
  /** 右のバッジ */
  status: string
  tone: BadgeTone
  /** 写真がないときの地の色 */
  tint: PhotoTint
  imageUrl?: string
}

/** マイページ（MyPage）の出品・購入の1行。52pxの写真＋商品名＋価格＋右の状態のバッジ */
export function HistoryRow({ to, title, price, status, tone, tint, imageUrl }: Props) {
  return (
    <Link to={to} className={styles.row}>
      <ItemPhoto imageUrl={imageUrl} tint={tint} className={styles.photo} />
      <div className={styles.body}>
        <span className={styles.title}>{title}</span>
        {price == null ? (
          <span className={styles.noPrice}>価格未設定</span>
        ) : (
          <span className={styles.price}>{formatPrice(price)}</span>
        )}
      </div>
      <Badge tone={tone}>{status}</Badge>
    </Link>
  )
}
