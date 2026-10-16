import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation } from 'react-router'
import { isApiError, type ApiError } from '../../../api/client'
import { Button } from '../../../components/Button/Button'
import { DefinitionList } from '../../../components/DefinitionList/DefinitionList'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { TextField } from '../../../components/TextField/TextField'
import { formatDate } from '../../../lib/format'
import { completePasswordReset, completeSignup, type StudentEstimate } from '../api'
import { AuthFormLayout } from '../components/AuthFormLayout/AuthFormLayout'
import { isPasswordResetState, isSignupPasswordState } from '../navigation'
import { useSession } from '../session'
import styles from './PasswordPage.module.css'

/** パスワードの長さ（AUTHN-5）。最終的な検証はサーバーが行う */
export const PASSWORD_MIN_LENGTH = 10
export const PASSWORD_MAX_LENGTH = 128

/** 文字数で数える（AUTHN-5）。絵文字などのサロゲートペアも1文字にする */
const countChars = (text: string) => Array.from(text).length

/** 手続きごとの違い。文言は Password.dc.html の mode ごとの表示と同じにする */
const flows = {
  signup: {
    /** 最初の画面。コードは使用済みなので、戻るとやり直しになる */
    start: '/signup',
    title: 'パスワードを決める',
    cta: '登録してはじめる',
    isState: isSignupPasswordState,
    complete: completeSignup,
  },
  passwordReset: {
    start: '/password-reset',
    title: '新しいパスワード',
    cta: 'パスワードを変更する',
    isState: isPasswordResetState,
    complete: completePasswordReset,
  },
} as const

type Props = {
  flow: keyof typeof flows
}

/**
 * パスワードの設定（AUTH-1, AUTHN-2, AUTHN-4, AUTHN-5）。デザインは Password.dc.html。
 * 新規登録ではアカウントを作り、再設定ではほかの端末のセッションを無効にして、どちらもそのままログイン状態にする。
 * 行き先は GuestOnly が決める（商品一覧）。
 */
export function PasswordPage({ flow }: Props) {
  const config = flows[flow]
  const { state } = useLocation()
  if (!config.isState(state)) return <Navigate to={config.start} replace />
  return (
    <PasswordForm
      flow={flow}
      setupToken={state.setupToken}
      estimate={'estimate' in state ? state.estimate : undefined}
    />
  )
}

function PasswordForm({
  flow,
  setupToken,
  estimate,
}: Props & { setupToken: string; estimate?: StudentEstimate }) {
  const config = flows[flow]
  const { signIn } = useSession()
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [mismatch, setMismatch] = useState(false)
  const [failure, setFailure] = useState<ApiError>()
  const [submitting, setSubmitting] = useState(false)

  const length = countChars(password)
  const tooLong = length > PASSWORD_MAX_LENGTH
  const canSubmit = length >= PASSWORD_MIN_LENGTH && !tooLong && confirmation !== '' && !submitting

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    if (password !== confirmation) {
      setMismatch(true)
      return
    }
    setSubmitting(true)
    setFailure(undefined)
    try {
      signIn(await config.complete(setupToken, password))
    } catch (err) {
      if (!isApiError(err)) throw err
      // 一時トークンの期限切れ（15分）・停止中・失効などは、サーバーの文言をそのまま出す
      setFailure(err)
      setSubmitting(false)
    }
  }

  return (
    <AuthFormLayout
      backTo={config.start}
      step={{ current: 3, total: 3 }}
      title={config.title}
      lead="次回からは、メールアドレスとこのパスワードでログインします。"
    >
      {estimate && (
        <DefinitionList
          size="sm"
          boxed
          className={styles.summary}
          items={[
            {
              term: '学籍番号',
              description: <span className={styles.number}>{estimate.studentNumber}</span>,
            },
            { term: '学部・学科', description: estimate.departmentName ?? '学部不明' },
            { term: '有効期限', description: formatDate(estimate.expiresOn) },
          ]}
        />
      )}
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <TextField
          label="パスワード"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => {
            setPassword(e.target.value)
            setMismatch(false)
          }}
          error={
            tooLong ? `${PASSWORD_MAX_LENGTH}文字以下にしてください（${length}文字）` : undefined
          }
          hint={
            tooLong ? undefined : length >= PASSWORD_MIN_LENGTH ? (
              <span className={styles.lengthOk}>
                <Icon name="check" size={16} strokeWidth={2.4} />
                {PASSWORD_MIN_LENGTH}文字以上（{length}文字）
              </span>
            ) : (
              `${PASSWORD_MIN_LENGTH}文字以上（${length}文字）`
            )
          }
        />
        <TextField
          label="パスワード（確認）"
          type="password"
          autoComplete="new-password"
          value={confirmation}
          onChange={(e) => {
            setConfirmation(e.target.value)
            setMismatch(false)
          }}
          error={mismatch ? 'パスワードが一致しません' : undefined}
        />
        {flow === 'passwordReset' && (
          <InfoBox tone="warning">変更すると、ほかの端末ではログアウトされます。</InfoBox>
        )}
        {failure && (
          <InfoBox tone="danger" icon="alert" role="alert">
            {failure.message}
            {failure.code === 'invalid_setup_token' && (
              <>
                <br />
                <Link to={config.start}>最初からやり直す</Link>
              </>
            )}
          </InfoBox>
        )}
        <Button type="submit" fullWidth className={styles.submit} disabled={!canSubmit}>
          {config.cta}
        </Button>
      </form>
    </AuthFormLayout>
  )
}
