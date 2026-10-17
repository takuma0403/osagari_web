import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { isApiError } from '../../../api/client'
import { Button } from '../../../components/Button/Button'
import { EmailField } from '../../../components/EmailField/EmailField'
import { Icon } from '../../../components/Icon/Icon'
import { formatDate } from '../../../lib/format'
import { STUDENT_ID_LENGTH } from '../../../lib/studentId'
import {
  estimateStudent,
  sendPasswordResetCode,
  sendSignupCode,
  type StudentEstimate,
} from '../api'
import { AuthFormLayout } from '../components/AuthFormLayout/AuthFormLayout'
import type { CodeSentState } from '../navigation'
import styles from './SignUpPage.module.css'

/** 推定結果の文（AUTH-2, AUTH-4）。表にない学部コードは「学部不明」として登録される */
function estimateText(e: StudentEstimate) {
  const department = e.departmentName ?? '学部不明'
  return `${department}・${e.admissionYear}年度入学として登録されます（有効期限 ${formatDate(e.expiresOn)}）`
}

/** 手続きごとの違い。文言は SignUp.dc.html の mode ごとの表示と同じにする */
const flows = {
  signup: {
    title: '新規登録',
    lead: '大学のメールアドレスに6桁の認証コードを送ります。コードを確認したあと、パスワードを決めます。',
    send: sendSignupCode,
    next: '/signup/code',
    /** 学部・入学年度・有効期限の推定と「登録済みの方は ログイン」を出す */
    signup: true,
  },
  passwordReset: {
    title: 'パスワードの再設定',
    lead: '登録したメールアドレスに6桁の認証コードを送ります。本人確認のあと、新しいパスワードを決めます。',
    send: sendPasswordResetCode,
    next: '/password-reset/code',
    signup: false,
  },
} as const

type Props = {
  flow?: keyof typeof flows
}

/**
 * 新規登録・パスワード再設定の1歩目：メールアドレスの入力（AUTH-1, AUTHN-2, AUTHN-4）。デザインは SignUp.dc.html。
 * 新規登録では、8桁そろったら学部・入学年度・有効期限の推定を出す。コードを送ったらコード入力へ進む。
 * 登録済みかどうかにかかわらず同じ応答・同じ遷移になる（登録の有無を判別させない）。
 */
export function SignUpPage({ flow = 'signup' }: Props) {
  const config = flows[flow]
  const navigate = useNavigate()
  const [studentId, setStudentId] = useState('')
  const [estimate, setEstimate] = useState<StudentEstimate | null>(null)
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  const complete = studentId.length === STUDENT_ID_LENGTH

  useEffect(() => {
    if (!complete || !config.signup) return
    const controller = new AbortController()
    estimateStudent(studentId, controller.signal)
      .then(setEstimate)
      // 推定は表示のためだけなので、失敗しても登録は続けられる（形式の誤りは送信時に伝える）
      .catch(() => {})
    return () => controller.abort()
  }, [complete, studentId, config.signup])

  const handleChange = (value: string) => {
    setStudentId(value)
    setEstimate(null)
    setError(undefined)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!complete || submitting) return
    setSubmitting(true)
    setError(undefined)
    try {
      await config.send(studentId)
      const state: CodeSentState = { studentId, sentAt: Date.now() }
      navigate(config.next, { state })
    } catch (err) {
      if (!isApiError(err)) throw err
      setError(err.message)
      setSubmitting(false)
    }
  }

  return (
    <AuthFormLayout
      backTo="/login"
      step={{ current: 1, total: 3 }}
      title={config.title}
      lead={config.lead}
    >
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <EmailField
          value={studentId}
          onChange={handleChange}
          error={error}
          hint={
            <span className={styles.hint}>
              {estimate && (
                <span className={styles.estimate}>
                  <Icon name="check" size={16} strokeWidth={2} className={styles.estimateIcon} />
                  {estimateText(estimate)}
                </span>
              )}
              学籍番号（数字8桁）を入力してください
            </span>
          }
        />
        <Button type="submit" fullWidth disabled={!complete || submitting}>
          認証コードを送る
        </Button>
      </form>
      {config.signup && (
        <p className={styles.footer}>
          登録済みの方は <Link to="/login">ログイン</Link>
        </p>
      )}
    </AuthFormLayout>
  )
}
