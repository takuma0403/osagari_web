import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { isApiError } from '../../../api/client'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Button } from '../../../components/Button/Button'
import { CodeInput } from '../../../components/CodeInput/CodeInput'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { parseHandoverFragment } from '../../../lib/handoverLink'
import type { HandoverCredential } from '../api'
import { DealItemSummary } from '../components/DealItemSummary/DealItemSummary'
import { ReceiptConfirmDialog } from '../components/ReceiptConfirmDialog/ReceiptConfirmDialog'
import { useReceiptConfirm } from '../hooks/useReceiptConfirm'
import { useTransaction } from '../hooks/useTransaction'
import styles from './DealCompletePage.module.css'

const unexpectedMessage = '照合できませんでした。もう一度お試しください'
const invalidCodeMessage =
  'コードが違うか、有効期限が切れています。出品者に新しいコードを発行してもらってください'
const invalidLinkMessage =
  'この完了リンクは使えません。有効期限（24時間）が切れたか、すでに使われています。出品者に新しいリンクかコードを発行してもらってください'
const invalidQrMessage =
  'このQRコードは使えません。有効期限（5分）が切れたか、すでに使われています。出品者に「表示し直す」を押してもらってください'

const isInvalidToken = (err: unknown) => isApiError(err) && err.code === 'invalid_handover_token'

/**
 * 受け渡しの完了（DEAL-3、DealCompleteEnter）。購入者が完了リンクを開くか、6桁のコードを入力する。
 * 完了リンク（`#t=`）と、スマホ標準のカメラで読んだ受け渡しQR（`#q=`）はフラグメントでトークンを受け取り（ADR 0025）、
 * 読んだらすぐ URL から消して照合する。フラグメントがなければ6桁のコード（完了コード・QRの下の確認コード）を入力する。
 * どちらも照合できたら「商品を受け取りましたか？」を出し、「受け取った」で完了する。
 */
export function DealCompletePage() {
  const { dealId = '' } = useParams()
  const location = useLocation()
  const navigate = useNavigate()
  // フラグメントは最初の表示で一度だけ読む（このあと URL から消す）
  const [fromUrl] = useState(() => parseHandoverFragment(location.hash))
  const [linkState, setLinkState] = useState<'checking' | 'invalid' | 'checked'>('checking')
  const [code, setCode] = useState('')
  const [codeError, setCodeError] = useState<string | undefined>()
  const [verifying, setVerifying] = useState(false)
  const { state } = useTransaction(dealId)
  const receipt = useReceiptConfirm(dealId)
  const { verify } = receipt

  const base = `/deals/${encodeURIComponent(dealId)}`

  // トークンを履歴・画面共有に残さない
  useEffect(() => {
    if (location.hash) navigate({ pathname: location.pathname }, { replace: true })
  }, [location.hash, location.pathname, navigate])

  // 完了リンク・QR のトークンを照合する（使用済みにはしない）
  useEffect(() => {
    if (!fromUrl) return
    let cancelled = false
    verify({ kind: fromUrl.kind, secret: fromUrl.token }).then(
      () => {
        if (!cancelled) setLinkState('checked')
      },
      () => {
        if (!cancelled) setLinkState('invalid')
      },
    )
    return () => {
      cancelled = true
    }
  }, [fromUrl, verify])

  const submitCode = async () => {
    const credential: HandoverCredential = { kind: 'code', secret: code }
    setVerifying(true)
    setCodeError(undefined)
    try {
      await verify(credential)
    } catch (err) {
      setCodeError(
        isInvalidToken(err)
          ? invalidCodeMessage
          : isApiError(err)
            ? err.message
            : unexpectedMessage,
      )
    } finally {
      setVerifying(false)
    }
  }

  const header = <AppHeader title="受け渡しの完了" backTo={base} />

  if (state.status !== 'ready') {
    return (
      <div className={styles.page}>
        {header}
        <main className={styles.body}>
          {state.status === 'loading' && <p role="status">読み込み中…</p>}
          {state.status === 'notFound' && (
            <>
              <h2 className={styles.statusTitle}>取引が見つかりません</h2>
              <p className={styles.muted}>
                この取引の購入者としてログインしているときだけ、受け渡しを完了できます。
              </p>
              <Link to="/deals">取引一覧へ戻る</Link>
            </>
          )}
          {state.status === 'error' && (
            <InfoBox tone="danger" icon="alert" role="alert">
              {state.message}
            </InfoBox>
          )}
        </main>
      </div>
    )
  }

  const { transaction } = state
  const canComplete = transaction.role === 'buyer' && transaction.status === 'in_deal'

  return (
    <div className={styles.page}>
      {header}
      <main className={styles.body}>
        <section className={styles.item}>
          <DealItemSummary
            title={transaction.item.title}
            price={transaction.price}
            tint="textbook"
          />
          {transaction.counterpartStudentNumber && (
            <p className={styles.muted}>
              出品者の学籍番号{' '}
              <span className={styles.studentNumber}>{transaction.counterpartStudentNumber}</span>
            </p>
          )}
        </section>
        {!canComplete ? (
          <InfoBox tone="danger" icon="alert" role="alert">
            {transaction.status === 'completed'
              ? 'この取引は受け渡しが完了しています。'
              : '受け渡しを完了できるのは、取引中の取引の購入者だけです。'}{' '}
            <Link to={base}>取引詳細へ</Link>
          </InfoBox>
        ) : fromUrl ? (
          linkState === 'checking' ? (
            <p role="status">確認しています…</p>
          ) : linkState === 'invalid' ? (
            <InfoBox tone="danger" icon="alert" role="alert">
              {fromUrl.kind === 'qr' ? invalidQrMessage : invalidLinkMessage}
            </InfoBox>
          ) : (
            !receipt.open && (
              <Button
                fullWidth
                onClick={() => verify({ kind: fromUrl.kind, secret: fromUrl.token })}
              >
                受け渡しを完了する
              </Button>
            )
          )
        ) : (
          <section className={styles.codeCard}>
            <h2 className={styles.cardTitle}>完了コードを入力</h2>
            <p className={styles.muted}>
              出品者からTeamsで届いた6桁のコード、または受け渡しQRの下に表示された確認コードを入力してください。
            </p>
            <CodeInput
              value={code}
              onChange={(next) => {
                setCode(next)
                setCodeError(undefined)
              }}
              label="完了コード"
              error={codeError}
              disabled={verifying}
            />
            <Button fullWidth onClick={submitCode} disabled={code.length !== 6 || verifying}>
              次へ
            </Button>
          </section>
        )}
      </main>
      <ReceiptConfirmDialog
        open={receipt.open}
        price={transaction.price}
        onConfirm={receipt.confirm}
        onCancel={receipt.cancel}
        submitting={receipt.submitting}
        error={receipt.error}
      />
    </div>
  )
}
