import type { ReactNode } from 'react'
import { Outlet } from 'react-router'
import styles from './MobileLayout.module.css'

type Props = {
  /** 画面下に固定するナビ（BottomNav）。タブのある画面だけで渡す */
  bottomNav?: ReactNode
  /** 中身を直接渡す（ルートの外で枠だけ使うとき）。省くと子ルート（Outlet）を出す */
  children?: ReactNode
}

/**
 * 利用者向け画面の枠（DS-1）。390px のスマホを基準にし、PC では最大幅で中央に寄せる。
 * 地の色・ヘッダーは各画面が持つ。
 */
export function MobileLayout({ bottomNav, children }: Props) {
  return (
    <div className={styles.frame}>
      <main className={styles.content}>{children ?? <Outlet />}</main>
      {bottomNav && <div className={styles.bottomNav}>{bottomNav}</div>}
    </div>
  )
}
