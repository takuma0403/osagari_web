import { useRef, useState } from 'react'
import { Link, useParams } from 'react-router'
import { isApiError } from '../../../api/client'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { ButtonLink } from '../../../components/Button/Button'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { cameraScanner, type StartScan } from '../../../lib/cameraScan'
import { parseHandoverQr } from '../../../lib/handoverLink'
import { isCameraDenied, useCameraScan } from '../../../lib/useCameraScan'
import { ReceiptConfirmDialog } from '../components/ReceiptConfirmDialog/ReceiptConfirmDialog'
import { useReceiptConfirm } from '../hooks/useReceiptConfirm'
import { useTransaction } from '../hooks/useTransaction'
import styles from './DealScanPage.module.css'

/** 受け渡しQR（QR_CODE）だけを読む */
const startQrScan = cameraScanner(['QR_CODE'])

function cameraErrorMessage(err: unknown): string {
  if (isCameraDenied(err))
    return 'カメラの使用が許可されていません。ブラウザの設定で許可するか、6桁のコードを入力してください'
  return 'カメラを起動できませんでした。6桁のコードを入力してください'
}

type Props = {
  /** 読み取りの開始（テスト・Storybook でカメラの代わりを渡す） */
  startScan?: StartScan
}

/**
 * 受け渡しQRの読み取り（DEAL-3、DealScan）。購入者が出品者の表示した QR を読み取る。
 * 読み取ったトークンを照合し、使えれば「商品を受け取りましたか？」を出す。「受け取った」で完了する。
 * 別の取引の QR や使えない QR は画面にエラーを出し、読み取りを続ける（同じ QR は何度も照合しない）。
 */
export function DealScanPage({ startScan = startQrScan }: Props) {
  const { dealId = '' } = useParams()
  const { state } = useTransaction(dealId)
  const receipt = useReceiptConfirm(dealId)
  const [notice, setNotice] = useState<string | null>(null)
  // 照合中、または照合して使えなかったトークン。同じ QR を写し続けても照合し直さない
  const checking = useRef(false)
  const rejected = useRef(new Set<string>())

  const base = `/deals/${encodeURIComponent(dealId)}`
  const transaction = state.status === 'ready' ? state.transaction : null
  const canScan = transaction?.role === 'buyer' && transaction.status === 'in_deal'

  const { videoRef, error: cameraError } = useCameraScan({
    active: canScan && !receipt.open,
    startScan,
    onText: (text) => {
      if (checking.current) return
      const token = parseHandoverQr(text, dealId)
      if (!token) {
        setNotice('この取引の受け渡しQRではありません。出品者の画面のQRを写してください')
        return
      }
      if (rejected.current.has(token)) return
      checking.current = true
      receipt
        .verify({ kind: 'qr', secret: token })
        .then(
          () => setNotice(null),
          (err: unknown) => {
            rejected.current.add(token)
            setNotice(
              isApiError(err) && err.code === 'invalid_handover_token'
                ? 'このQRコードは使えません（有効期限切れか使用済み）。出品者に「表示し直す」を押してもらってください'
                : isApiError(err)
                  ? err.message
                  : '照合できませんでした。もう一度お試しください',
            )
          },
        )
        .finally(() => {
          checking.current = false
        })
    },
  })

  return (
    <div className={styles.page}>
      <AppHeader title="QRを読み取る" backTo={base} closable bordered={false} tone="dark" />
      <main className={styles.body}>
        {state.status === 'loading' && <p role="status">読み込み中…</p>}
        {state.status === 'notFound' && (
          <InfoBox tone="danger" icon="alert" role="alert">
            取引が見つかりません。<Link to="/deals">取引一覧へ戻る</Link>
          </InfoBox>
        )}
        {state.status === 'error' && (
          <InfoBox tone="danger" icon="alert" role="alert">
            {state.message}
          </InfoBox>
        )}
        {transaction && !canScan && (
          <InfoBox tone="danger" icon="alert" role="alert">
            {transaction.status === 'completed'
              ? 'この取引は受け渡しが完了しています。'
              : 'QRを読み取れるのは、取引中の取引の購入者だけです。'}{' '}
            <Link to={base}>取引詳細へ</Link>
          </InfoBox>
        )}
        {canScan && (
          <>
            <div className={styles.frame}>
              <video
                ref={videoRef}
                className={styles.video}
                muted
                playsInline
                aria-label="カメラの映像"
              />
            </div>
            {cameraError ? (
              <InfoBox tone="danger" icon="alert" role="alert">
                {cameraErrorMessage(cameraError)}
              </InfoBox>
            ) : (
              <p className={styles.hint}>出品者が表示したQRコードを枠に合わせてください</p>
            )}
            {notice && (
              <p role="alert" className={styles.notice}>
                {notice}
              </p>
            )}
            <ButtonLink
              to={`${base}/complete`}
              variant="ghost"
              size="sm"
              className={styles.codeLink}
            >
              6桁のコードを入力する
            </ButtonLink>
          </>
        )}
      </main>
      {transaction && (
        <ReceiptConfirmDialog
          open={receipt.open}
          price={transaction.price}
          itemTitle={transaction.item.title}
          onConfirm={receipt.confirm}
          onCancel={receipt.cancel}
          submitting={receipt.submitting}
          error={receipt.error}
          overlay="dark"
        />
      )}
    </div>
  )
}
