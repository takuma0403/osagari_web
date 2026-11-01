import { useState, type ComponentProps, type FormEvent } from 'react'
import { isApiError, type ApiError } from '../../../api/client'
import type { CodeInput } from '../../../components/CodeInput/CodeInput'
import { formatCountdown } from '../../../lib/format'
import { CODE_RESEND_INTERVAL_SECONDS, CODE_TTL_SECONDS } from '../api'
import { useCountdown } from './useCountdown'

export const CODE_LENGTH = 6

type Options = {
  /** コードを送った時刻（ミリ秒）。有効期限（10分）と再送の待ち時間（1分）はここから数える */
  sentAt: number
  /** コードを確認する。成功したら呼び出し側が次へ進む。失敗は ApiError を投げる */
  verify: (code: string) => Promise<void>
  /** コードを送り直す。成功したら呼び出し側が sentAt を新しくする */
  resend: () => Promise<void>
}

/**
 * 認証コードの入力の処理（AUTHN-1, AUTHN-6）。新規登録・パスワード再設定（CodePage）と再確認（RecheckPage）で使う。
 * 入力・確認・再送・残り時間を持ち、CodeInput に渡す props と送信の処理を返す。
 * 送り直したら呼び出し側で key を変えて作り直し、入力と残り時間を新しいコードの分から始める。
 */
export function useCodeEntry({ sentAt, verify, resend }: Options) {
  const [code, setCode] = useState('')
  const [failure, setFailure] = useState<ApiError>()
  const [verifying, setVerifying] = useState(false)
  const [resending, setResending] = useState(false)

  const expiresIn = useCountdown(sentAt + CODE_TTL_SECONDS * 1000)
  const resendIn = useCountdown(sentAt + CODE_RESEND_INTERVAL_SECONDS * 1000)

  const canSubmit = code.length === CODE_LENGTH && !verifying

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    setVerifying(true)
    setFailure(undefined)
    try {
      await verify(code)
    } catch (err) {
      if (!isApiError(err)) throw err
      // 違うコード（残り回数入り）・期限切れ・使えなくなったコードは、サーバーの文言をそのまま出す（AUTHN-6）
      setFailure(err)
      setCode('')
      setVerifying(false)
    }
  }

  const handleResend = async () => {
    setResending(true)
    setFailure(undefined)
    try {
      await resend()
    } catch (err) {
      if (!isApiError(err)) throw err
      // 送信数の制限（429）など。断られたときは前のコードがそのまま使える（AUTHN-6）
      setFailure(err)
    } finally {
      setResending(false)
    }
  }

  const inputProps = {
    value: code,
    onChange: (value: string) => {
      setCode(value)
      setFailure(undefined)
    },
    error: failure?.message,
    expiresIn: formatCountdown(expiresIn),
    resend: {
      label: resendIn > 0 ? `コードを再送（${formatCountdown(resendIn)} 後）` : 'コードを再送',
      onClick: handleResend,
      disabled: resendIn > 0 || resending,
    },
    disabled: verifying,
  } satisfies ComponentProps<typeof CodeInput>

  return { inputProps, canSubmit, submit, failure }
}
