import { NavLink } from 'react-router'
import { Icon } from '../Icon/Icon'
import type { IconName } from '../Icon/iconPaths'
import styles from './BottomNav.module.css'

const tabs: { to: string; label: string; icon: IconName }[] = [
  { to: '/', label: 'ホーム', icon: 'home' },
  { to: '/search', label: '検索', icon: 'search' },
  { to: '/sell', label: '出品', icon: 'sell' },
  { to: '/deals', label: '取引', icon: 'deal' },
  { to: '/mypage', label: 'マイページ', icon: 'user' },
]

/**
 * 利用者向け画面の下部ナビ（DS-3）。5タブで、いまの画面のタブを primary + 太字にし aria-current="page" を付ける。
 * routes.tsx でナビのある画面の MobileLayout に渡す。
 */
export function BottomNav() {
  return (
    <nav aria-label="メインメニュー" className={styles.nav}>
      {tabs.map(({ to, label, icon }) => (
        <NavLink
          key={to}
          to={to}
          // ホームは子のパス（/search など）で選択中にしない
          end={to === '/'}
          className={({ isActive }) =>
            [styles.link, isActive && styles.active].filter(Boolean).join(' ')
          }
        >
          <Icon name={icon} />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
