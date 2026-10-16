import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { isApiError } from '../../../api/client'
import { Button } from '../../../components/Button/Button'
import { CodeInput } from '../../../components/CodeInput/CodeInput'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { formatCountdown } from '../../../lib/format'
import { toUniversityEmail } from '../../../lib/studentId'
import {
  CODE_RESEND_INTERVAL_SECONDS,
  CODE_TTL_SECONDS,
  sendSignupCode,
  verifySignupCode,
} from '../api'
import { AuthFormLayout } from '../components/AuthFormLayout/AuthFormLayout'
import { useCountdown } from '../hooks/useCountdown'
import { isSignupCodeState, type SignupCodeState } from '../navigation'
import styles from './CodePage.module.css'

const CODE_LENGTH = 6

/**
 * コード入力の手続きごとの違い。パスワード再設定・再確認（#14）はここに足す。
 * 文言は Code.dc.html の mode ごとの表示と同じにする。
 */
const flows = {
  signup: {
    /** 前の画面（メールアドレスの入力）。state がないときもここへ戻す */
    backTo: '/signup',
    next: '/signup/password',
    step: { current: 2, total: 3 },
    title: '認証コードを入力',
    send: sendSignupCode,
    /** 成功したら一時トークンと推定結果（SignupPasswordState）を次の画面へ渡す */
    verify: verifySignupCode,
    note: (
      <>
        すでに登録済みのアドレスには、コードの代わりに「登録済みです」という案内が届きます。その場合は
        <Link to="/login">ログイン</Link>
        してください。
      </>
    ),
  },
} as const

type Props = {
  flow: keyof typeof flows
}

/**
 * 認証コードの入力（AUTH-1, AUTHN-1, AUTHN-2, AUTHN-6）。デザインは Code.dc.html。
 * 有効期限（10分）と再送の待ち時間（1分）は、前の画面でコードを送った時刻から数える。
 */
export function CodePage({ flow }: Props) {
  const { state } = useLocation()
  const config = flows[flow]
  if (!isSignupCodeState(state)) return <Navigate to={config.backTo} replace />
  // 送り直したら作り直し、入力と残り時間を新しいコードの分から始める
  return <CodeForm key={state.sentAt} flow={flow} sent={state} />
}

function CodeForm({ flow, sent }: Props & { sent: SignupCodeState }) {
  const config = flows[flow]
  const navigate = useNavigate()
  const { studentId, sentAt } = sent
  const [code, setCode] = useState('')
  const [error, setError] = useState<string>()
  const [verifying, setVerifying] = useState(false)
  const [resending, setResending] = useState(false)

  const expiresIn = useCountdown(sentAt + CODE_TTL_SECONDS * 1000)
  const resendIn = useCountdown(sentAt + CODE_RESEND_INTERVAL_SECONDS * 1000)

  const handleChange = (value: string) => {
    setCode(value)
    setError(undefined)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (code.length !== CODE_LENGTH || verifying) return
    setVerifying(true)
    setError(undefined)
    try {
      const next = await config.verify(studentId, code)
      navigate(config.next, { state: next, replace: true })
    } catch (err) {
      if (!isApiError(err)) throw err
      // 違うコード（残り回数入り）・期限切れ・使えなくなったコードは、サーバーの文言をそのまま出す（AUTHN-6）
      setError(err.message)
      setCode('')
      setVerifying(false)
    }
  }

  const handleResend = async () => {
    setResending(true)
    setError(undefined)
    try {
      await config.send(studentId)
      // 送った時刻を state に残し、再読み込みしても有効期限と待ち時間が続くようにする
      const next: SignupCodeState = { studentId, sentAt: Date.now() }
      navigate('.', { state: next, replace: true })
    } catch (err) {
      if (!isApiError(err)) throw err
      // 送信数の制限（429）など。断られたときは前のコードがそのまま使える（AUTHN-6）
      setError(err.message)
    } finally {
      setResending(false)
    }
  }

  return (
    <AuthFormLayout
      backTo={config.backTo}
      step={config.step}
      title={config.title}
      lead={
        <>
          <span className={styles.email}>{toUniversityEmail(studentId)}</span>{' '}
          に送った6桁のコードを入力してください。
        </>
      }
    >
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <CodeInput
          value={code}
          onChange={handleChange}
          error={error}
          expiresIn={formatCountdown(expiresIn)}
          resend={{
            label:
              resendIn > 0 ? `コードを再送（${formatCountdown(resendIn)} 後）` : 'コードを再送',
            onClick: handleResend,
            disabled: resendIn > 0 || resending,
          }}
          disabled={verifying}
        />
        <Button type="submit" fullWidth disabled={code.length !== CODE_LENGTH || verifying}>
          確認する
        </Button>
      </form>
      <InfoBox tone="neutral">
        メールが届かないときは、迷惑メールフォルダも確認してください。
        {config.note}
      </InfoBox>
    </AuthFormLayout>
  )
}
