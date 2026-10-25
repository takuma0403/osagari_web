import { ConfirmDialog } from '../../../../components/ConfirmDialog/ConfirmDialog'
import { formatPrice } from '../../../../lib/format'
import { DealItemSummary } from '../DealItemSummary/DealItemSummary'
import styles from './ReceiptConfirmDialog.module.css'

type Props = {
  open: boolean
  /** 代金（円の整数） */
  price: number
  /** 商品名。渡したときは商品の要約も出す（QRの読み取りの上では、何の取引かを見せる） */
  itemTitle?: string
  /** 「受け取った」 */
  onConfirm: () => void
  /** 「まだ受け取っていない」 */
  onCancel: () => void
  /** 完了の送信中 */
  submitting?: boolean
  /** 完了の送信に失敗したとき */
  error?: string | null
  /** カメラ画面の上では dark */
  overlay?: 'default' | 'dark'
}

/**
 * 「商品を受け取りましたか？」の確認（DEAL-3、DealScan・DealCompleteEnter）。
 * 受け取る前に完了させないよう、QR・完了リンク・完了コードのどれで完了するときも必ず出す。
 */
export function ReceiptConfirmDialog({
  open,
  price,
  itemTitle,
  onConfirm,
  onCancel,
  submitting,
  error,
  overlay,
}: Props) {
  return (
    <ConfirmDialog
      open={open}
      title="商品を受け取りましたか？"
      confirmLabel="受け取った"
      onConfirm={onConfirm}
      confirmDisabled={submitting}
      cancelLabel="まだ受け取っていない"
      onCancel={onCancel}
      overlay={overlay}
    >
      <p>
        「受け取った」を押すと <b className={styles.price}>{formatPrice(price)}</b>{' '}
        が出品者に支払われ、取引が完了します。完了後はアプリでキャンセル・返金できません。
      </p>
      {itemTitle && (
        <div className={styles.item}>
          <DealItemSummary size="sm" title={itemTitle} price={price} tint="textbook" />
        </div>
      )}
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
    </ConfirmDialog>
  )
}
