import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { isApiError } from '../../../api/client'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Badge } from '../../../components/Badge/Badge'
import { Button, ButtonLink } from '../../../components/Button/Button'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { formatPrice, formatShortDateTime } from '../../../lib/format'
import { buildHandoverUrl } from '../../../lib/handoverLink'
import {
  createHandoverToken,
  type HandoverToken,
  type HandoverTokenKind,
  type Transaction,
} from '../api'
import { DealItemSummary } from '../components/DealItemSummary/DealItemSummary'
import { DealSteps } from '../components/DealSteps/DealSteps'
import { HandoverIssued } from '../components/HandoverIssued/HandoverIssued'
import { HandoverPanel } from '../components/HandoverPanel/HandoverPanel'
import { useTransaction } from '../hooks/useTransaction'
import styles from './DealDetailPage.module.css'

/** 出品者が受け渡しQRを表示している間、購入者が完了したかを確かめる間隔（ADR 0025） */
const POLL_MS = 5000
const unexpectedMessage = '発行できませんでした。もう一度お試しください'

const messageOf = (err: unknown) => (isApiError(err) ? err.message : unexpectedMessage)

/** 取引中・完了以外のステータスの説明 */
const statusText: Record<Exclude<Transaction['status'], 'in_deal' | 'completed'>, string> = {
  requested: '出品者の承認を待っています。',
  rejected: 'この購入希望は成立しませんでした。',
  expired: '承認期限を過ぎたため、この購入希望は成立しませんでした。',
  cancelled: 'この取引はキャンセルされました。',
}

/**
 * 取引詳細（DEAL-1〜3）。見る人と取引のステータスで出し分ける。
 * - 出品者・取引中：受け渡しQR（DealQR）と、読み取ってもらい忘れたときの完了リンク・完了コードの発行（DealCompleteIssue）
 * - 購入者・取引中：QRの読み取り・完了コードの入力への導線
 * - 受け渡し完了：完了の表示（出品者が QR を表示している間は5秒ごとに取り直し、購入者が完了したら切り替える）
 * キャンセルの申請（DEAL-4）は #30 で足す。
 */
export function DealDetailPage() {
  const { dealId = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const [issued, setIssued] = useState<HandoverToken | null>(null)
  const { state, reload } = useTransaction(dealId)
  const transaction = state.status === 'ready' ? state.transaction : null
  // 完了リンク・完了コードの表示中は、出品者がそれを Teams に送っている間なので取り直さない
  const showsQr = transaction?.role === 'seller' && transaction.status === 'in_deal' && !issued

  // アプリ内の履歴がない（URL を直接開いた）ときは取引一覧へ戻る
  const back = () => (location.key === 'default' ? navigate('/deals') : navigate(-1))

  if (!transaction) {
    return (
      <div className={styles.page}>
        <AppHeader title="取引詳細" onBack={back} />
        <div className={styles.status}>
          {state.status === 'loading' && <p role="status">読み込み中…</p>}
          {state.status === 'notFound' && (
            <>
              <h2 className={styles.statusTitle}>取引が見つかりません</h2>
              <p className={styles.muted}>自分が購入者か出品者の取引だけ表示できます。</p>
              <Link to="/deals">取引一覧へ戻る</Link>
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

  const inDeal = transaction.status === 'in_deal'
  const header = (
    <AppHeader
      title={inDeal ? '受け渡し' : '取引詳細'}
      onBack={issued ? () => setIssued(null) : back}
      right={inDeal && <Badge>取引中</Badge>}
    />
  )

  return (
    <div className={styles.page}>
      {header}
      <main className={styles.body}>
        {transaction.status === 'completed' ? (
          <CompletedView transaction={transaction} />
        ) : transaction.status !== 'in_deal' ? (
          <section className={styles.card}>
            <Summary transaction={transaction} />
            <DealSteps status={transaction.item.status} />
            <p className={styles.text}>{statusText[transaction.status]}</p>
          </section>
        ) : transaction.role === 'buyer' ? (
          <BuyerView transaction={transaction} />
        ) : issued ? (
          <>
            <HandoverIssued
              kind={issued.kind === 'link' ? 'link' : 'code'}
              value={
                issued.kind === 'link'
                  ? buildHandoverUrl(window.location.origin, dealId, 'link', issued.token ?? '')
                  : (issued.code ?? '')
              }
              expiresAt={issued.expiresAt}
            />
            <Button variant="text" size="sm" onClick={() => setIssued(null)}>
              受け渡し画面に戻る
            </Button>
          </>
        ) : (
          <SellerHandover dealId={dealId} onIssued={setIssued} onExpired={reload} />
        )}
      </main>
      {showsQr && <CompletionWatcher onTick={reload} />}
    </div>
  )
}

/** 出品者が受け渡しQRを表示している間だけ、画面が前面なら5秒ごとに取引を取り直す（ADR 0025） */
function CompletionWatcher({ onTick }: { onTick: () => void }) {
  useEffect(() => {
    const timer = setInterval(() => {
      if (document.visibilityState === 'visible') onTick()
    }, POLL_MS)
    return () => clearInterval(timer)
  }, [onTick])
  return null
}

function Summary({ transaction }: { transaction: Transaction }) {
  return (
    <DealItemSummary
      size="sm"
      title={transaction.item.title}
      price={transaction.price}
      tint="textbook"
      aside={transaction.role === 'seller' && <Badge tone="warning">あなたが出品</Badge>}
    />
  )
}

/** 出品者・取引中（DealQR）。開いたら受け渡しQRを発行し、期限切れ・「表示し直す」で発行し直す */
function SellerHandover({
  dealId,
  onIssued,
  onExpired,
}: {
  dealId: string
  onIssued: (token: HandoverToken) => void
  /** 発行できなかった（取引中でなくなった）とき、取引を取り直す */
  onExpired: () => void
}) {
  const [qr, setQr] = useState<HandoverToken | null>(null)
  const [issueCount, setIssueCount] = useState(0)
  const [issuing, setIssuing] = useState<HandoverTokenKind | null>('qr')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    createHandoverToken(dealId, 'qr').then(
      (token) => {
        if (cancelled) return
        setQr(token)
        setIssuing(null)
      },
      (err: unknown) => {
        if (cancelled) return
        setError(messageOf(err))
        setIssuing(null)
        if (isApiError(err) && err.code === 'invalid_status_transition') onExpired()
      },
    )
    return () => {
      cancelled = true
    }
  }, [dealId, issueCount, onExpired])

  const reissue = () => {
    setIssuing('qr')
    setError(null)
    setIssueCount((n) => n + 1)
  }

  const issue = async (kind: 'link' | 'code') => {
    setIssuing(kind)
    setError(null)
    try {
      onIssued(await createHandoverToken(dealId, kind))
    } catch (err) {
      setError(messageOf(err))
      setIssuing(null)
      if (isApiError(err) && err.code === 'invalid_status_transition') onExpired()
    }
  }

  return (
    <>
      {error && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {error}
        </InfoBox>
      )}
      <HandoverPanel
        url={qr?.token ? buildHandoverUrl(window.location.origin, dealId, 'qr', qr.token) : null}
        code={qr?.code ?? null}
        expiresAt={qr?.expiresAt ?? null}
        onReissue={reissue}
        reissuing={issuing !== null}
      />
      <section className={styles.card}>
        <h2 className={styles.cardTitle}>読み取ってもらい忘れたとき</h2>
        <p className={styles.muted}>
          完了リンクか6桁のコードを発行してTeamsで送ってください（24時間有効）。
        </p>
        <div className={styles.issueButtons}>
          <Button
            variant="secondary"
            size="sm"
            fullWidth
            onClick={() => issue('link')}
            disabled={issuing !== null}
          >
            完了リンクを発行
          </Button>
          <Button
            variant="secondary"
            size="sm"
            fullWidth
            onClick={() => issue('code')}
            disabled={issuing !== null}
          >
            完了コードを発行
          </Button>
        </div>
      </section>
    </>
  )
}

/** 購入者・取引中。商品を受け取ってから、出品者の QR を読み取る（またはコードを入力する） */
function BuyerView({ transaction }: { transaction: Transaction }) {
  const base = `/deals/${encodeURIComponent(transaction.id)}`
  return (
    <>
      <section className={styles.card}>
        <Summary transaction={transaction} />
        {transaction.counterpartStudentNumber && (
          <p className={styles.muted}>
            出品者の学籍番号{' '}
            <span className={styles.studentNumber}>{transaction.counterpartStudentNumber}</span>
          </p>
        )}
        <DealSteps status={transaction.item.status} />
      </section>
      <InfoBox icon="qr">
        商品を受け取って確認したら、出品者が表示したQRコードを読み取ってください。読み取ると代金が出品者に支払われ、取引が完了します。
      </InfoBox>
      <ButtonLink to={`${base}/scan`} icon="camera" fullWidth>
        QRを読み取る
      </ButtonLink>
      <ButtonLink to={`${base}/complete`} variant="secondary" size="md" fullWidth>
        完了コードを入力
      </ButtonLink>
    </>
  )
}

/** 受け渡し完了。出品者には確定した売上を出す（PAY-2。金額はサーバーの値） */
function CompletedView({ transaction }: { transaction: Transaction }) {
  return (
    <>
      <section className={styles.done}>
        <span className={styles.check}>
          <Icon name="check" size={26} strokeWidth={2} />
        </span>
        <h2 className={styles.statusTitle}>受け渡しが完了しました</h2>
        {transaction.role === 'seller' && transaction.sellerProceeds !== null ? (
          <p className={styles.text}>
            売上 <b className={styles.num}>{formatPrice(transaction.sellerProceeds)}</b>{' '}
            が確定しました。
          </p>
        ) : (
          <p className={styles.text}>代金が出品者に支払われました。</p>
        )}
        {transaction.completedAt && (
          <p className={styles.muted}>{formatShortDateTime(transaction.completedAt)} 完了</p>
        )}
      </section>
      <section className={styles.card}>
        <Summary transaction={transaction} />
        <DealSteps status="completed" />
      </section>
      {transaction.role === 'seller' && (
        <ButtonLink to="/mypage/sales" variant="secondary" size="md" fullWidth>
          売上管理を見る
        </ButtonLink>
      )}
    </>
  )
}
