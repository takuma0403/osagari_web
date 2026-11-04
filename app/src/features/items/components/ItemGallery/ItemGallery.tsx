import { useState, type ReactNode, type UIEvent } from 'react'
import { ItemPhoto, type PhotoTint } from '../../../../components/ItemPhoto/ItemPhoto'
import styles from './ItemGallery.module.css'

type Props = {
  /** 写真の URL（表示順）。空なら写真の地の色だけを出す */
  imageUrls: string[]
  tint: PhotoTint
  /** 画面側で写真の枠の大きさを変えるとき（商品詳細の PC 幅など） */
  className?: string
  /** 写真の上に重ねるもの（戻るボタンなど） */
  children?: ReactNode
}

/**
 * 商品ページの写真（BROWSE、Detail）。横にスワイプして切り替え、下のドットで何枚目かを示す。
 * 写真は商品の状態を伝える内容なので、何枚目かを alt に入れる。
 */
export function ItemGallery({ imageUrls, tint, className, children }: Props) {
  const [current, setCurrent] = useState(0)
  const total = imageUrls.length

  const onScroll = (e: UIEvent<HTMLDivElement>) => {
    const { scrollLeft, clientWidth } = e.currentTarget
    if (clientWidth > 0) setCurrent(Math.round(scrollLeft / clientWidth))
  }

  return (
    <ItemPhoto tint={tint} className={[styles.gallery, className].filter(Boolean).join(' ')}>
      {total > 0 && (
        // キーボードでも左右に送れるよう、スクロールする枠にフォーカスできるようにする
        <div
          className={styles.track}
          role="region"
          aria-label="商品の写真"
          tabIndex={0}
          onScroll={onScroll}
        >
          {imageUrls.map((url, i) => (
            <img
              key={url}
              className={styles.slide}
              src={url}
              alt={`商品の写真 ${i + 1} / ${total}`}
            />
          ))}
        </div>
      )}
      {children}
      {total > 1 && (
        <div className={styles.dots} aria-hidden="true">
          {imageUrls.map((url, i) => (
            <span
              key={url}
              className={[styles.dot, i === current && styles.active].filter(Boolean).join(' ')}
            />
          ))}
        </div>
      )}
    </ItemPhoto>
  )
}
