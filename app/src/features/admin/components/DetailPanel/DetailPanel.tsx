import type { ReactNode } from 'react'
import styles from './DetailPanel.module.css'

type Props = {
  /** パネルの名前（section の aria-label。例 ユーザーの詳細） */
  label: string
  children: ReactNode
}

/** 管理画面の一覧の右に置く、選んだ1件の詳細と操作の枠（AdminUsers・AdminDeals） */
export function DetailPanel({ label, children }: Props) {
  return (
    <section aria-label={label} className={styles.panel}>
      {children}
    </section>
  )
}

type SectionProps = {
  /** 操作の下に出す補足 */
  note?: string
  /** 操作の失敗（role="alert"） */
  error?: string | null
  /** ボタンを横に並べる（既定は縦に積み、ボタンは左寄せ） */
  row?: boolean
  children?: ReactNode
}

/** 詳細パネルの中の、上に区切り線を引いた操作の欄 */
export function PanelSection({ note, error, row, children }: SectionProps) {
  return (
    <div className={styles.section}>
      {children && <div className={row ? styles.row : styles.stack}>{children}</div>}
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
      {note && <span className={styles.note}>{note}</span>}
    </div>
  )
}
