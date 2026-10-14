import { Icon } from '../Icon/Icon'
import type { IconName } from '../Icon/iconPaths'
import styles from './Toast.module.css'

type Props = {
  /** 表示中の文。空なら何も出さない。消すタイミングは利用側で決める */
  message?: string
  icon?: IconName
}

/**
 * 操作の結果を短く知らせる紺地のトースト（DS-3）。role="status" で読み上げる。
 * 読み上げ用の領域を先に置いておく必要があるため、表示しないときも message を空にして描画しておく。
 */
export function Toast({ message, icon = 'check' }: Props) {
  return (
    <div role="status" className={styles.region}>
      {message && (
        <div className={styles.toast}>
          <Icon name={icon} size={18} strokeWidth={2.4} />
          {message}
        </div>
      )}
    </div>
  )
}
