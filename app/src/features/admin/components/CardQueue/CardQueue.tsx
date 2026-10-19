import { formatShortDateTime } from '../../../../lib/format'
import type { AdminStudentCard } from '../../api'
import { departmentLabel } from '../../departmentLabel'
import styles from './CardQueue.module.css'

type Props = {
  cards: AdminStudentCard[]
  selectedId: string | null
  onSelect: (id: string) => void
}

/** 審査待ちの学生証の一覧（AdminCard）。行を押すとその学生証を審査パネルに出す */
export function CardQueue({ cards, selectedId, onSelect }: Props) {
  return (
    <section aria-label="審査待ち一覧" className={styles.queue}>
      {cards.map((card) => (
        <button
          key={card.id}
          type="button"
          aria-pressed={card.id === selectedId}
          className={styles.row}
          onClick={() => onSelect(card.id)}
        >
          <span className={styles.who}>
            <span className={styles.number}>{card.studentNumber}</span>
            <span className={styles.muted}>{departmentLabel(card.departmentName)}</span>
          </span>
          <span className={styles.muted}>
            <span className="visually-hidden">提出 </span>
            {formatShortDateTime(card.submittedAt)}
          </span>
        </button>
      ))}
    </section>
  )
}
