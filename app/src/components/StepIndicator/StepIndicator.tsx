import styles from './StepIndicator.module.css'

type Props = {
  current: number
  total: number
}

/** 登録・再設定の手順の位置（`STEP n / 3`、DS-3） */
export function StepIndicator({ current, total }: Props) {
  return (
    <p className={styles.step}>
      STEP {current} / {total}
    </p>
  )
}
