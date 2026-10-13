import { NavLink, Outlet } from 'react-router'
import { AppLogo } from '../../components/AppLogo/AppLogo'
import styles from './AdminLayout.module.css'

// 管理メニュー（AdminCard のサイドバー）。件数バッジは各画面の実装（ADMIN）で足す
const menu = [
  { to: '/admin/student-cards', label: '学生証審査' },
  { to: '/admin/users', label: 'ユーザー' },
  { to: '/admin/items', label: '出品・取引' },
  { to: '/admin/settings', label: '設定（手数料率）' },
] as const

/** 管理画面の枠（DS-1）。PC 幅の流体レイアウトで、狭い画面ではサイドバーが上に回り込む */
export function AdminLayout() {
  return (
    <div className={styles.frame}>
      <nav aria-label="管理メニュー" className={styles.sidebar}>
        <div className={styles.brand}>
          <AppLogo size={30} />
          <span>管理画面</span>
        </div>
        {menu.map((m) => (
          <NavLink
            key={m.to}
            to={m.to}
            className={({ isActive }) =>
              isActive ? `${styles.link} ${styles.active}` : styles.link
            }
          >
            {m.label}
          </NavLink>
        ))}
      </nav>
      <main className={styles.content}>
        <Outlet />
      </main>
    </div>
  )
}
