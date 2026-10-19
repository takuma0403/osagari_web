import { useCallback, useMemo, useState } from 'react'
import { NavLink, Outlet } from 'react-router'
import { AppLogo } from '../../components/AppLogo/AppLogo'
import type { AdminLayoutContext, AdminMenuBadgeKey } from './adminLayoutContext'
import styles from './AdminLayout.module.css'

// 管理メニュー（AdminCard のサイドバー）。badge は件数を出す行で、件数は各画面が useAdminLayout で設定する
const menu: { to: string; label: string; badge?: { key: AdminMenuBadgeKey; label: string } }[] = [
  {
    to: '/admin/student-cards',
    label: '学生証審査',
    badge: { key: 'studentCards', label: '審査待ち' },
  },
  { to: '/admin/users', label: 'ユーザー' },
  { to: '/admin/items', label: '出品・取引' },
  { to: '/admin/settings', label: '設定（手数料率）' },
]

/** 管理画面の枠（DS-1）。PC 幅の流体レイアウトで、狭い画面ではサイドバーが上に回り込む */
export function AdminLayout() {
  const [badges, setBadges] = useState<Partial<Record<AdminMenuBadgeKey, number>>>({})
  const setMenuBadge = useCallback<AdminLayoutContext['setMenuBadge']>((key, count) => {
    setBadges((prev) => (prev[key] === count ? prev : { ...prev, [key]: count }))
  }, [])
  const context = useMemo<AdminLayoutContext>(() => ({ setMenuBadge }), [setMenuBadge])

  return (
    <div className={styles.frame}>
      <nav aria-label="管理メニュー" className={styles.sidebar}>
        <div className={styles.brand}>
          <AppLogo size={30} />
          <span>管理画面</span>
        </div>
        {menu.map((m) => {
          const count = m.badge && badges[m.badge.key]
          return (
            <NavLink
              key={m.to}
              to={m.to}
              className={({ isActive }) =>
                isActive ? `${styles.link} ${styles.active}` : styles.link
              }
            >
              {m.label}
              {m.badge && count ? (
                <>
                  {/* 読み上げで項目名と件数がつながらないよう、空白を挟む（見た目は flex で詰まる） */}{' '}
                  <span className={styles.badge}>
                    <span className="visually-hidden">{m.badge.label}</span>
                    {count}
                    <span className="visually-hidden">件</span>
                  </span>
                </>
              ) : null}
            </NavLink>
          )
        })}
      </nav>
      <main className={styles.content}>
        <Outlet context={context} />
      </main>
    </div>
  )
}
