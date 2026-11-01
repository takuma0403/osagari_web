import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { isApiError } from '../../../api/client'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Badge } from '../../../components/Badge/Badge'
import { Button } from '../../../components/Button/Button'
import { ConfirmDialog } from '../../../components/ConfirmDialog/ConfirmDialog'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { TwoColumn } from '../../../components/TwoColumn/TwoColumn'
import { categoryTint } from '../../../lib/itemOptions'
import { itemPath } from '../../items/itemDisplay'
import {
  acknowledgeTransaction,
  approveTransaction,
  rejectTransaction,
  type PurchaseRequest,
  type Transaction,
} from '../api'
import { DealConfirmSheet } from '../components/DealConfirmSheet/DealConfirmSheet'
import { DealItemSummary } from '../components/DealItemSummary/DealItemSummary'
import { DealSteps } from '../components/DealSteps/DealSteps'
import { RequestCard } from '../components/RequestCard/RequestCard'
import { usePurchaseRequests } from '../hooks/usePurchaseRequests'
import styles from './DealRequestsPage.module.css'

/** 期限までの残りを数え直す間隔 */
const TICK_MS = 60 * 1000
const unexpectedMessage = '操作できませんでした。もう一度お試しください'

const messageOf = (err: unknown) => (isApiError(err) ? err.message : unexpectedMessage)

/** 確認ダイアログで確かめている操作 */
type Pending = { kind: 'approve' | 'reject'; request: PurchaseRequest }

/**
 * 購入希望の一覧と承認（DEAL-1、DealRequests）。出品者が /sell/:itemId/requests で開く。
 * 承認できるのは1人だけで、承認すると代金を預かり、購入者の学籍番号が開示され、ほかの購入希望は自動で却下される。
 * 承認の直後に取引成立の確認モーダル（DEAL-2、DealConfirm）を出し、「確認しました」で取引詳細へ進む。
 */
export function DealRequestsPage() {
  const { itemId = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const { state, reload } = usePurchaseRequests(itemId)
  const [now, setNow] = useState(() => Date.now())
  const [pending, setPending] = useState<Pending | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)
  const [approved, setApproved] = useState<Transaction | null>(null)
  const [acknowledging, setAcknowledging] = useState(false)
  const [acknowledgeError, setAcknowledgeError] = useState<string | null>(null)

  // 承認期限までの残りを1分ごとに数え直す
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), TICK_MS)
    return () => clearInterval(timer)
  }, [])

  // アプリ内の履歴がない（URL を直接開いた）ときはマイページへ戻る
  const back = () => (location.key === 'default' ? navigate('/mypage') : navigate(-1))
  const header = <AppHeader title="取引詳細" onBack={back} />

  if (state.status !== 'ready') {
    return (
      <div className={styles.page}>
        {header}
        <div className={styles.status}>
          {state.status === 'loading' && <p role="status">読み込み中…</p>}
          {state.status === 'notFound' && (
            <>
              <h2 className={styles.statusTitle}>商品が見つかりません</h2>
              <p className={styles.muted}>自分の出品だけ、購入希望を確認できます。</p>
              <Link to="/">商品一覧へ戻る</Link>
            </>
          )}
          {state.status === 'error' && (
            <InfoBox tone="danger" icon="alert" role="alert">
              <p>{state.message}</p>
              <Button variant="text" size="sm" onClick={reload}>
                もう一度読み込む
              </Button>
            </InfoBox>
          )}
        </div>
      </div>
    )
  }

  const { item, requests } = state

  const confirmPending = async () => {
    if (!pending) return
    setSubmitting(true)
    setActionError(null)
    try {
      if (pending.kind === 'approve') {
        setApproved(await approveTransaction(pending.request.transactionId))
      } else {
        await rejectTransaction(pending.request.transactionId)
        reload()
      }
    } catch (err) {
      setActionError(messageOf(err))
      // 期限切れ・ほかの人で成立済みなど状態が変わっていた（409）ときは、いまの一覧を取り直す
      if (isApiError(err) && err.code === 'invalid_status_transition') reload()
    } finally {
      setPending(null)
      setSubmitting(false)
    }
  }

  const acknowledge = async () => {
    if (!approved) return
    setAcknowledging(true)
    setAcknowledgeError(null)
    try {
      await acknowledgeTransaction(approved.id)
      navigate(`/deals/${encodeURIComponent(approved.id)}`, { replace: true })
    } catch (err) {
      setAcknowledgeError(messageOf(err))
      setAcknowledging(false)
    }
  }

  return (
    <div className={styles.page}>
      {header}
      <main className={styles.body}>
        {/* PC 幅では商品（左）と操作（右）の2カラム（DS-1） */}
        <TwoColumn
          start={
            <section className={styles.itemCard}>
              <DealItemSummary
                size="sm"
                title={item.title}
                price={item.price ?? 0}
                imageUrl={item.images[0]?.url}
                tint={item.category ? categoryTint[item.category] : 'textbook'}
                aside={<Badge tone="warning">あなたが出品</Badge>}
              />
              <DealSteps status={item.status} />
            </section>
          }
          end={
            <>
              {actionError && (
                <InfoBox tone="danger" icon="alert" role="alert">
                  {actionError}
                </InfoBox>
              )}
              {requests.length === 0 ? (
                <div className={styles.empty}>
                  <p>承認を待っている購入希望はありません。</p>
                  <Link to={itemPath(item.shareId)}>商品ページを見る</Link>
                </div>
              ) : (
                <>
                  <div className={styles.listHeading}>
                    <h2 className={styles.count}>購入希望 {requests.length}件</h2>
                    <span className={styles.muted}>承認できるのは1人だけです</span>
                  </div>
                  {requests.map((request) => (
                    <RequestCard
                      key={request.transactionId}
                      request={request}
                      now={now}
                      disabled={submitting || approved !== null}
                      onApprove={() => setPending({ kind: 'approve', request })}
                      onReject={() => setPending({ kind: 'reject', request })}
                    />
                  ))}
                </>
              )}
            </>
          }
        />
      </main>
      <ConfirmDialog
        open={pending?.kind === 'approve'}
        title="この人に決めますか？"
        confirmLabel="この人に決める"
        onConfirm={confirmPending}
        cancelLabel="やめる"
        onCancel={() => setPending(null)}
        confirmDisabled={submitting}
      >
        承認すると代金を預かり、取引が成立して相手の学籍番号が表示されます。ほかの購入希望は自動で見送られます。
      </ConfirmDialog>
      <ConfirmDialog
        open={pending?.kind === 'reject'}
        title="この購入希望を見送りますか？"
        confirmLabel="見送る"
        onConfirm={confirmPending}
        cancelLabel="やめる"
        onCancel={() => setPending(null)}
        confirmDisabled={submitting}
      >
        見送った購入希望は元に戻せません。
      </ConfirmDialog>
      {approved && (
        <DealConfirmSheet
          open
          price={approved.price}
          studentNumber={approved.counterpartStudentNumber ?? ''}
          onAcknowledge={acknowledge}
          acknowledging={acknowledging}
          error={acknowledgeError}
        />
      )}
    </div>
  )
}
