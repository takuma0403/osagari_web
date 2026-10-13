import type { ReactNode } from 'react'
import styles from './PlaceholderPage.module.css'

type Props = {
  title: string
  children?: ReactNode
}

/** 画面の実装までルートを埋めておく仮ページ。画面ごとの issue で本物の画面に置き換える */
export function PlaceholderPage({ title, children }: Props) {
  return (
    <section className={styles.page}>
      <h1 className={styles.title}>{title}</h1>
      <p className={styles.note}>この画面は準備中です。</p>
      {children}
    </section>
  )
}
