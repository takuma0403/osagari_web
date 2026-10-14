import { iconPaths, type IconName } from './iconPaths'

type Props = {
  name: IconName
  /** 表示する一辺の大きさ（px）。デザインではナビ・ヘッダー 22、入力欄・案内 18〜20 */
  size?: number
  /** 線の太さ。デザインでは 1.8〜2.4 */
  strokeWidth?: number
  className?: string
}

/**
 * 線画のアイコン（DS-1：絵文字は使わない）。意味はボタンの aria-label や隣の文字で伝えるので、アイコン自体は装飾として読み上げない。
 */
export function Icon({ name, size = 22, strokeWidth = 1.8, className }: Props) {
  return (
    <svg
      className={className}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {iconPaths[name]}
    </svg>
  )
}
