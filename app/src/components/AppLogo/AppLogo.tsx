import logo from '../../assets/logo.png'
import styles from './AppLogo.module.css'

type Props = {
  /** 表示する一辺の大きさ（px）。デザインではヘッダー 32、管理画面 30 */
  size: number
  /** 角丸を変えるときなどに足すクラス（Login の 72px は --radius-dialog） */
  className?: string
}

/**
 * アプリのロゴ。元画像（docs/design/assets/logo.png）は余白があるため、デザインと同じく拡大して切り抜く。
 * 文字のタイトルと並べて使うので、画像自体は装飾（alt なし）として扱う。
 */
export function AppLogo({ size, className }: Props) {
  return (
    <span
      className={[styles.frame, className].filter(Boolean).join(' ')}
      style={{ width: size, height: size }}
    >
      <img className={styles.image} src={logo} alt="" />
    </span>
  )
}
