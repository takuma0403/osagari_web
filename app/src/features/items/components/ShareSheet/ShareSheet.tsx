import { BottomSheet } from '../../../../components/BottomSheet/BottomSheet'
import { Button } from '../../../../components/Button/Button'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import styles from './ShareSheet.module.css'

type Props = {
  open: boolean
  /** 「リンクをコピー」。コピーできたかのトーストは画面側で出す */
  onCopy: () => void
  onClose: () => void
  /** コピーできなかったときの文 */
  error?: string | null
}

/**
 * 商品ページの共有（BROWSE、DetailShare）。リンクに商品名・価格・状態を添えた文をコピーする。
 * デザインの「ほかのアプリで共有」（Web Share API）は使わない（ADR 0020）。
 */
export function ShareSheet({ open, onCopy, onClose, error }: Props) {
  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      title="リンクをコピー"
      description="リンクと、商品の簡単な説明のテキストがコピーされます。"
    >
      {error && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {error}
        </InfoBox>
      )}
      <Button icon="copy" fullWidth onClick={onCopy}>
        リンクをコピー
      </Button>
      <Button variant="text" size="sm" className={styles.close} onClick={onClose}>
        閉じる
      </Button>
    </BottomSheet>
  )
}
