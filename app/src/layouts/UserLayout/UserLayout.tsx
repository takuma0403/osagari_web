import type { ReactNode } from 'react'
import { Outlet } from 'react-router'
import { BottomNav } from '../../components/BottomNav/BottomNav'
import { PC_MEDIA_QUERY, useMediaQuery } from '../../lib/useMediaQuery'
import styles from './UserLayout.module.css'

type Props = {
  /** ナビ（BottomNav）を出す。デザインでナビのある画面（Main・Search・DealList・MyPage）だけで付ける */
  nav?: boolean
  /** 列を最大 1080px に広げる（一覧・商品詳細・出品・取引・学生証・売上管理。DS-1）。省くとフォーム向けの最大 560px */
  wide?: boolean
  /** 中身を直接渡す（ルートの外で枠だけ使うとき）。省くと子ルート（Outlet）を出す */
  children?: ReactNode
}

/**
 * 利用者向け画面の枠（DS-1）。390px のスマホを基準にし、広い画面では最大幅の列を中央に寄せる。
 * ナビはスマホでは下部、PC 幅（1024px 以上）では上部に置く。見た目だけでなく DOM の順も入れ替え、
 * Tab の順を見た目の順（PC ではナビ → 本文）にそろえる。地の色・ヘッダーは各画面が持つ。
 */
export function UserLayout({ nav, wide, children }: Props) {
  const pc = useMediaQuery(PC_MEDIA_QUERY)
  return (
    <div className={styles.root}>
      {nav && pc && (
        <div className={styles.topNav}>
          <BottomNav placement="top" />
        </div>
      )}
      <div className={[styles.frame, wide && styles.wide].filter(Boolean).join(' ')}>
        <main className={styles.content}>{children ?? <Outlet />}</main>
        {nav && !pc && (
          <div className={styles.bottomNav}>
            <BottomNav />
          </div>
        )}
      </div>
    </div>
  )
}
