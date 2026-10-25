import { Button } from '../../../../components/Button/Button'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import styles from './CancelStatus.module.css'

type Props = {
  /** requested：自分が申請中／received：相手の申請が届いている */
  kind: 'requested' | 'received'
  /** 「申請を取り下げる」「内容を確認する」 */
  onAction: () => void
  /** 取り下げの送信中 */
  busy?: boolean
}

/**
 * 取引詳細の上部に出すキャンセルの案内（DEAL-4、DealCancel の「申請中」）。
 * 申請中は相手が同意するまで取引が続くことを伝え、取り下げられるようにする。
 * 相手の申請が届いているときは、閉じたシートを開き直せるようにする。
 */
export function CancelStatus({ kind, onAction, busy }: Props) {
  const requested = kind === 'requested'
  return (
    <InfoBox tone="warning" icon="clock" role="status" className={styles.box}>
      <p>
        {requested
          ? 'キャンセルを申請しました。相手が同意するまで取引は続いています。'
          : '相手からキャンセルの申請が届いています。同意しなければ取引は続きます。'}
      </p>
      <Button
        variant="ghost"
        size="sm"
        className={styles.action}
        onClick={onAction}
        disabled={busy}
      >
        {requested ? '申請を取り下げる' : '内容を確認する'}
      </Button>
    </InfoBox>
  )
}
