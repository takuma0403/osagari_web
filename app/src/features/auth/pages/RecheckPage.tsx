import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { Button } from '../../../components/Button/Button'
import { CodeInput } from '../../../components/CodeInput/CodeInput'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { resendRecheckCode, verifyRecheck, type LoginChallenge } from '../api'
import { AuthFormLayout } from '../components/AuthFormLayout/AuthFormLayout'
import { useCodeEntry } from '../hooks/useCodeEntry'
import { isRecheckState, type RecheckState } from '../navigation'
import { useSession } from '../session'
import styles from './RecheckPage.module.css'

/** 再確認の理由ごとの説明（AUTHN-3）。文言は Recheck.dc.html の reason ごとの表示と同じにする */
const reasonText: Record<LoginChallenge['reason'], string> = {
  code_expired:
    'パスワードは確認できました。前回メールで確認してから30日たったため、大学のメールに届いたコードを入力してください。',
  new_device:
    'パスワードは確認できました。はじめて使う端末からのログインのため、大学のメールに届いたコードを入力してください。次回からこの端末では不要です（30日ごとの確認を除く）。',
}

/**
 * ログイン時の本人確認（AUTH-1, AUTHN-3）。デザインは Recheck.dc.html。
 * 最後のコード確認から30日たったか、はじめての端末のとき、ログイン画面から来る。
 * コードの確認と再送はログインチャレンジで行い、パスワードは送り直さない。
 * 確認できたらログイン状態にし、GuestOnly が開こうとしていたページ（なければ商品一覧）へ送る。
 */
export function RecheckPage() {
  const { state } = useLocation()
  if (!isRecheckState(state)) return <Navigate to="/login" replace />
  // 送り直したら作り直し、入力と残り時間を新しいコードの分から始める
  return <RecheckForm key={state.sentAt} recheck={state} />
}

function RecheckForm({ recheck }: { recheck: RecheckState }) {
  const navigate = useNavigate()
  const { signIn } = useSession()
  const { challenge, sentAt } = recheck
  const { inputProps, canSubmit, submit, failure } = useCodeEntry({
    sentAt,
    verify: async (code) => signIn(await verifyRecheck(challenge.challengeToken, code)),
    resend: async () => {
      await resendRecheckCode(challenge.challengeToken)
      const next: RecheckState = { ...recheck, sentAt: Date.now() }
      navigate('.', { state: next, replace: true })
    },
  })
  // チャレンジ（15分）が切れたら、パスワードの入力からやり直す
  const challengeExpired = failure?.code === 'invalid_login_challenge'

  return (
    <AuthFormLayout
      backTo="/login"
      icon={
        <span className={styles.badge}>
          <Icon name="shield" size={26} />
        </span>
      }
      title="本人確認をお願いします"
      lead={reasonText[challenge.reason]}
    >
      <div className={styles.destination}>
        <div className={styles.address}>
          <span className={styles.addressLabel}>コードの送信先</span>
          <span className={styles.email}>{challenge.email}</span>
        </div>
        <Link to="/login" className={styles.switch}>
          別のアカウント
        </Link>
      </div>
      <form className={styles.form} onSubmit={submit} noValidate>
        <CodeInput {...inputProps} label="メールに届いた6桁のコード" showLabel />
        {challengeExpired && (
          <p className={styles.restart}>
            <Link to="/login">ログイン画面に戻る</Link>
          </p>
        )}
        <Button type="submit" fullWidth disabled={!canSubmit || challengeExpired}>
          確認してログイン
        </Button>
      </form>
      <InfoBox tone="warning" icon="alert" className={styles.note}>
        卒業などで大学のメールが受け取れなくなると、この確認ができないためログインできません。メールが届かないときは、迷惑メールフォルダも確認してください。
      </InfoBox>
    </AuthFormLayout>
  )
}
