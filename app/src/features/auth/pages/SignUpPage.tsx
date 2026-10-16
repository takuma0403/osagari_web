import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'
import { isApiError } from '../../../api/client'
import { Button } from '../../../components/Button/Button'
import { EmailField } from '../../../components/EmailField/EmailField'
import { Icon } from '../../../components/Icon/Icon'
import { formatDate } from '../../../lib/format'
import { STUDENT_ID_LENGTH } from '../../../lib/studentId'
import { estimateStudent, sendSignupCode, type StudentEstimate } from '../api'
import { AuthFormLayout } from '../components/AuthFormLayout/AuthFormLayout'
import type { CodeSentState } from '../navigation'
import styles from './SignUpPage.module.css'

/** 推定結果の文（AUTH-2, AUTH-4）。表にない学部コードは「学部不明」として登録される */
function estimateText(e: StudentEstimate) {
  const department = e.departmentName ?? '学部不明'
  return `${department}・${e.admissionYear}年度入学として登録されます（有効期限 ${formatDate(e.expiresOn)}）`
}

/**
 * 新規登録の1歩目：メールアドレスの入力（AUTH-1, AUTHN-2）。デザインは SignUp.dc.html（新規登録）。
 * 8桁そろったら学部・入学年度・有効期限の推定を出し、コードを送ったら /signup/code へ進む。
 * 登録済みでも同じ応答・同じ遷移になる（登録の有無を判別させない）。
 */
export function SignUpPage() {
  const navigate = useNavigate()
  const [studentId, setStudentId] = useState('')
  const [estimate, setEstimate] = useState<StudentEstimate | null>(null)
  const [error, setError] = useState<string>()
  const [submitting, setSubmitting] = useState(false)

  const complete = studentId.length === STUDENT_ID_LENGTH

  useEffect(() => {
    if (!complete) return
    const controller = new AbortController()
    estimateStudent(studentId, controller.signal)
      .then(setEstimate)
      // 推定は表示のためだけなので、失敗しても登録は続けられる（形式の誤りは送信時に伝える）
      .catch(() => {})
    return () => controller.abort()
  }, [complete, studentId])

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
      await sendSignupCode(studentId)
      const state: CodeSentState = { studentId, sentAt: Date.now() }
      navigate('/signup/code', { state })
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
      title="新規登録"
      lead="大学のメールアドレスに6桁の認証コードを送ります。コードを確認したあと、パスワードを決めます。"
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
      <p className={styles.footer}>
        登録済みの方は <Link to="/login">ログイン</Link>
      </p>
    </AuthFormLayout>
  )
}
