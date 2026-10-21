import type { ReactNode } from 'react'
import type { useFieldIds } from './useFieldIds'
import styles from './TextField.module.css'

/** 入力部品（TextField・SelectField・TextAreaField）に共通の props */
export type FieldProps = {
  /** 入力欄のラベル（DS-1：入力には必ず label を付ける） */
  label: string
  /** ラベルを見た目の上で隠す（読み上げには残る）。授業名など、プレースホルダで分かる欄に使う */
  hideLabel?: boolean
  /** 入力欄の下に出す補足 */
  hint?: ReactNode
  /** 入力エラー。あると枠が danger になり、role="alert" で読み上げる */
  error?: string
}

type FrameProps = FieldProps & {
  ids: ReturnType<typeof useFieldIds>
  className?: string
  /** 入力欄の枠（.box）を含む中身 */
  children: ReactNode
}

/** ラベル＋入力欄＋補足＋エラーの並び（DS-3 の TextField の形）。入力部品の外枠として使う */
export function FieldFrame({
  label,
  hideLabel,
  hint,
  error,
  ids,
  className,
  children,
}: FrameProps) {
  return (
    <div className={[styles.field, className].filter(Boolean).join(' ')}>
      <label htmlFor={ids.inputId} className={hideLabel ? 'visually-hidden' : styles.label}>
        {label}
      </label>
      {children}
      {hint && (
        <div id={ids.hintId} className={styles.hint}>
          {hint}
        </div>
      )}
      {error && (
        <div id={ids.errorId} role="alert" className={styles.error}>
          {error}
        </div>
      )}
    </div>
  )
}
