import { useState } from 'react'
import { Button } from '../../../../components/Button/Button'
import { ConfirmDialog } from '../../../../components/ConfirmDialog/ConfirmDialog'
import { DefinitionList } from '../../../../components/DefinitionList/DefinitionList'
import { Icon } from '../../../../components/Icon/Icon'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import { TextField } from '../../../../components/TextField/TextField'
import { formatDate, formatShortDateTime } from '../../../../lib/format'
import { normalizeStudentId, STUDENT_ID_LENGTH } from '../../../../lib/studentId'
import type { AdminStudentCard, StudentCardView } from '../../api'
import { departmentLabel } from '../../departmentLabel'
import type { CardImageState, ReviewDecision } from '../../hooks/useStudentCardReview'
import styles from './CardReviewPanel.module.css'

type Props = {
  card: AdminStudentCard
  image: CardImageState
  /** 画像の閲覧URLを発行し直す（閲覧がもう1件記録される） */
  onReloadImage: () => void
  /** 閲覧記録（新しい順）。読み込み中は undefined */
  views?: StudentCardView[]
  viewsError?: string | null
  /** 送信中の操作 */
  deciding: ReviewDecision | null
  decisionError?: string | null
  onDecide: (decision: ReviewDecision) => void
}

const confirmTexts: Record<
  ReviewDecision,
  { title: string; body: (studentNumber: string) => string; label: string }
> = {
  approve: {
    title: '学生証を承認しますか？',
    body: (n) => `${n} の人が出品できるようになります。`,
    label: '承認する',
  },
  reject: {
    title: '学生証を差し戻しますか？',
    body: (n) => `${n} の人は学生証の画面で差し戻しを確認し、撮り直して再提出します。`,
    label: '差し戻す',
  },
}

/**
 * 1件の学生証の審査（AdminCard、ADMIN, CARD）。画像と本人の情報を並べ、学生証に書かれた番号を入力して
 * メールの学籍番号と照合する。一致したときだけ承認でき、承認・差し戻しは確認してから行う。
 * 入力した番号は学生証ごとに消すため、利用側で key={card.id} を付けて描画する。
 */
export function CardReviewPanel({
  card,
  image,
  onReloadImage,
  views,
  viewsError,
  deciding,
  decisionError,
  onDecide,
}: Props) {
  const [cardNumber, setCardNumber] = useState('')
  const [confirming, setConfirming] = useState<ReviewDecision | null>(null)
  const entered = cardNumber.length === STUDENT_ID_LENGTH
  const matched = entered && cardNumber === card.studentNumber
  const confirm = confirming && confirmTexts[confirming]

  return (
    <section aria-label="審査" className={styles.panel}>
      <div className={styles.top}>
        <CardImage studentNumber={card.studentNumber} image={image} onReload={onReloadImage} />
        <DefinitionList
          className={styles.facts}
          items={[
            {
              term: 'メールの学籍番号',
              description: <span className={styles.number}>{card.studentNumber}</span>,
            },
            { term: '推定した学部', description: departmentLabel(card.departmentName) },
            { term: '入学年度', description: `${card.admissionYear}年度` },
            { term: '有効期限', description: formatDate(card.expiresOn) },
          ]}
        />
      </div>

      <div className={styles.check}>
        <TextField
          label="学生証に書かれた番号"
          size="md"
          numeric
          inputMode="numeric"
          autoComplete="off"
          value={cardNumber}
          onChange={(e) => setCardNumber(normalizeStudentId(e.target.value))}
          inputClassName={styles.cardNumber}
          className={styles.cardNumberField}
        />
        <p role="status" className={styles.match}>
          {entered &&
            (matched ? (
              <span className={styles.matched}>
                <Icon name="check" size={18} strokeWidth={2.4} />
                メールの学籍番号と一致
              </span>
            ) : (
              <span className={styles.mismatched}>
                <Icon name="alert" size={18} />
                メールの学籍番号と一致しません
              </span>
            ))}
        </p>
      </div>

      {decisionError && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {decisionError}
        </InfoBox>
      )}

      <div className={styles.actions}>
        <Button
          size="sm"
          disabled={!matched || deciding !== null}
          onClick={() => setConfirming('approve')}
        >
          {deciding === 'approve' ? '承認しています…' : '承認する'}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={deciding !== null}
          onClick={() => setConfirming('reject')}
        >
          {deciding === 'reject' ? '差し戻しています…' : '差し戻す'}
        </Button>
        <span className={styles.viewNote}>
          この画像の閲覧は記録されます
          {views?.[0] &&
            `（${formatShortDateTime(views[0].viewedAt)} · ${views[0].adminStudentNumber}）`}
        </span>
      </div>

      <ViewLog views={views} error={viewsError} />

      <ConfirmDialog
        open={confirm !== null && confirm !== undefined}
        title={confirm ? confirm.title : ''}
        confirmLabel={confirm ? confirm.label : ''}
        cancelLabel="やめる"
        onCancel={() => setConfirming(null)}
        onConfirm={() => {
          if (confirming) onDecide(confirming)
          setConfirming(null)
        }}
      >
        {confirm && <p>{confirm.body(card.studentNumber)}</p>}
      </ConfirmDialog>
    </section>
  )
}

function CardImage({
  studentNumber,
  image,
  onReload,
}: {
  studentNumber: string
  image: CardImageState
  onReload: () => void
}) {
  // 署名付きURLは15分で切れるので、表示に失敗したら発行し直せるようにする
  const [brokenUrl, setBrokenUrl] = useState<string | null>(null)
  const failed =
    image.status === 'error'
      ? image.message
      : image.status === 'ready' && brokenUrl === image.url
        ? '画像を表示できませんでした。URLの期限が切れた可能性があります'
        : null

  return (
    <div className={styles.image}>
      {failed ? (
        <div className={styles.imageMessage} role="alert">
          <p>{failed}</p>
          <Button variant="text" size="sm" onClick={onReload}>
            もう一度表示する
          </Button>
        </div>
      ) : image.status === 'ready' ? (
        <img
          src={image.url}
          alt={`学生証の画像（${studentNumber}）`}
          className={styles.photo}
          onError={() => setBrokenUrl(image.url)}
        />
      ) : (
        <p className={styles.imageMessage} role="status">
          学生証の画像を読み込み中…
        </p>
      )}
    </div>
  )
}

function ViewLog({ views, error }: { views?: StudentCardView[]; error?: string | null }) {
  if (error) {
    return (
      <p className={styles.viewNote} role="alert">
        閲覧記録を読み込めませんでした：{error}
      </p>
    )
  }
  if (!views) return null
  return (
    <details className={styles.log}>
      <summary className={styles.logSummary}>閲覧記録を見る（{views.length}件）</summary>
      <ol className={styles.logList}>
        {views.map((v, i) => (
          // 同じ時刻・同じ管理者の閲覧もありうるので、並び順で区別する
          <li key={i}>
            {formatShortDateTime(v.viewedAt)} ·{' '}
            <span className={styles.number}>{v.adminStudentNumber}</span>
          </li>
        ))}
      </ol>
    </details>
  )
}
