import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import styles from './Modal.module.css'

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

type Props = {
  open: boolean
  /** Esc・背面の幕を押したとき */
  onClose: () => void
  /** 見出しの id（aria-labelledby） */
  labelledBy: string
  /** bottom：ボトムシート／center：中央のモーダル */
  placement: 'bottom' | 'center'
  /** 幕の色。カメラ画面の上では dark */
  overlay?: 'default' | 'dark'
  className?: string
  children: ReactNode
}

/**
 * BottomSheet・ConfirmDialog の土台（DS-1：role="dialog" と aria-labelledby）。
 * 開いている間は背面のスクロールを止め、フォーカスを中に閉じ込める。閉じたら開く前の要素にフォーカスを戻す。
 * 最初のフォーカスはダイアログ自体に置き、確定ボタンを不用意に押させない（受け渡し完了など取り消せない操作があるため）。
 */
export function Modal({
  open,
  onClose,
  labelledBy,
  placement,
  overlay = 'default',
  className,
  children,
}: Props) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const onCloseRef = useRef(onClose)
  useEffect(() => {
    onCloseRef.current = onClose
  }, [onClose])

  useEffect(() => {
    if (!open) return
    const previous = document.activeElement as HTMLElement | null
    const dialog = dialogRef.current
    dialog?.focus()
    const { overflow } = document.body.style
    document.body.style.overflow = 'hidden'

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onCloseRef.current()
        return
      }
      if (e.key !== 'Tab' || !dialog) return
      const focusable = Array.from(dialog.querySelectorAll<HTMLElement>(FOCUSABLE))
      if (focusable.length === 0) {
        e.preventDefault()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      const active = document.activeElement
      if (e.shiftKey && (active === first || active === dialog)) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && active === last) {
        e.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = overflow
      previous?.focus()
    }
  }, [open])

  if (!open) return null

  return createPortal(
    <>
      <div
        className={[styles.overlay, overlay === 'dark' && styles.dark].filter(Boolean).join(' ')}
        onClick={onClose}
        data-testid="modal-overlay"
      />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        tabIndex={-1}
        className={[styles.dialog, styles[placement], className].filter(Boolean).join(' ')}
      >
        {children}
      </div>
    </>,
    document.body,
  )
}
