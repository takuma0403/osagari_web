import { useEffect, useState } from 'react'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Badge } from '../../../components/Badge/Badge'
import { Button } from '../../../components/Button/Button'
import { ConfirmDialog } from '../../../components/ConfirmDialog/ConfirmDialog'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { Toast } from '../../../components/Toast/Toast'
import { TwoColumn } from '../../../components/TwoColumn/TwoColumn'
import { formatPrice } from '../../../lib/format'
import { SalesEntryRow } from '../components/SalesEntryRow/SalesEntryRow'
import { SalesSummary } from '../components/SalesSummary/SalesSummary'
import { useSales } from '../hooks/useSales'
import styles from './SalesPage.module.css'

const TOAST_MS = 3000

/**
 * 売上管理（PAY-5、Sales）。出品者として成立した取引の売上の集計と明細を出し、出金できる売上をまとめて出金する。
 * 金額はサーバーが取引成立時に記録した値をそのまま出し、画面では計算しない（PAY-2）。
 * 出金はフェーズ1ではモックで、記録するだけで実際のお金は動かない（ADR 0029）。取り消せないので確認してから申請する。
 */
export function SalesPage() {
  const { state, reload, payout, paying, payoutError, clearPayoutError } = useSales()
  const [confirming, setConfirming] = useState(false)
  // 続けて申請したときも消すまでの時間を数え直すよう、毎回新しいオブジェクトにする
  const [toast, setToast] = useState<{ message: string } | null>(null)

  useEffect(() => {
    if (toast === null) return
    const timer = setTimeout(() => setToast(null), TOAST_MS)
    return () => clearTimeout(timer)
  }, [toast])

  const openConfirm = () => {
    clearPayoutError()
    setConfirming(true)
  }

  const confirmPayout = async () => {
    const ok = await payout()
    setConfirming(false)
    if (ok) setToast({ message: '出金を申請しました' })
  }

  return (
    <div className={styles.page}>
      <AppHeader title="売上管理" backTo="/mypage" right={<Badge tone="warning">モック</Badge>} />
      <div className={styles.body}>
        {payoutError && (
          <InfoBox tone="danger" icon="alert" role="alert">
            <p>{payoutError}</p>
          </InfoBox>
        )}
        {state.status === 'loading' && (
          <p className={styles.message} role="status">
            読み込み中…
          </p>
        )}
        {state.status === 'error' && (
          <InfoBox tone="danger" icon="alert" role="alert">
            <p>{state.message}</p>
            <Button variant="text" size="sm" onClick={reload}>
              もう一度読み込む
            </Button>
          </InfoBox>
        )}
        {state.status === 'ready' && (
          <>
            {/* PC 幅では売上の概要（左、360px）と明細（右）の2カラム（DS-1） */}
            <div className={styles.columns}>
              <TwoColumn
                variant="narrowStart"
                start={
                  <div className={styles.summary}>
                    <SalesSummary
                      availableAmount={state.sales.availableAmount}
                      awaitingHandoverAmount={state.sales.awaitingHandoverAmount}
                      paidOutAmount={state.sales.paidOutAmount}
                      onPayout={openConfirm}
                      payoutDisabled={paying}
                    />
                  </div>
                }
                end={
                  <>
                    <h2 className={styles.sectionTitle}>売上の明細</h2>
                    {state.sales.entries.length === 0 ? (
                      <p className={styles.message}>まだ売上はありません。</p>
                    ) : (
                      <ul className={styles.entries}>
                        {state.sales.entries.map((entry) => (
                          <li key={entry.transactionId}>
                            <SalesEntryRow entry={entry} />
                          </li>
                        ))}
                      </ul>
                    )}
                  </>
                }
              />
            </div>
            <ConfirmDialog
              open={confirming}
              title="出金を申請しますか？"
              confirmLabel="出金を申請する"
              onConfirm={confirmPayout}
              confirmDisabled={paying}
              cancelLabel="やめる"
              onCancel={() => setConfirming(false)}
            >
              <p>
                出金できる売上{' '}
                <b className={styles.price}>{formatPrice(state.sales.availableAmount)}</b>{' '}
                をすべて出金します。申請した出金は取り消せません。
              </p>
              <p className={styles.mockNote}>
                現在はお試しの仕組み（モック）のため、実際のお金は動きません。
              </p>
            </ConfirmDialog>
          </>
        )}
      </div>
      <Toast message={toast?.message} />
    </div>
  )
}
