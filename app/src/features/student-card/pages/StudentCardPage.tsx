import { useRef, type ChangeEvent } from 'react'
import { AppHeader } from '../../../components/AppHeader/AppHeader'
import { Button, ButtonLink } from '../../../components/Button/Button'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { TwoColumn } from '../../../components/TwoColumn/TwoColumn'
import { PC_MEDIA_QUERY, useMediaQuery } from '../../../lib/useMediaQuery'
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
  // PC 幅では画像を選ぶボタンを左カラム（審査の段階の下）に出す。スマホは画像の下のまま（DS-1）。
  // CSS の order ではなく DOM の位置を変え、Tab の順を見た目にそろえる
  const pc = useMediaQuery(PC_MEDIA_QUERY)
  const shownPreview = card?.status !== 'approved' ? preview : null
  const actions = card && (
    <CardActions card={card} submitting={submitting} submitError={submitError} onSelect={submit} />
  )

  return (
    <div className={styles.page}>
      <AppHeader title="学生証の登録" backTo="/mypage" />
      <div className={styles.body}>
        {/* PC 幅では審査の段階（左）と画像（右）の2カラム（DS-1） */}
        <TwoColumn
          start={
            <>
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
                  {pc && <div className={styles.image}>{actions}</div>}
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
            </>
          }
          end={
            <>
              {card && (shownPreview || !pc) && (
                <section className={styles.image}>
                  {shownPreview && <CardPreview preview={shownPreview} />}
                  {!pc && actions}
                </section>
              )}
              <InfoBox tone="info" icon="lock">
                画像は暗号化して保存し、見られるのは管理者だけです。トラブル時に当事者を特定するため、アカウントの失効・削除後も3年間保持し、その後自動で削除します。
              </InfoBox>
            </>
          }
        />
      </div>
    </div>
  )
}

/** 登録した画像のプレビュー */
function CardPreview({ preview }: { preview: string }) {
  return (
    <>
      <h2 className={styles.sectionTitle}>登録した画像</h2>
      <div className={styles.frame}>
        <img src={preview} alt="登録した学生証の画像" className={styles.preview} />
      </div>
    </>
  )
}

type CardActionsProps = {
  card: StudentCardState
  submitting: boolean
  submitError: string | null
  onSelect: (file: File) => void
}

/** 画像の選択（撮り直し）。承認済みなら撮り直せないので出品へ案内する */
function CardActions({ card, submitting, submitError, onSelect }: CardActionsProps) {
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
    <>
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
    </>
  )
}
