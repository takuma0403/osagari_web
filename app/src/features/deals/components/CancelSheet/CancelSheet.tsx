import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import { formatPrice } from '../../../../lib/format'
import styles from './CancelSheet.module.css'

type Props = {
  open: boolean
  /** request：自分から申請する／agree：相手の申請に同意を求められた */
  side: 'request' | 'agree'
  /** 返金される代金（円の整数） */
  price: number
  /** 「キャンセルを申請する」「キャンセルに同意する」 */
  onSubmit: () => void
  /** 「やめる」「同意しない」・幕・Esc。どちらも閉じるだけで API は呼ばない（ADR 0026） */
  onClose: () => void
  /** 申請・同意の送信中 */
  submitting?: boolean
  /** 申請・同意の送信に失敗したとき */
  error?: string | null
}

/**
 * キャンセルの申請と同意（DEAL-4、DealCancel）。
 * キャンセルには双方の同意が要る。同意しなければ取引は続く（断る操作はない）。
 */
export function CancelSheet({ open, side, price, onSubmit, onClose, submitting, error }: Props) {
  const request = side === 'request'
  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title={request ? '取引のキャンセルを申請' : 'キャンセルの申請が届いています'}
    >
      <p className={styles.text}>
        {request
          ? 'キャンセルには相手の同意が必要です。相手が同意すると'
          : '相手がこの取引のキャンセルを申請しました。同意すると'}
        取引はキャンセルになり、代金 <b className={styles.num}>{formatPrice(price)}</b>{' '}
        は購入者に返金され、商品は公開中に戻ります。
      </p>
      {request ? (
        <InfoBox tone="neutral" icon="alert" className={styles.notice}>
          キャンセルの回数は記録されます。回数が多い場合は、警告やアカウント停止の対象になります。
        </InfoBox>
      ) : (
        <InfoBox tone="warning" icon="warning">
          同意したあとは取り消せません。受け渡しを続けるなら、同意せずにTeamsで相手と相談してください。
        </InfoBox>
      )}
      {error && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {error}
        </InfoBox>
      )}
      <Button fullWidth onClick={onSubmit} disabled={submitting}>
        {request ? 'キャンセルを申請する' : 'キャンセルに同意する'}
      </Button>
      {request ? (
        <Button variant="text" size="sm" className={styles.dismiss} onClick={onClose}>
          やめる
        </Button>
      ) : (
        <Button variant="ghost" size="md" fullWidth onClick={onClose}>
          同意しない
        </Button>
      )}
    </BottomSheet>
  )
}
