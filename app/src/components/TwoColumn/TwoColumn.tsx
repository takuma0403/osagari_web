import type { ReactNode } from 'react'
import styles from './TwoColumn.module.css'

type Props = {
  /** 左（PC 幅）に置くもの。スマホでは end より先に出る */
  start: ReactNode
  /** 右（PC 幅）に置くもの */
  end?: ReactNode
  /** 左右の幅。even は半分ずつ、narrowStart は左を 360px（出品の写真欄） */
  variant?: 'even' | 'narrowStart'
}

/**
 * PC 幅（1024px 以上）で中身を左右2カラムに分ける枠（DS-1。キャンバスにない形を既存の部品で補う）。
 * スマホでは枠を持たず（display: contents）、親の縦並びにそのまま入るので見た目は変わらない。
 * カラムの中の間隔は親の gap を引き継ぐ。
 */
export function TwoColumn({ start, end, variant = 'even' }: Props) {
  return (
    <div className={[styles.columns, styles[variant]].join(' ')}>
      <div className={styles.column}>{start}</div>
      {end && <div className={styles.column}>{end}</div>}
    </div>
  )
}
