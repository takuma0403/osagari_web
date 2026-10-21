import { useState, type ComponentProps, type ReactNode } from 'react'
import { Icon } from '../Icon/Icon'
import { FieldFrame, type FieldProps } from './Field'
import { useFieldIds } from './useFieldIds'
import styles from './TextField.module.css'

export type TextFieldProps = Omit<ComponentProps<'input'>, 'size' | 'prefix'> &
  FieldProps & {
    /** lg 52px・17px（ログイン・登録）／md 48px・15px（出品）／sm 44px・14px（出品のカテゴリ別の追加項目） */
    size?: 'lg' | 'md' | 'sm'
    /** 枠の色。primary は primary-border（出品のカテゴリ別の追加項目。淡色の地の上に置く） */
    tone?: 'default' | 'primary'
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
  tone = 'default',
  prefix,
  suffix,
  numeric,
  inputClassName,
  className,
  id,
  type = 'text',
  ...rest
}: TextFieldProps) {
  const ids = useFieldIds(id, { hint, error })
  const [revealed, setRevealed] = useState(false)
  const isPassword = type === 'password'

  const boxClass = [
    styles.box,
    styles[size],
    tone === 'primary' && styles.tonePrimary,
    error && styles.invalid,
    isPassword && styles.withAction,
  ]
    .filter(Boolean)
    .join(' ')
  const inputClass = [styles.input, numeric && styles.numeric, inputClassName]
    .filter(Boolean)
    .join(' ')

  return (
    <FieldFrame
      label={label}
      hideLabel={hideLabel}
      hint={hint}
      error={error}
      ids={ids}
      className={className}
    >
      <div className={boxClass}>
        {prefix && <span className={`${styles.affix} ${styles.prefix}`}>{prefix}</span>}
        {/* 補足・エラーとの関連付けを呼び出し側の props で上書きさせないよう、rest を先に展開する */}
        <input
          {...rest}
          id={ids.inputId}
          type={isPassword && revealed ? 'text' : type}
          className={inputClass}
          aria-invalid={error ? true : undefined}
          aria-describedby={ids.describedBy}
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
    </FieldFrame>
  )
}
