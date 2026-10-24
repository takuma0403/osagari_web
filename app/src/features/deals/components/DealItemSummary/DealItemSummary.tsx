import type { ReactNode } from 'react'
import { ItemPhoto, type PhotoTint } from '../../../../components/ItemPhoto/ItemPhoto'
import { formatPrice } from '../../../../lib/format'
import styles from './DealItemSummary.module.css'

type Props = {
  title: string
  /** 円の整数 */
  price: number
  imageUrl?: string
  tint: PhotoTint
  /** 商品名の下に添える補足（受け渡しのキャンパスなど） */
  note?: string
  /** 右端に置くもの（「あなたが出品」のバッジなど） */
  aside?: ReactNode
  /** lg：購入希望の入力（写真64px）／sm：取引詳細の見出し（写真52px） */
  size?: 'lg' | 'sm'
}

/** 取引の画面の上に出す商品の要約（Request・DealRequests）。写真＋商品名＋価格 */
export function DealItemSummary({ title, price, imageUrl, tint, note, aside, size = 'lg' }: Props) {
  return (
    <div className={[styles.summary, styles[size]].join(' ')}>
      <ItemPhoto imageUrl={imageUrl} tint={tint} className={styles.photo} />
      <div className={styles.text}>
        <span className={styles.title}>{title}</span>
        {note && <span className={styles.note}>{note}</span>}
        <span className={styles.price}>{formatPrice(price)}</span>
      </div>
      {aside}
    </div>
  )
}
