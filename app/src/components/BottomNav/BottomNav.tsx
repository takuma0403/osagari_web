import { NavLink } from 'react-router'
import { AppLogo } from '../AppLogo/AppLogo'
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

type Props = {
  /**
   * 置く場所。bottom はスマホの下部5タブ（デザインのとおり）、top は PC 幅（1024px 以上）の上部の横ナビ。
   * top はキャンバスにないため、ロゴとアプリ名を左に置き、同じ5項目を横に並べる（DS-1）
   */
  placement?: 'bottom' | 'top'
}

/**
 * 利用者向け画面のナビ（DS-3）。5タブで、いまの画面のタブを primary + 太字にし aria-current="page" を付ける。
 * UserLayout がナビのある画面で描き、画面幅に応じて placement を切り替える。
 */
export function BottomNav({ placement = 'bottom' }: Props) {
  const top = placement === 'top'
  return (
    <nav aria-label="メインメニュー" className={top ? styles.top : styles.nav}>
      {top && (
        <div className={styles.brand}>
          <AppLogo size={32} />
          <span>佐賀大お下がりフリマ</span>
        </div>
      )}
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
