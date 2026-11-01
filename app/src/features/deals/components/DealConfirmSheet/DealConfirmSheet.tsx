import { useState } from 'react'
import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { Icon } from '../../../../components/Icon/Icon'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import { formatPrice } from '../../../../lib/format'
import { copyText } from '../../../../lib/share'
import styles from './DealConfirmSheet.module.css'

const copyFailedMessage = 'コピーできませんでした。学籍番号を書き写してください'

type Props = {
  open: boolean
  /** 預かった代金（円の整数） */
  price: number
  /** 購入者の学籍番号（承認の応答で開示されたもの。DEAL-1） */
  studentNumber: string
  /** 「確認しました」 */
  onAcknowledge: () => void
  /** 「確認しました」の送信中 */
  acknowledging?: boolean
  /** 「確認しました」の送信に失敗したとき */
  error?: string | null
}

/**
 * 取引成立の確認モーダル（DEAL-2、DealConfirm）。出品者が承認した直後に出す。
 * QRの読み取りで入金が確定することを必ず読んでもらうため、つまみを出さず、幕・Esc では閉じない。
 */
export function DealConfirmSheet({
  open,
  price,
  studentNumber,
  onAcknowledge,
  acknowledging,
  error,
}: Props) {
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState<string | null>(null)

  const copy = () => {
    setCopyError(null)
    copyText(studentNumber).then(
      () => setCopied(true),
      () => setCopyError(copyFailedMessage),
    )
  }

  return (
    <BottomSheet
      open={open}
      // 「確認しました」を押すまで閉じない
      onClose={() => {}}
      handle={false}
      leading={
        <span className={styles.check}>
          <Icon name="check" size={28} strokeWidth={2.2} />
        </span>
      }
      title="取引が成立しました"
      description={
        <>
          代金 {formatPrice(price)} は運営が預かっています。
          <br />
          Teamsで相手に連絡して受け渡しを決めてください。
        </>
      }
    >
      <div className={styles.studentNumber}>
        <div className={styles.numberText}>
          <span className={styles.numberLabel}>購入者の学籍番号</span>
          <span className={styles.number}>{studentNumber}</span>
        </div>
        <Button variant="ghost" size="sm" onClick={copy}>
          {copied ? 'コピーしました' : 'コピー'}
        </Button>
      </div>
      {copyError && (
        <p role="alert" className={styles.error}>
          {copyError}
        </p>
      )}
      <InfoBox tone="warning" icon="qr" className={styles.notice}>
        入金は、受け渡しの場で購入者がQRコードを読み取った時点で確定します。受け渡しの際は、忘れずにQRコードを読み取ってもらってください。
      </InfoBox>
      {error && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {error}
        </InfoBox>
      )}
      <Button fullWidth onClick={onAcknowledge} disabled={acknowledging}>
        確認しました
      </Button>
    </BottomSheet>
  )
}
