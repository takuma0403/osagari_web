import { Link, type To } from 'react-router'
import { formatPrice } from '../../lib/format'
import { Badge } from '../Badge/Badge'
import { ItemPhoto, type PhotoTint } from '../ItemPhoto/ItemPhoto'
import styles from './ItemRow.module.css'

type Props = {
  /** 商品詳細への行き先 */
  to: To
  title: string
  /** タイトルの下の補足（教科書なら授業名・学部など） */
  subtitle?: string
  /** 価格（円の整数） */
  price: number
  /** 価格の右に出す状態（例 書き込みなし） */
  note?: string
  /** 写真がないときの地の色 */
  tint: PhotoTint
  imageUrl?: string
}

/** 検索結果などの横長の行（DS-3）。84pxのサムネイル＋タイトル＋補足＋価格＋状態 */
export function ItemRow({ to, title, subtitle, price, note, tint, imageUrl }: Props) {
  return (
    <Link to={to} className={styles.row}>
      <ItemPhoto imageUrl={imageUrl} tint={tint} className={styles.photo} />
      <div className={styles.body}>
        <div className={styles.title}>{title}</div>
        {subtitle && <div className={styles.sub}>{subtitle}</div>}
        <div className={styles.footer}>
          <span className={styles.price}>{formatPrice(price)}</span>
          {note && <Badge tone="neutral">{note}</Badge>}
        </div>
      </div>
    </Link>
  )
}
