import { useId, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { isApiError } from '../../../api/client'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Badge } from '../../../components/Badge/Badge'
import { Button } from '../../../components/Button/Button'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { formatPrice } from '../../../lib/format'
import { campusOptions, categoryTint, labelOf } from '../../../lib/itemOptions'
import type { PublicItem } from '../../items/api'
import { useItem } from '../../items/hooks/useItem'
import { isSoldOut, itemPath } from '../../items/itemDisplay'
import { createPurchaseRequest, type PaymentMethod } from '../api'
import { DealItemSummary } from '../components/DealItemSummary/DealItemSummary'
import styles from './RequestPage.module.css'

const unexpectedMessage = '購入希望を送れませんでした。もう一度お試しください'

/** 商品詳細に戻ったときにトーストを出すための location.state */
export type RequestSentState = { requestSent: true }

const paymentOptions: { value: PaymentMethod; label: string; note?: string }[] = [
  // フェーズ1は決済がモックのため、カードは表示用の番号（PAY）
  { value: 'card', label: 'クレジットカード', note: '•••• 4242' },
  { value: 'paypay', label: 'PayPay' },
]

/** 購入希望を出せない理由。出せるなら null */
function blockedReason(item: PublicItem): string | null {
  if (item.isOwn) return '自分の出品には購入希望を出せません'
  if (isSoldOut(item)) return 'この商品は売り切れです'
  if (item.myRequest) return 'この商品にはすでに購入希望を出しています'
  return null
}

/**
 * 購入希望・決済（DEAL-1, PAY、Request）。/items/:shareId/request で開く。
 * 支払い方法を選んで購入希望を送ると、枠を確保する（フェーズ1はモック）。代金は受け渡しのQR読み取りで出品者に支払われる。
 * 送れたら商品詳細へ戻り、トーストで知らせる。
 */
export function RequestPage() {
  const { shareId = '' } = useParams()
  const navigate = useNavigate()
  const { state, reload } = useItem(shareId)
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('card')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const radioName = useId()

  const header = (
    <AppHeader
      title="購入希望を出す"
      backTo={itemPath(shareId)}
      right={<Badge tone="warning">決済はモック</Badge>}
    />
  )

  if (state.status !== 'ready') {
    return (
      <div className={styles.page}>
        {header}
        <div className={styles.status}>
          {state.status === 'loading' && <p role="status">読み込み中…</p>}
          {state.status === 'notFound' && (
            <>
              <h2 className={styles.statusTitle}>商品が見つかりません</h2>
              <p className={styles.muted}>取り下げられたか、公開が止まっている可能性があります。</p>
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

  const { item } = state
  const blocked = blockedReason(item)

  const submit = async () => {
    setSubmitting(true)
    setError(null)
    try {
      await createPurchaseRequest(item.shareId, paymentMethod)
      const sent: RequestSentState = { requestSent: true }
      navigate(itemPath(item.shareId), { replace: true, state: sent })
    } catch (err) {
      setError(isApiError(err) ? err.message : unexpectedMessage)
      setSubmitting(false)
    }
  }

  return (
    <div className={styles.page}>
      {header}
      <main className={styles.body}>
        <section className={styles.card}>
          <DealItemSummary
            title={item.title}
            price={item.price}
            imageUrl={item.images[0]?.url}
            tint={categoryTint[item.category]}
            note={`${labelOf(campusOptions, item.campus)}キャンパスで受け渡し`}
          />
        </section>
        <fieldset className={styles.payment} disabled={submitting || !!blocked}>
          <legend className={styles.legend}>支払い方法</legend>
          {paymentOptions.map((option) => (
            <label key={option.value} className={styles.option}>
              <input
                type="radio"
                name={radioName}
                value={option.value}
                checked={paymentMethod === option.value}
                onChange={() => setPaymentMethod(option.value)}
                className={styles.radio}
              />
              {option.label}
              {option.note && <span className={styles.optionNote}>{option.note}</span>}
            </label>
          ))}
        </fieldset>
        <div className={styles.qrNote}>
          <Icon name="qr" size={20} className={styles.qrIcon} />
          <span>代金は、商品を受け取る際、QRコードを読み取った時点で出品者に支払われます。</span>
        </div>
        {blocked && (
          <InfoBox tone="warning" icon="info">
            {blocked}
          </InfoBox>
        )}
        {error && (
          <InfoBox tone="danger" icon="alert" role="alert">
            {error}
          </InfoBox>
        )}
      </main>
      <div className={styles.actions}>
        <div className={styles.total}>
          <span>お支払い予定額</span>
          <span className={styles.totalPrice}>{formatPrice(item.price)}</span>
        </div>
        <Button fullWidth onClick={submit} disabled={submitting || !!blocked}>
          {submitting ? '送信中…' : '購入希望を送る'}
        </Button>
      </div>
    </div>
  )
}
