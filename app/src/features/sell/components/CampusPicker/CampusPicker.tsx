import { useId } from 'react'
import type { Campus } from '../../api'
import { campusOptions } from '../../form'
import styles from './CampusPicker.module.css'

type Props = {
  /** 未選択は空文字 */
  value: Campus | ''
  onChange: (value: Campus) => void
  error?: string
  disabled?: boolean
}

/**
 * 受け渡し希望キャンパス（本庄・鍋島）の選択（ITEM）。白地の枠の中に2つ並べ、選択中は primary の塗り。
 * 見た目はボタンだが、ラジオボタン（input type="radio"）で作り、矢印キーでの選択をブラウザに任せる。
 */
export function CampusPicker({ value, onChange, error, disabled }: Props) {
  const name = useId()
  const errorId = `${name}-error`
  return (
    <fieldset
      className={styles.fieldset}
      disabled={disabled}
      aria-describedby={error ? errorId : undefined}
    >
      <legend className={styles.legend}>受け渡し希望キャンパス</legend>
      <div className={[styles.group, error && styles.invalid].filter(Boolean).join(' ')}>
        {campusOptions.map((option) => (
          <label key={option.value} className={styles.option}>
            <input
              type="radio"
              name={name}
              value={option.value}
              checked={value === option.value}
              onChange={() => onChange(option.value)}
              className="visually-hidden"
            />
            {option.label}
          </label>
        ))}
      </div>
      {error && (
        <p id={errorId} role="alert" className={styles.error}>
          {error}
        </p>
      )}
    </fieldset>
  )
}
