import type { ReactNode } from 'react'
import styles from './ItemPhoto.module.css'

/**
 * 写真がないときの地の色（DS-2 の --color-cat-*）。
 * 要件のカテゴリ（ItemCategory）との対応は商品一覧の画面で決める（glossary の注記）。
 */
export type PhotoTint = 'textbook' | 'appliance' | 'furniture' | 'game'

type Props = {
  imageUrl?: string
  tint: PhotoTint
  className?: string
  /** 写真の上に重ねるもの（カテゴリのバッジなど） */
  children?: ReactNode
}

/**
 * 商品の写真の枠。ItemCard・ItemRow で使う。商品名を隣に表示するので、写真は装飾（alt なし）として扱う。
 */
export function ItemPhoto({ imageUrl, tint, className, children }: Props) {
  return (
    <div className={[styles.photo, styles[tint], className].filter(Boolean).join(' ')}>
      {imageUrl && <img className={styles.image} src={imageUrl} alt="" loading="lazy" />}
      {children}
    </div>
  )
}
