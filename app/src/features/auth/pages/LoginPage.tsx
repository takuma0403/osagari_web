import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { isApiError } from '../../../api/client'
import { AppLogo } from '../../../components/AppLogo/AppLogo'
import { Button, ButtonLink } from '../../../components/Button/Button'
import { EmailField } from '../../../components/EmailField/EmailField'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { TextField } from '../../../components/TextField/TextField'
import { STUDENT_ID_LENGTH } from '../../../lib/studentId'
import { login } from '../api'
import styles from './LoginPage.module.css'

type Failure = { locked: boolean; message: string }

/**
 * ログイン（AUTH-1, AUTHN-3）。デザインは Login.dc.html。
 * パスワードが一致して再確認が必要なとき（202）は、チャレンジを持って /recheck へ進む。
 * 失敗の文言はサーバーの message をそのまま出す（登録の有無を区別しない文言はサーバーが返す。AUTHN-3）。
 */
export function LoginPage() {
  const navigate = useNavigate()
  const [studentId, setStudentId] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [failure, setFailure] = useState<Failure | null>(null)

  const complete = studentId.length === STUDENT_ID_LENGTH && password !== ''
  const locked = failure?.locked ?? false

  // 入力を直したら前回の失敗の表示を消す（ロック中の表示も、別のアドレスなら当てはまらないため）
  const edit =
    <T,>(set: (value: T) => void) =>
    (value: T) => {
      set(value)
      setFailure(null)
    }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!complete || submitting || locked) return
    setSubmitting(true)
    setFailure(null)
    try {
      const result = await login(studentId, password)
      if (result.kind === 'loggedIn') {
        navigate('/', { replace: true })
      } else {
        navigate('/recheck', { state: { challenge: result.challenge } })
      }
    } catch (err) {
      if (!isApiError(err)) throw err
      // AUTHN-6: パスワードの入力ミスが続いて止まっている
      setFailure({ locked: err.code === 'too_many_requests', message: err.message })
      setSubmitting(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.intro}>
        <AppLogo size={72} className={styles.logo} />
        <h1 className={styles.catch}>
          佐賀大生どうしで、
          <br />
          お下がりを回そう。
        </h1>
        <p className={styles.lead}>佐賀大学の学生だけが使えるフリマです。</p>
      </div>

      {failure && (
        <InfoBox tone="danger" icon={locked ? 'lock' : 'alert'} role="alert">
          {failure.message}
          {locked && (
            <>
              <br />
              思い出せない場合は、パスワードを再設定してください。
            </>
          )}
        </InfoBox>
      )}

      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <div className={styles.fields}>
          <EmailField value={studentId} onChange={edit(setStudentId)} />
          <div className={styles.password}>
            <TextField
              label="パスワード"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => edit(setPassword)(e.target.value)}
            />
            <Link to="/password-reset" className={styles.forgot}>
              パスワードを忘れた場合
            </Link>
          </div>
        </div>
        <Button type="submit" fullWidth disabled={!complete || submitting || locked}>
          ログイン
        </Button>
      </form>

      <div className={styles.divider}>はじめての方</div>
      <ButtonLink to="/signup" variant="secondary" fullWidth>
        新規登録
      </ButtonLink>

      <p className={styles.footer}>
        <Link to="/terms">利用規約</Link> · <Link to="/privacy">プライバシーポリシー</Link>
      </p>
    </div>
  )
}
