import { useId, type ReactNode } from 'react'
import { Button } from '../Button/Button'
import { Modal } from '../Modal/Modal'
import styles from './ConfirmDialog.module.css'

type Props = {
  open: boolean
  title: string
  /** 確定すると何が起きるかの説明（金額・取り消せないことなど） */
  children?: ReactNode
  confirmLabel: string
  onConfirm: () => void
  /** 省くと取り消しボタンを出さない */
  cancelLabel?: string
  /** 取り消しボタン・Esc・背面の幕を押したとき。省くと Esc・幕では閉じず、確定ボタンでだけ閉じる */
  onCancel?: () => void
  /** 送信中など、確定ボタンを押せないとき */
  confirmDisabled?: boolean
  /** カメラ画面の上では dark */
  overlay?: 'default' | 'dark'
}

/**
 * 中央の確認モーダル（DS-3）。角丸18px、主要ボタン＋テキストボタン。
 * 受け渡し完了のように取り消せない操作の確認に使う（DEAL-3）。
 * 取り消しを省くと、確定ボタンだけのお知らせになる（試験運用の初回ダイアログ。OVERVIEW）。
 */
export function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel,
  onConfirm,
  cancelLabel,
  onCancel,
  confirmDisabled,
  overlay,
}: Props) {
  const titleId = useId()
  return (
    <Modal
      open={open}
      onClose={onCancel ?? (() => {})}
      labelledBy={titleId}
      placement="center"
      overlay={overlay}
      className={styles.dialog}
    >
      <h2 id={titleId} className={styles.title}>
        {title}
      </h2>
      {children && <div className={styles.body}>{children}</div>}
      <div className={styles.actions}>
        <Button fullWidth onClick={onConfirm} disabled={confirmDisabled}>
          {confirmLabel}
        </Button>
        {cancelLabel && (
          <Button variant="text" size="sm" fullWidth onClick={onCancel}>
            {cancelLabel}
          </Button>
        )}
      </div>
    </Modal>
  )
}
