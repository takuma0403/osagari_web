import type { BadgeTone } from '../../components/Badge/Badge'
import { formatMonthDay, formatRemaining } from '../../lib/format'
import type { TransactionListEntry, TransactionRole } from './api'

const MINUTE_MS = 60 * 1000

/** 取引詳細（/deals/:dealId） */
export const dealPath = (transactionId: string) => `/deals/${encodeURIComponent(transactionId)}`

/** 取引一覧の1行に出すもの */
export type DealRowDisplay = {
  /** 行の行き先 */
  to: string
  /** 状態のバッジ */
  status: string
  tone: BadgeTone
  /** バッジの右の補足 */
  note: string
}

/** 終了した取引の補足（購入側の受け渡し日を除く） */
const endedNotes = {
  cancelled: { status: 'キャンセル', note: '双方の同意で取り消し' },
  expired: { status: '期限切れ', note: '枠を解放しました' },
  rejected: { status: '却下', note: '枠を解放しました' },
} as const

/**
 * 取引一覧（DealList）の1行の状態・補足・行き先を決める（DEAL-1・3・4、ADR 0028）。
 * 出品側の承認待ちの購入希望は商品ごとの1行で、購入希望の一覧（/sell/:itemId/requests）へ進む。
 * 購入側の購入希望は承認期限までの残りを出す。期限を過ぎても、定期処理で期限切れになるまでは承認待ちのまま。
 */
export function dealRowDisplay(
  entry: TransactionListEntry,
  role: TransactionRole,
  now: number = Date.now(),
): DealRowDisplay {
  const to = entry.transactionId ? dealPath(entry.transactionId) : ''
  switch (entry.status) {
    case 'requested':
      if (role === 'seller') {
        return {
          to: `/sell/${encodeURIComponent(entry.itemId ?? '')}/requests`,
          status: `購入希望 ${entry.requestCount ?? 0}件`,
          tone: 'warning',
          note: '相手を選んでください',
        }
      }
      return {
        to,
        status: '承認待ち',
        tone: 'warning',
        note:
          // 残り1分未満（「あと0分」になる）と期限を過ぎたものは「まもなく期限切れ」にする
          entry.approvalDeadline && Date.parse(entry.approvalDeadline) - now >= MINUTE_MS
            ? `あと${formatRemaining(entry.approvalDeadline, now)}で期限切れ`
            : 'まもなく期限切れ',
      }
    case 'in_deal':
      return {
        to,
        status: '取引中',
        tone: 'primary',
        note: role === 'seller' ? '受け渡しでQRを表示' : '受け渡しの日時をTeamsで相談',
      }
    case 'completed':
      return {
        to,
        status: '完了',
        tone: 'neutral',
        note:
          role === 'seller'
            ? '売上に計上済み'
            : entry.completedAt
              ? `${formatMonthDay(entry.completedAt)} 受け渡し`
              : '受け渡し済み',
      }
    default:
      return { to, tone: 'neutral', ...endedNotes[entry.status] }
  }
}
