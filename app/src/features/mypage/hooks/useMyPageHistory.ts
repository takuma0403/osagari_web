import { useCallback, useEffect, useState } from 'react'
import { isApiError } from '../../../api/client'
import { listTransactions, type TransactionListEntry } from '../../deals/api'
import { listOwnItems, type OwnItemSummary } from '../api'
import { isPurchased } from '../myPageDisplay'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

/** 終了した取引を読むときの1回の件数（API の上限） */
const ENDED_PAGE_LIMIT = 50

function messageOf(err: unknown): string {
  return isApiError(err) ? err.message : unexpectedMessage
}

export type MyPageHistoryState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | {
      status: 'ready'
      /** 自分の出品（最後に更新した順） */
      items: OwnItemSummary[]
      /** 購入した商品（取引中・完了。進行中、終了の順） */
      purchases: TransactionListEntry[]
    }

/** 購入側の終了した取引を、続きがなくなるまで読む */
async function listAllEnded(signal: AbortSignal) {
  const entries: TransactionListEntry[] = []
  let cursor: string | undefined
  do {
    const page = await listTransactions('buyer', 'ended', cursor, signal, ENDED_PAGE_LIMIT)
    entries.push(...page.entries)
    cursor = page.nextCursor ?? undefined
  } while (cursor)
  return entries
}

/** 読み込んだ結果と、それが何回目の読み込みのものか */
type Result = { key: number; state: MyPageHistoryState }

/**
 * マイページ（AUTH-3、MyPage）の出品と購入の履歴を並行して読み込む（ADR 0029・0030）。
 * 購入は購入側の取引のうち成立したもの（取引中・完了）だけにし、件数をタブに出すため終了した取引は最後まで読む。
 */
export function useMyPageHistory() {
  const [reloadCount, setReloadCount] = useState(0)
  const [result, setResult] = useState<Result>()

  // いまの読み込みの結果がまだなければ読み込み中
  const state: MyPageHistoryState =
    result?.key === reloadCount ? result.state : { status: 'loading' }

  useEffect(() => {
    const controller = new AbortController()
    Promise.all([
      listOwnItems(controller.signal),
      listTransactions('buyer', 'active', undefined, controller.signal),
      listAllEnded(controller.signal),
    ]).then(
      ([own, active, ended]) =>
        setResult({
          key: reloadCount,
          state: {
            status: 'ready',
            items: own.items,
            purchases: [...active.entries, ...ended].filter(isPurchased),
          },
        }),
      (err) => {
        if (!controller.signal.aborted)
          setResult({ key: reloadCount, state: { status: 'error', message: messageOf(err) } })
      },
    )
    return () => controller.abort()
  }, [reloadCount])

  const reload = useCallback(() => setReloadCount((n) => n + 1), [])

  return { state, reload }
}
