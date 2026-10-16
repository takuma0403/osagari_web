import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { Button } from '../../../components/Button/Button'
import { CodeInput } from '../../../components/CodeInput/CodeInput'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { toUniversityEmail } from '../../../lib/studentId'
import { sendSignupCode, verifySignupCode } from '../api'
import { AuthFormLayout } from '../components/AuthFormLayout/AuthFormLayout'
import { useCodeEntry } from '../hooks/useCodeEntry'
import { isCodeSentState, type CodeSentState } from '../navigation'
import styles from './CodePage.module.css'

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
  if (!isCodeSentState(state)) return <Navigate to={config.backTo} replace />
  // 送り直したら作り直し、入力と残り時間を新しいコードの分から始める
  return <CodeForm key={state.sentAt} flow={flow} sent={state} />
}

function CodeForm({ flow, sent }: Props & { sent: CodeSentState }) {
  const config = flows[flow]
  const navigate = useNavigate()
  const { studentId, sentAt } = sent
  const { inputProps, canSubmit, submit } = useCodeEntry({
    sentAt,
    verify: async (code) => {
      const next = await config.verify(studentId, code)
      navigate(config.next, { state: next, replace: true })
    },
    resend: async () => {
      await config.send(studentId)
      // 送った時刻を state に残し、再読み込みしても有効期限と待ち時間が続くようにする
      const next: CodeSentState = { studentId, sentAt: Date.now() }
      navigate('.', { state: next, replace: true })
    },
  })

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
      <form className={styles.form} onSubmit={submit} noValidate>
        <CodeInput {...inputProps} />
        <Button type="submit" fullWidth disabled={!canSubmit}>
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
