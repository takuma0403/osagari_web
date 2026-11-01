import type { ReactNode } from 'react'
import { Badge } from '../../../../components/Badge/Badge'
import { Icon } from '../../../../components/Icon/Icon'
import { formatShortDateTime } from '../../../../lib/format'
import type { StudentCardState } from '../../api'
import styles from './CardReviewSteps.module.css'

/** done：済み／current：いまの段階／rejected：差し戻し／upcoming：まだ */
type StepState = 'done' | 'current' | 'rejected' | 'upcoming'

type Step = {
  label: string
  state: StepState
  aside?: ReactNode
}

const stateText: Record<StepState, string> = {
  done: '完了',
  current: '',
  rejected: '差し戻し',
  upcoming: '未完了',
}

function time(iso: string | null) {
  return iso && <span className={styles.time}>{formatShortDateTime(iso)}</span>
}

/** 審査状況から3段の表示を決める。審査中の表示は StudentCard.dc.html、それ以外はデザインにないため同じ形で状態を変える */
function stepsOf({ status, submittedAt, reviewedAt }: StudentCardState): Step[] {
  const uploaded = status !== 'none'
  const upload: Step = {
    label: '画像をアップロード',
    state: uploaded ? 'done' : 'current',
    aside: uploaded && time(submittedAt),
  }
  const review: Step =
    status === 'pending'
      ? {
          label: '管理者の確認待ち',
          state: 'current',
          aside: <Badge tone="warning">審査中</Badge>,
        }
      : status === 'rejected'
        ? {
            label: '管理者の確認',
            state: 'rejected',
            aside: <Badge tone="danger">差し戻し</Badge>,
          }
        : status === 'approved'
          ? { label: '管理者の確認', state: 'done', aside: time(reviewedAt) }
          : { label: '管理者の確認', state: 'upcoming' }
  const sell: Step = {
    label: '出品できるようになる',
    state: status === 'approved' ? 'done' : 'upcoming',
  }
  return [upload, review, sell]
}

function Marker({ state }: { state: StepState }) {
  return (
    <span className={[styles.marker, styles[state]].join(' ')} aria-hidden="true">
      {state === 'done' ? (
        <Icon name="check" size={14} strokeWidth={3} />
      ) : (
        state !== 'upcoming' && <span className={styles.dot} />
      )}
    </span>
  )
}

type Props = {
  card: StudentCardState
}

/** 学生証の登録から出品できるようになるまでの3段の進み具合（CARD） */
export function CardReviewSteps({ card }: Props) {
  return (
    <ol className={styles.steps} aria-label="審査の進み具合">
      {stepsOf(card).map(({ label, state, aside }) => (
        <li
          key={label}
          className={[styles.step, styles[`${state}Step`]].join(' ')}
          aria-current={state === 'current' ? 'step' : undefined}
        >
          <Marker state={state} />
          <span className={styles.label}>
            {label}
            {/* 印の意味を読み上げで伝える。差し戻しはバッジの文字で伝わる */}
            {state !== 'current' && state !== 'rejected' && (
              <span className="visually-hidden">（{stateText[state]}）</span>
            )}
          </span>
          {aside && <span className={styles.aside}>{aside}</span>}
        </li>
      ))}
    </ol>
  )
}
