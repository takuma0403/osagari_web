import { useRef, type ChangeEvent } from 'react'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Button, ButtonLink } from '../../../components/Button/Button'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import type { StudentCardState } from '../api'
import { CardReviewSteps } from '../components/CardReviewSteps/CardReviewSteps'
import { useStudentCard } from '../hooks/useStudentCard'
import styles from './StudentCardPage.module.css'

/**
 * 学生証の登録と審査状況（CARD, STORAGE）。デザインは StudentCard.dc.html（審査中の表示）。
 * 未登録・差し戻し・承認済みの表示はデザインにないため、同じ構成で既存の部品を使って表す。
 * 承認済みになったら撮り直せない（サーバーが 409 を返す）ので、出品へ案内する。
 */
export function StudentCardPage() {
  const { card, loadError, reload, submit, submitting, submitError, preview } = useStudentCard()

  return (
    <div className={styles.page}>
      <AppHeader title="学生証の登録" backTo="/mypage" />
      <div className={styles.body}>
        <p className={styles.lead}>
          出品するには学生証の登録が必要です。管理者が学籍番号を確認したら出品できるようになります。
        </p>
        {card ? (
          <>
            <CardReviewSteps card={card} />
            {card.status === 'rejected' && (
              <InfoBox tone="danger" icon="alert">
                学生証を確認できませんでした。撮り直して提出してください。
              </InfoBox>
            )}
            <CardImage
              card={card}
              preview={preview}
              submitting={submitting}
              submitError={submitError}
              onSelect={submit}
            />
          </>
        ) : loadError ? (
          <InfoBox tone="danger" icon="alert" role="alert">
            <p>{loadError}</p>
            <Button variant="text" size="sm" onClick={reload}>
              もう一度読み込む
            </Button>
          </InfoBox>
        ) : (
          <p className={styles.loading} role="status">
            読み込み中…
          </p>
        )}
        <InfoBox tone="info" icon="lock">
          画像は暗号化して保存し、見られるのは管理者だけです。トラブル時に当事者を特定するため、アカウントの失効・削除後も3年間保持し、その後自動で削除します。
        </InfoBox>
      </div>
    </div>
  )
}

type CardImageProps = {
  card: StudentCardState
  preview: string | null
  submitting: boolean
  submitError: string | null
  onSelect: (file: File) => void
}

/** 登録した画像のプレビューと、画像の選択（撮り直し） */
function CardImage({ card, preview, submitting, submitError, onSelect }: CardImageProps) {
  const inputRef = useRef<HTMLInputElement>(null)

  if (card.status === 'approved') {
    return (
      <ButtonLink to="/sell" fullWidth>
        出品する
      </ButtonLink>
    )
  }

  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    // 同じ画像を選び直しても change が起きるように空にする
    event.target.value = ''
    if (file) onSelect(file)
  }

  const first = card.status === 'none'
  return (
    <section className={styles.image}>
      {preview && (
        <>
          <h2 className={styles.sectionTitle}>登録した画像</h2>
          <div className={styles.frame}>
            <img src={preview} alt="登録した学生証の画像" className={styles.preview} />
          </div>
        </>
      )}
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        aria-label="学生証の画像"
        className="visually-hidden"
        tabIndex={-1}
        onChange={handleChange}
      />
      <Button
        variant={first ? 'primary' : 'ghost'}
        size={first ? 'lg' : 'sm'}
        icon={first ? 'camera' : undefined}
        fullWidth
        disabled={submitting}
        onClick={() => inputRef.current?.click()}
      >
        {submitting ? 'アップロード中…' : first ? '学生証の画像を選ぶ' : '撮り直す'}
      </Button>
      {submitError && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {submitError}
        </InfoBox>
      )}
    </section>
  )
}
