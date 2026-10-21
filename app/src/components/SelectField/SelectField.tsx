import type { ComponentProps } from 'react'
import { FieldFrame, type FieldProps } from '../TextField/Field'
import { useFieldIds } from '../TextField/useFieldIds'
import styles from '../TextField/TextField.module.css'

export type SelectOption<T extends string> = { value: T; label: string }

type Props<T extends string> = Omit<ComponentProps<'select'>, 'value' | 'onChange' | 'size'> &
  FieldProps & {
    options: SelectOption<T>[]
    /** 未選択は空文字 */
    value: T | ''
    onChange: (value: T | '') => void
    /** 未選択のときに出す選択肢の文言。省くと未選択の選択肢を出さない */
    placeholder?: string
  }

/**
 * ラベル付きの選択（出品のカテゴリ・状態）。見た目は TextField の md（48px）と同じ枠で、選択肢はブラウザ標準の一覧を使う。
 */
export function SelectField<T extends string>({
  label,
  hideLabel,
  hint,
  error,
  options,
  value,
  onChange,
  placeholder,
  className,
  id,
  ...rest
}: Props<T>) {
  const ids = useFieldIds(id, { hint, error })
  const boxClass = [styles.box, styles.md, error && styles.invalid].filter(Boolean).join(' ')
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
        <select
          {...rest}
          id={ids.inputId}
          value={value}
          onChange={(e) => onChange(e.target.value as T | '')}
          className={`${styles.input} ${styles.select}`}
          aria-invalid={error ? true : undefined}
          aria-describedby={ids.describedBy}
        >
          {placeholder !== undefined && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </div>
    </FieldFrame>
  )
}
