import type { ComponentProps } from 'react'
import { FieldFrame, type FieldProps } from '../TextField/Field'
import { useFieldIds } from '../TextField/useFieldIds'
import styles from '../TextField/TextField.module.css'

type Props = ComponentProps<'textarea'> & FieldProps

/** ラベル付きの複数行入力（出品の説明）。見た目は TextField と同じ枠で、高さは rows で決める */
export function TextAreaField({ label, hideLabel, hint, error, className, id, ...rest }: Props) {
  const ids = useFieldIds(id, { hint, error })
  const boxClass = [styles.box, styles.multiline, error && styles.invalid].filter(Boolean).join(' ')
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
        <textarea
          {...rest}
          id={ids.inputId}
          className={`${styles.input} ${styles.textarea}`}
          aria-invalid={error ? true : undefined}
          aria-describedby={ids.describedBy}
        />
      </div>
    </FieldFrame>
  )
}
