import type { AriaRole, ReactNode } from 'react'
import { Icon } from '../Icon/Icon'
import type { IconName } from '../Icon/iconPaths'
import styles from './InfoBox.module.css'

type Props = {
  /** info：案内／warning：注意／danger：ロック中などの警告／neutral：補足 */
  tone?: 'info' | 'warning' | 'danger' | 'neutral'
  /** 文の前に置くアイコン */
  icon?: IconName
  /** エラーを伝えるときは alert、状況の更新は status（DS-1） */
  role?: AriaRole
  className?: string
  children: ReactNode
}

/** 角丸12pxの案内・注意・警告の枠（DS-3） */
export function InfoBox({ tone = 'info', icon, role, className, children }: Props) {
  return (
    <div className={[styles.box, styles[tone], className].filter(Boolean).join(' ')} role={role}>
      {icon && <Icon name={icon} size={20} className={styles.icon} />}
      <div className={styles.body}>{children}</div>
    </div>
  )
}
