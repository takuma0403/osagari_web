import { useRef, type KeyboardEvent } from 'react'
import styles from './SegmentedTabs.module.css'

export type SegmentedTabOption<T extends string> = {
  value: T
  label: string
  /** 切り替わる中身（role="tabpanel"）の id */
  panelId?: string
}

type Props<T extends string> = {
  /** タブ全体の名前（例 取引の種類） */
  label: string
  options: SegmentedTabOption<T>[]
  value: T
  onChange: (value: T) => void
  /** 地の色（--color-bg）の上に置くときは白地＋枠にする */
  onBg?: boolean
  /** pill：pill の枠内で選択中を塗る（DealList・AdminDeals）／underline：選択中に下線を引く（MyPage） */
  variant?: 'pill' | 'underline'
}

/**
 * 2択の切り替え（DS-3、DealList・AdminDeals・MyPage）。role="tablist" で、左右キーで選択を移す。
 * 選択中のタブだけを Tab キーで止まる位置にする（ロービング tabindex）。
 */
export function SegmentedTabs<T extends string>({
  label,
  options,
  value,
  onChange,
  onBg,
  variant = 'pill',
}: Props<T>) {
  const refs = useRef<(HTMLButtonElement | null)[]>([])

  const handleKeyDown = (e: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const moves: Record<string, number> = {
      ArrowRight: index + 1,
      ArrowLeft: index - 1,
      Home: 0,
      End: options.length - 1,
    }
    if (!(e.key in moves)) return
    e.preventDefault()
    const next = (moves[e.key] + options.length) % options.length
    onChange(options[next].value)
    refs.current[next]?.focus()
  }

  return (
    <div
      role="tablist"
      aria-label={label}
      className={[styles.tablist, styles[variant], onBg && styles.onBg].filter(Boolean).join(' ')}
    >
      {options.map((option, i) => {
        const selected = option.value === value
        return (
          <button
            key={option.value}
            ref={(el) => {
              refs.current[i] = el
            }}
            type="button"
            role="tab"
            aria-selected={selected}
            aria-controls={option.panelId}
            tabIndex={selected ? 0 : -1}
            className={styles.tab}
            onClick={() => onChange(option.value)}
            onKeyDown={(e) => handleKeyDown(e, i)}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}
