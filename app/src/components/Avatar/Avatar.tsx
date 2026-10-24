import styles from './Avatar.module.css'

type Props = {
  /** 学部・学科の名前。頭の1文字を出し、不明なら「?」 */
  departmentName: string | null
}

/**
 * 利用者の丸いアイコン（40px、学部の頭文字）。商品詳細の出品者、購入希望の購入者に使う。
 * 名前は隣に文字で出すので、アイコン自体は読み上げない。学籍番号は出さない（DEAL-1）。
 */
export function Avatar({ departmentName }: Props) {
  return (
    <span className={styles.avatar} aria-hidden="true">
      {departmentName?.charAt(0) ?? '?'}
    </span>
  )
}
