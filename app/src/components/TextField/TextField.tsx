import { useId, useState, type ComponentProps, type ReactNode } from 'react'
import { Icon } from '../Icon/Icon'
import styles from './TextField.module.css'

export type TextFieldProps = Omit<ComponentProps<'input'>, 'size' | 'prefix'> & {
  /** 入力欄のラベル（DS-1：入力には必ず label を付ける） */
  label: string
  /** ラベルを見た目の上で隠す（読み上げには残る）。授業名など、プレースホルダで分かる欄に使う */
  hideLabel?: boolean
  /** 入力欄の下に出す補足 */
  hint?: ReactNode
  /** 入力エラー。あると枠が danger になり、role="alert" で読み上げる */
  error?: string
  /** lg 52px・17px（ログイン・登録）／md 48px・15px（出品） */
  size?: 'lg' | 'md'
  /** 入力欄の前に置く固定の文字（例 価格の ¥） */
  prefix?: ReactNode
  /** 入力欄の後ろに置く固定の文字（例 @edu.cc.saga-u.ac.jp） */
  suffix?: ReactNode
  /** 数字（学籍番号・価格）を Outfit で表示する */
  numeric?: boolean
  /** input に足すクラス（幅の調整など） */
  inputClassName?: string
}

/**
 * ラベル付きの1行入力（DS-3）。フォーカス中は primary、エラーは danger の 1.5px 枠。
 * type="password" のときは、表示・非表示を切り替えるボタン（44px）を右端に置く。
 */
export function TextField({
  label,
  hideLabel,
  hint,
  error,
  size = 'lg',
  prefix,
  suffix,
  numeric,
  inputClassName,
  className,
  id,
  type = 'text',
  ...rest
}: TextFieldProps) {
  const autoId = useId()
  const inputId = id ?? autoId
  const hintId = `${inputId}-hint`
  const errorId = `${inputId}-error`
  const [revealed, setRevealed] = useState(false)
  const isPassword = type === 'password'
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(' ') || undefined

  const boxClass = [
    styles.box,
    styles[size],
    error && styles.invalid,
    isPassword && styles.withAction,
  ]
    .filter(Boolean)
    .join(' ')
  const inputClass = [styles.input, numeric && styles.numeric, inputClassName]
    .filter(Boolean)
    .join(' ')

  return (
    <div className={[styles.field, className].filter(Boolean).join(' ')}>
      <label htmlFor={inputId} className={hideLabel ? 'visually-hidden' : styles.label}>
        {label}
      </label>
      <div className={boxClass}>
        {prefix && <span className={`${styles.affix} ${styles.prefix}`}>{prefix}</span>}
        {/* 補足・エラーとの関連付けを呼び出し側の props で上書きさせないよう、rest を先に展開する */}
        <input
          {...rest}
          id={inputId}
          type={isPassword && revealed ? 'text' : type}
          className={inputClass}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
        />
        {suffix && <span className={styles.affix}>{suffix}</span>}
        {isPassword && (
          <button
            type="button"
            className={styles.toggle}
            aria-label={revealed ? 'パスワードを隠す' : 'パスワードを表示'}
            onClick={() => setRevealed((v) => !v)}
          >
            <Icon name={revealed ? 'eyeOff' : 'eye'} size={20} />
          </button>
        )}
      </div>
      {hint && (
        <div id={hintId} className={styles.hint}>
          {hint}
        </div>
      )}
      {error && (
        <div id={errorId} role="alert" className={styles.error}>
          {error}
        </div>
      )}
    </div>
  )
}
