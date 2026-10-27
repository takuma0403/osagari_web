import styles from './Avatar.module.css'

type Props = {
  /** 学部・学科の名前。頭の1文字を出し、不明なら「?」 */
  departmentName: string | null
  /** md 40px（商品詳細・購入希望）／lg 48px（マイページ） */
  size?: 'md' | 'lg'
  /** primary の面の上に置くときは白地＋primary の文字にする（マイページのプロフィール） */
  inverse?: boolean
}

/**
 * 利用者の丸いアイコン（学部の頭文字）。商品詳細の出品者、購入希望の購入者、マイページの本人に使う。
 * 名前は隣に文字で出すので、アイコン自体は読み上げない。学籍番号は出さない（DEAL-1）。
 */
export function Avatar({ departmentName, size = 'md', inverse }: Props) {
  return (
    <span
      className={[styles.avatar, styles[size], inverse && styles.inverse].filter(Boolean).join(' ')}
      aria-hidden="true"
    >
      {departmentName?.charAt(0) ?? '?'}
    </span>
  )
}
