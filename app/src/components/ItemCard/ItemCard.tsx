import { Link, type To } from 'react-router'
import { formatPrice } from '../../lib/format'
import { Badge } from '../Badge/Badge'
import { ItemPhoto, type PhotoTint } from '../ItemPhoto/ItemPhoto'
import styles from './ItemCard.module.css'

type Props = {
  /** 商品詳細への行き先 */
  to: To
  title: string
  /** 価格（円の整数） */
  price: number
  /** 写真の左上に出すカテゴリ名 */
  categoryLabel: string
  /** 写真がないときの地の色 */
  tint: PhotoTint
  /** 受け渡し希望キャンパス（例 本庄） */
  campus: string
  /** 状態（例 目立った傷なし） */
  condition: string
  imageUrl?: string
}

/** 商品一覧のカード（DS-3）。正方形の写真＋カテゴリ＋タイトル2行＋価格＋キャンパス・状態。並べ方（2列グリッド）は利用側で決める */
export function ItemCard({
  to,
  title,
  price,
  categoryLabel,
  tint,
  campus,
  condition,
  imageUrl,
}: Props) {
  return (
    <Link to={to} className={styles.card}>
      <ItemPhoto imageUrl={imageUrl} tint={tint} className={styles.photo}>
        <Badge tone="surface" className={styles.category}>
          {categoryLabel}
        </Badge>
      </ItemPhoto>
      <div className={styles.body}>
        <div className={styles.title}>{title}</div>
        <div className={styles.price}>{formatPrice(price)}</div>
        <div className={styles.meta}>
          {campus}キャンパス · {condition}
        </div>
      </div>
    </Link>
  )
}
