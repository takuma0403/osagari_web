import type { ButtonHTMLAttributes } from 'react'
import { Icon } from '../Icon/Icon'
import type { IconName } from '../Icon/iconPaths'
import styles from './Chip.module.css'

type Props = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> & {
  label: string
  /** カテゴリの切り替えで選択中（primary 塗り）。指定すると aria-pressed を付ける */
  selected?: boolean
  /** 絞り込み条件のチップにする。押すと条件を外す（primary-soft＋×） */
  onRemove?: () => void
  /** 文字の前に置くアイコン（例 絞り込み） */
  icon?: IconName
}

/** 高さ36pxのチップ（DS-3）。カテゴリの切り替え・絞り込み条件・絞り込みボタンに使う */
export function Chip({ label, selected, onRemove, icon, className, onClick, ...rest }: Props) {
  const classes = [
    styles.chip,
    selected && styles.selected,
    onRemove && styles.removable,
    className,
  ]
    .filter(Boolean)
    .join(' ')

  if (onRemove) {
    return (
      <button
        type="button"
        className={classes}
        aria-label={`${label}を外す`}
        onClick={onRemove}
        {...rest}
      >
        {label}
        <Icon name="close" size={14} strokeWidth={2.2} />
      </button>
    )
  }

  return (
    <button type="button" className={classes} aria-pressed={selected} onClick={onClick} {...rest}>
      {icon && <Icon name={icon} size={16} />}
      {label}
    </button>
  )
}
