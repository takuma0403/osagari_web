import type { ReactNode } from 'react'
import styles from './Badge.module.css'

export type BadgeTone = 'primary' | 'warning' | 'danger' | 'neutral' | 'surface'

type Props = {
  /** primary：取引中など／warning：購入希望 n件・モック表示／danger：停止など／neutral：終了／surface：写真の上 */
  tone?: BadgeTone
  /** sm 11px（一覧・ヘッダー）／md 12px（詳細） */
  size?: 'sm' | 'md'
  className?: string
  children: ReactNode
}

/** 状態やカテゴリを示す小さなラベル（DS-3） */
export function Badge({ tone = 'primary', size = 'sm', className, children }: Props) {
  return (
    <span
      className={[styles.badge, styles[tone], styles[size], className].filter(Boolean).join(' ')}
    >
      {children}
    </span>
  )
}
