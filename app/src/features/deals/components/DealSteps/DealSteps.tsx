import type { components } from '../../../../api/schema.gen'
import styles from './DealSteps.module.css'

type ItemStatus = components['schemas']['ItemStatus']

const steps: { status: ItemStatus; label: string }[] = [
  { status: 'published', label: '公開中' },
  { status: 'requested', label: '購入希望あり' },
  { status: 'in_deal', label: '取引中' },
  { status: 'completed', label: '取引完了' },
]

type Props = {
  /** 商品のステータス。下書き・取り下げ・停止はどの段階にも当たらない */
  status: ItemStatus
}

/** 商品が取引のどの段階にあるか（DEAL-1：公開中 → 購入希望あり → 取引中 → 取引完了）。いまの段階までの棒を primary にする */
export function DealSteps({ status }: Props) {
  const current = steps.findIndex((step) => step.status === status)
  return (
    <ol className={styles.steps} aria-label="取引の段階">
      {steps.map((step, i) => (
        <li
          key={step.status}
          className={[styles.step, i <= current && styles.done].filter(Boolean).join(' ')}
          aria-current={i === current ? 'step' : undefined}
        >
          <span className={styles.bar} aria-hidden="true" />
          <span className={styles.label}>{step.label}</span>
        </li>
      ))}
    </ol>
  )
}
