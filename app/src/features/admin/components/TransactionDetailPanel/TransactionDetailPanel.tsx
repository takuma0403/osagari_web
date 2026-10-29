import { useState } from 'react'
import { Badge } from '../../../../components/Badge/Badge'
import { Button } from '../../../../components/Button/Button'
import { ConfirmDialog } from '../../../../components/ConfirmDialog/ConfirmDialog'
import { DefinitionList } from '../../../../components/DefinitionList/DefinitionList'
import { formatFeeRate } from '../../../../lib/fee'
import { formatPrice } from '../../../../lib/format'
import {
  cancelRequestText,
  isTransactionCancellable,
  transactionStatusDisplay,
} from '../../adminDisplay'
import type { AdminTransaction } from '../../api'
import { DetailPanel, PanelSection } from '../DetailPanel/DetailPanel'
import styles from './TransactionDetailPanel.module.css'

type Props = {
  transaction: AdminTransaction
  running: boolean
  error: string | null
  /** 確認のあとキャンセル・返金する。終えたか（useAdminAction の run）を返し、終えたら確認を閉じる */
  onCancel: () => Promise<boolean>
}

/**
 * 取引の詳細と、管理者によるキャンセル・返金（モック）（AdminDeals の取引。ADMIN, DEAL-4、ADR 0034）。
 * 管理者には承認前でも当事者の学籍番号を見せる。キャンセルできるのは取引中だけで、受け渡し完了後などはボタンを押せない。
 */
export function TransactionDetailPanel({ transaction: tx, running, error, onCancel }: Props) {
  const [confirming, setConfirming] = useState(false)

  const confirm = async () => {
    if (await onCancel()) setConfirming(false)
  }

  const num = (value: string) => <span className={styles.strongNum}>{value}</span>

  return (
    <DetailPanel label="取引の詳細">
      <div className={styles.head}>
        <h2 className={styles.title}>{tx.item.title}</h2>
        <Badge tone="warning">決済はモック</Badge>
      </div>
      <DefinitionList
        items={[
          { term: '出品者', description: num(tx.sellerStudentNumber) },
          { term: '購入者', description: num(tx.buyerStudentNumber) },
          { term: '金額', description: num(formatPrice(tx.price)) },
          {
            term: '手数料率',
            // 料率は取引が成立したときに決まる（PAY-2）
            description:
              tx.feeRateBasisPoints === null
                ? '成立前'
                : `${formatFeeRate(tx.feeRateBasisPoints)}（この取引に適用）`,
          },
          { term: 'ステータス', description: transactionStatusDisplay(tx).label },
          { term: 'キャンセル申請', description: cancelRequestText(tx) },
        ]}
      />
      <PanelSection
        note="トラブル対応として管理者が取引をキャンセルし、代金を購入者に返金します。受け渡し完了後の取引にはできません。"
        error={confirming ? null : error}
      >
        <Button
          variant="danger"
          size="sm"
          onClick={() => setConfirming(true)}
          disabled={running || !isTransactionCancellable(tx)}
        >
          キャンセル・返金（モック）を行う
        </Button>
      </PanelSection>
      <ConfirmDialog
        open={confirming}
        title="取引をキャンセルしますか？"
        confirmLabel={running ? 'キャンセル中…' : 'キャンセル・返金する'}
        onConfirm={confirm}
        cancelLabel="やめる"
        onCancel={() => setConfirming(false)}
        confirmDisabled={running}
      >
        当事者の同意がなくても取引をキャンセルし、代金 {num(formatPrice(tx.price))}{' '}
        を購入者に返金（モック）します。元に戻せません。
        {error && (
          <p role="alert" className={styles.error}>
            {error}
          </p>
        )}
      </ConfirmDialog>
    </DetailPanel>
  )
}
