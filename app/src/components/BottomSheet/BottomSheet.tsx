import { useId, type ReactNode } from 'react'
import { Modal } from '../Modal/Modal'
import styles from './BottomSheet.module.css'

type Props = {
  open: boolean
  /** Esc・背面の幕を押したとき（閉じるボタンは children に置く） */
  onClose: () => void
  title: string
  description?: ReactNode
  /** 見出しの上に置くもの（取引成立のチェックアイコンなど）。あると見出しを中央にそろえる */
  leading?: ReactNode
  /** 上端のつまみ。取引成立の確認のように、必ず読んでほしいシートでは false */
  handle?: boolean
  /** 見出しの右に置くもの（絞り込みの「クリア」など） */
  action?: ReactNode
  children?: ReactNode
}

/** 画面下から出るシート（DS-3）。上端角丸20px＋つまみ＋背面の幕 */
export function BottomSheet({
  open,
  onClose,
  title,
  description,
  leading,
  handle = true,
  action,
  children,
}: Props) {
  const titleId = useId()
  return (
    <Modal
      open={open}
      onClose={onClose}
      labelledBy={titleId}
      placement="bottom"
      className={[styles.sheet, !handle && styles.noHandle].filter(Boolean).join(' ')}
    >
      {handle && <div className={styles.handle} aria-hidden="true" />}
      <div className={[styles.header, leading && styles.centered].filter(Boolean).join(' ')}>
        {leading}
        <div className={styles.titleRow}>
          <h2 id={titleId} className={styles.title}>
            {title}
          </h2>
          {action}
        </div>
        {description && <p className={styles.description}>{description}</p>}
      </div>
      {children}
    </Modal>
  )
}
