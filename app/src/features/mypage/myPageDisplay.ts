import type { BadgeTone } from '../../components/Badge/Badge'
import type { CurrentUser } from '../auth/api'
import { dealRowDisplay } from '../deals/dealListDisplay'
import type { TransactionListEntry } from '../deals/api'
import { itemPath } from '../items/itemDisplay'
import type { OwnItemSummary } from './api'

/** マイページの履歴の1行に出すもの */
export type HistoryRowDisplay = {
  /** 行の行き先 */
  to: string
  /** 右のバッジ */
  status: string
  tone: BadgeTone
}

const sellPath = (itemId: string) => `/sell/${encodeURIComponent(itemId)}`

/**
 * 「出品した商品」の1行の状態と行き先（AUTH-3, ITEM、ADR 0030）。
 * 編集できる下書き・取り下げと、編集できない停止中は出品の編集画面へ、購入希望ありは購入希望の一覧へ、
 * 公開中・取引中・売却済みは商品ページへ進む。
 */
export function ownItemDisplay(item: OwnItemSummary): HistoryRowDisplay {
  const page = itemPath(item.shareId)
  switch (item.status) {
    case 'requested':
      return {
        to: `${sellPath(item.id)}/requests`,
        status: `購入希望 ${item.requestCount}件`,
        tone: 'warning',
      }
    case 'draft':
      return { to: sellPath(item.id), status: '下書き', tone: 'neutral' }
    case 'withdrawn':
      return { to: sellPath(item.id), status: '取り下げ', tone: 'neutral' }
    case 'suspended':
      return { to: sellPath(item.id), status: '停止中', tone: 'danger' }
    case 'published':
      return { to: page, status: '公開中', tone: 'primary' }
    case 'in_deal':
      return { to: page, status: '取引中', tone: 'primary' }
    case 'completed':
      return { to: page, status: '売却済み', tone: 'neutral' }
  }
}

/** 「購入した商品」に出す取引（成立したもの。ADR 0030） */
export const isPurchased = (entry: TransactionListEntry) =>
  entry.status === 'in_deal' || entry.status === 'completed'

/** 「購入した商品」の1行の状態と行き先。取引一覧の購入側と同じバッジで、取引詳細へ進む */
export function purchaseDisplay(entry: TransactionListEntry): HistoryRowDisplay {
  const { to, status, tone } = dealRowDisplay(entry, 'buyer')
  return { to, status, tone }
}

/** 設定の「学生証の登録」の右に出す審査状況（CARD） */
export const sellerStatusLabels: Record<CurrentUser['sellerStatus'], string> = {
  none: '未登録',
  pending: '審査中',
  approved: '承認済み',
  rejected: '差し戻し',
}
