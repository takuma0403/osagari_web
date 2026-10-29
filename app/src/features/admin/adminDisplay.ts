import type { BadgeTone } from '../../components/Badge/Badge'
import type { AdminItem, AdminTransaction, AdminUser, AdminUserFilter } from './api'

/** 表のバッジに出す状態の名前と色 */
export type StatusDisplay = { label: string; tone: BadgeTone }

/** ユーザーの状態（AdminUsers のバッジ。AUTH-3） */
export const userStatusDisplay: Record<AdminUser['status'], StatusDisplay> = {
  active: { label: '有効', tone: 'primary' },
  suspended: { label: '停止', tone: 'danger' },
  expired: { label: '失効', tone: 'neutral' },
  deleted: { label: '削除', tone: 'neutral' },
}

/** ユーザーの状態の絞り込み（AdminUsers）。並びはボタンに出す順 */
export const userFilterOptions: { value: AdminUserFilter; label: string }[] = [
  { value: 'all', label: 'すべて' },
  { value: 'active', label: '有効' },
  { value: 'suspended', label: '停止' },
  { value: 'expired', label: '失効' },
  { value: 'withdrawal_requested', label: '退学申請' },
]

/** 有効期限を変えられる・退学により削除できるのは有効・停止のユーザーだけ（ADR 0034） */
export const isUserManageable = (user: AdminUser) =>
  user.status === 'active' || user.status === 'suspended'

/** 商品のステータス（AdminDeals の出品のバッジ） */
export const itemStatusDisplay: Record<AdminItem['status'], StatusDisplay> = {
  draft: { label: '下書き', tone: 'neutral' },
  published: { label: '公開中', tone: 'primary' },
  requested: { label: '購入希望あり', tone: 'warning' },
  in_deal: { label: '取引中', tone: 'primary' },
  completed: { label: '取引完了', tone: 'neutral' },
  withdrawn: { label: '取り下げ', tone: 'neutral' },
  suspended: { label: '停止', tone: 'danger' },
}

/** 出品を停止できるのは下書き・公開中・購入希望あり・取引中だけ（ADMIN, ITEM-1、ADR 0034） */
export const isItemSuspendable = (item: AdminItem) =>
  item.status === 'draft' ||
  item.status === 'published' ||
  item.status === 'requested' ||
  item.status === 'in_deal'

const transactionStatusDisplays: Record<AdminTransaction['status'], StatusDisplay> = {
  requested: { label: '購入希望', tone: 'warning' },
  rejected: { label: '却下', tone: 'neutral' },
  expired: { label: '期限切れ', tone: 'neutral' },
  in_deal: { label: '取引中', tone: 'primary' },
  cancelled: { label: 'キャンセル', tone: 'neutral' },
  completed: { label: '完了', tone: 'neutral' },
}

/** 取引のステータス（AdminDeals の取引のバッジ）。取引中で片方がキャンセルに同意していれば「キャンセル申請中」（DEAL-4） */
export function transactionStatusDisplay(tx: AdminTransaction): StatusDisplay {
  if (tx.status === 'in_deal' && (tx.buyerCancelAgreedAt || tx.sellerCancelAgreedAt))
    return { label: 'キャンセル申請中', tone: 'warning' }
  return transactionStatusDisplays[tx.status]
}

/** 取引の詳細の「キャンセル申請」の文（DEAL-4）。どちらが申請し、相手が同意したか */
export function cancelRequestText(tx: AdminTransaction): string {
  const buyer = tx.buyerCancelAgreedAt !== null
  const seller = tx.sellerCancelAgreedAt !== null
  if (buyer && seller) return '双方が同意'
  if (buyer) return '購入者が申請（出品者は未同意）'
  if (seller) return '出品者が申請（購入者は未同意）'
  return 'なし'
}

/** 管理者がキャンセル・返金（モック）できるのは取引中の取引だけ（DEAL-4、ADR 0034） */
export const isTransactionCancellable = (tx: AdminTransaction) => tx.status === 'in_deal'

/** 手数料率の入力（%）。0〜100、小数点以下2桁まで（AdminSettings） */
const FEE_RATE_INPUT = /^\d{1,3}(\.\d{1,2})?$/

/**
 * 手数料率の入力（`5`・`5.25` の %）をベーシスポイントにする（PAY-2）。範囲外・形が不正なら null。
 * 最終的な範囲の検証はサーバーが行う
 */
export function parseFeeRatePercent(input: string): number | null {
  const value = input.trim()
  if (!FEE_RATE_INPUT.test(value)) return null
  const [whole, fraction = ''] = value.split('.')
  // 小数の誤差を避けるため、文字列のまま 0.01% 単位の整数にする
  const basisPoints = Number(whole) * 100 + Number(fraction.padEnd(2, '0'))
  return basisPoints <= 10000 ? basisPoints : null
}
