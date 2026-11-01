import { useEffect } from 'react'
import { Button } from '../../../components/Button/Button'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { Toast } from '../../../components/Toast/Toast'
import { useAdminLayout } from '../../../layouts/AdminLayout/adminLayoutContext'
import { CardQueue } from '../components/CardQueue/CardQueue'
import { CardReviewPanel } from '../components/CardReviewPanel/CardReviewPanel'
import { useStudentCardReview } from '../hooks/useStudentCardReview'
import styles from './AdminCardPage.module.css'

/**
 * 管理：学生証審査（ADMIN, CARD）。デザインは AdminCard.dc.html。
 * 審査待ちを古い順に並べ、選んだ学生証の画像（閲覧は記録される）と本人の情報を見て承認・差し戻しする。
 * 読み込み中・0件・エラーの表示はデザインにないため、既存の部品で表す。
 */
export function AdminCardPage() {
  const review = useStudentCardReview()
  const { cards, selected } = review
  const { setMenuBadge } = useAdminLayout()

  useEffect(() => {
    setMenuBadge('studentCards', cards?.length)
  }, [cards, setMenuBadge])

  return (
    <>
      <div className={styles.heading}>
        <h1 className={styles.title}>学生証審査</h1>
        {cards && <span className={styles.summary}>審査待ち {cards.length}件 · 古い順</span>}
      </div>
      {cards ? (
        cards.length === 0 ? (
          <InfoBox tone="neutral" icon="check">
            審査待ちの学生証はありません。
          </InfoBox>
        ) : (
          <div className={styles.columns}>
            <CardQueue cards={cards} selectedId={selected?.id ?? null} onSelect={review.select} />
            {selected && (
              <CardReviewPanel
                key={selected.id}
                card={selected}
                image={review.image}
                onReloadImage={() => review.select(selected.id)}
                views={review.views}
                viewsError={review.viewsError}
                deciding={review.deciding}
                decisionError={review.decisionError}
                onDecide={review.decide}
              />
            )}
          </div>
        )
      ) : review.loadError ? (
        <InfoBox tone="danger" icon="alert" role="alert">
          <p>{review.loadError}</p>
          <Button variant="text" size="sm" onClick={review.reload}>
            もう一度読み込む
          </Button>
        </InfoBox>
      ) : (
        <p className={styles.loading} role="status">
          読み込み中…
        </p>
      )}
      <Toast
        message={review.notice?.message}
        icon={review.notice?.tone === 'alert' ? 'alert' : 'check'}
      />
    </>
  )
}
