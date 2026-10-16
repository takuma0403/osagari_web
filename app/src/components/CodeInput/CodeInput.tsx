import { useEffect, useId, useRef, type ClipboardEvent, type KeyboardEvent } from 'react'
import styles from './CodeInput.module.css'

const LENGTH = 6

type Props = {
  /** 入力中のコード（数字だけ、最大6桁） */
  value: string
  onChange: (code: string) => void
  /** 6マスをまとめた名前。既定は読み上げだけ */
  label?: string
  /** label を見出しとして画面にも出す（再確認の「メールに届いた6桁のコード」） */
  showLabel?: boolean
  /** 入力エラー（例 コードが違います。あと3回…）。role="alert" で読み上げる */
  error?: string
  /** 有効期限の残り（例 9:41）。カウントダウンは利用側で行う */
  expiresIn?: string
  /** コードの再送ボタン。待ち時間中は disabled にして label に残り時間を入れる */
  resend?: { label: string; onClick: () => void; disabled?: boolean }
  disabled?: boolean
}

const digitsOf = (text: string) => text.normalize('NFKC').replace(/\D/g, '')

/**
 * 6桁の認証コードの入力（DS-3、AUTHN-1）。1桁ずつのマスに分け、入力すると次のマスへ進む。
 * 貼り付けやワンタイムコードの自動入力では、6桁をまとめて各マスに振り分ける。
 */
export function CodeInput({
  value,
  onChange,
  label = '認証コード',
  showLabel,
  error,
  expiresIn,
  resend,
  disabled,
}: Props) {
  const errorId = useId()
  const refs = useRef<(HTMLInputElement | null)[]>([])
  // onChange の結果が描画される前に次のマスへ移るため、入力済みの桁はここで先に持っておく
  const latest = useRef(value)
  useEffect(() => {
    latest.current = value
  }, [value])
  const digits = Array.from({ length: LENGTH }, (_, i) => value[i] ?? '')

  const focus = (index: number) => {
    const target = refs.current[Math.max(0, Math.min(index, LENGTH - 1))]
    target?.focus()
    target?.select()
  }

  /** index のマスから先を text の数字で置き換える */
  const fill = (index: number, text: string) => {
    const incoming = digitsOf(text)
    if (!incoming) return
    const next = [...digits]
    incoming
      .slice(0, LENGTH - index)
      .split('')
      .forEach((d, i) => (next[index + i] = d))
    latest.current = next.join('').slice(0, LENGTH)
    onChange(latest.current)
    focus(index + incoming.length)
  }

  const handleKeyDown = (index: number, e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Backspace') {
      e.preventDefault()
      if (digits[index]) {
        latest.current = value.slice(0, index) + value.slice(index + 1)
        onChange(latest.current)
      } else if (index > 0) {
        latest.current = value.slice(0, index - 1) + value.slice(index)
        onChange(latest.current)
        focus(index - 1)
      }
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault()
      focus(index - 1)
    } else if (e.key === 'ArrowRight') {
      e.preventDefault()
      focus(index + 1)
    }
  }

  const handlePaste = (index: number, e: ClipboardEvent<HTMLInputElement>) => {
    e.preventDefault()
    fill(index, e.clipboardData.getData('text'))
  }

  return (
    <fieldset className={[styles.root, error && styles.invalid].filter(Boolean).join(' ')}>
      <legend className={showLabel ? styles.label : 'visually-hidden'}>{label}</legend>
      <div className={styles.boxes}>
        {digits.map((digit, i) => (
          <input
            key={i}
            ref={(el) => {
              refs.current[i] = el
            }}
            className={styles.box}
            value={digit}
            aria-label={`${i + 1}桁目`}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? errorId : undefined}
            inputMode="numeric"
            autoComplete={i === 0 ? 'one-time-code' : 'off'}
            disabled={disabled}
            // 前のマスが空のうちは、空いている最初のマスから入力させる
            onFocus={(e) =>
              i > latest.current.length ? focus(latest.current.length) : e.target.select()
            }
            onChange={(e) => fill(i, e.target.value.replace(digit, '') || e.target.value)}
            onKeyDown={(e) => handleKeyDown(i, e)}
            onPaste={(e) => handlePaste(i, e)}
          />
        ))}
      </div>
      {error && (
        <div id={errorId} role="alert" className={styles.error}>
          {error}
        </div>
      )}
      {(expiresIn || resend) && (
        <div className={styles.meta}>
          {expiresIn ? (
            <span>
              有効期限 あと <b className={styles.time}>{expiresIn}</b>
            </span>
          ) : (
            <span />
          )}
          {resend && (
            <button
              type="button"
              className={styles.resend}
              onClick={resend.onClick}
              disabled={resend.disabled}
            >
              {resend.label}
            </button>
          )}
        </div>
      )}
    </fieldset>
  )
}
