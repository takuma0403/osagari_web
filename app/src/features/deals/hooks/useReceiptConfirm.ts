import { useCallback, useState } from 'react'
import { useNavigate } from 'react-router'
import { isApiError } from '../../../api/client'
import { completeTransaction, verifyHandover, type HandoverCredential } from '../api'

const unexpectedMessage = '受け渡しを完了できませんでした。もう一度お試しください'

/**
 * 受け渡しの完了の流れ（DEAL-3）。読み取った・開いた・入力したトークンを照合し（使用済みにしない）、
 * 使えれば「商品を受け取りましたか？」を開く。「受け取った」で完了し、取引詳細（完了の表示）へ進む。
 * QRの読み取り（DealScan）と完了リンク・完了コード（DealCompleteEnter）で使う。
 */
export function useReceiptConfirm(dealId: string) {
  const navigate = useNavigate()
  const [credential, setCredential] = useState<HandoverCredential | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  /** 照合して、使えれば確認を開く。使えなければ ApiError（400 invalid_handover_token など）を投げる */
  const verify = useCallback(
    async (cred: HandoverCredential) => {
      await verifyHandover(dealId, cred)
      setError(null)
      setCredential(cred)
    },
    [dealId],
  )

  const confirm = async () => {
    if (!credential) return
    setSubmitting(true)
    setError(null)
    try {
      await completeTransaction(dealId, credential)
      navigate(`/deals/${encodeURIComponent(dealId)}`, { replace: true })
    } catch (err) {
      setError(isApiError(err) ? err.message : unexpectedMessage)
      setSubmitting(false)
    }
  }

  /** 「まだ受け取っていない」。トークンは使用済みになっていないので、あとで同じものを使える */
  const cancel = () => {
    setCredential(null)
    setError(null)
  }

  return { open: credential !== null, verify, confirm, cancel, submitting, error }
}
