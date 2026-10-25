import { useState } from 'react'
import { isApiError } from '../../../api/client'
import { cancelTransaction, withdrawCancelRequest, type Transaction } from '../api'

const unexpectedMessage = '送信できませんでした。もう一度お試しください'

/** none：どちらも申請していない／requested：自分が申請中／received：相手の申請が届いている */
export type CancelSide = 'none' | 'requested' | 'received'

/** 自分と相手の同意日時から、キャンセルの申請がどうなっているかを決める（DEAL-4、ADR 0026） */
export function cancelSideOf(transaction: Transaction): CancelSide {
  const [mine, theirs] =
    transaction.role === 'buyer'
      ? [transaction.buyerCancelAgreedAt, transaction.sellerCancelAgreedAt]
      : [transaction.sellerCancelAgreedAt, transaction.buyerCancelAgreedAt]
  if (mine) return 'requested'
  if (theirs) return 'received'
  return 'none'
}

/**
 * 取引のキャンセルの申請・同意・取り下げ（DEAL-4、ADR 0026）。
 * 相手の申請が届いていたら、シートを一度だけ自動で開く（閉じた後は案内から開き直す）。
 * 成功したら応答の取引で表示を更新し、409（相手が先に完了・取り下げたなど）なら取引を取り直す。
 */
export function useCancelRequest(
  transaction: Transaction,
  { replace, reload }: { replace: (transaction: Transaction) => void; reload: () => void },
) {
  const side = cancelSideOf(transaction)
  const theirs =
    transaction.role === 'buyer'
      ? transaction.sellerCancelAgreedAt
      : transaction.buyerCancelAgreedAt
  const [requestOpen, setRequestOpen] = useState(false)
  // 閉じた「同意を求められた」シートの申請日時。相手が取り下げて申請し直したら、また開く
  const [dismissed, setDismissed] = useState<string | null>(null)
  const [reopened, setReopened] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const sheetOpen =
    side === 'none' ? requestOpen : side === 'received' && (reopened || theirs !== dismissed)

  const send = async (action: (transactionId: string) => Promise<Transaction>) => {
    setSubmitting(true)
    setError(null)
    try {
      replace(await action(transaction.id))
      setRequestOpen(false)
    } catch (err) {
      setError(isApiError(err) ? err.message : unexpectedMessage)
      if (isApiError(err) && err.code === 'invalid_status_transition') reload()
    } finally {
      setSubmitting(false)
    }
  }

  const open = () => {
    setError(null)
    if (side === 'received') setReopened(true)
    else setRequestOpen(true)
  }

  const close = () => {
    setError(null)
    setRequestOpen(false)
    setReopened(false)
    setDismissed(theirs)
  }

  return {
    side,
    sheetOpen,
    open,
    close,
    /** 申請する・同意する（同じ操作） */
    submit: () => send(cancelTransaction),
    withdraw: () => send(withdrawCancelRequest),
    submitting,
    error,
  }
}
