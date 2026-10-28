import { Button } from '../../../../components/Button/Button'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import styles from './AdminLoadState.module.css'

type Props = {
  /** 読み込みに失敗したときの文。null なら読み込み中を出す */
  error: string | null
  onReload: () => void
}

/** 管理画面の一覧を読み込む前の表示。読み込み中か、失敗と読み直しのボタン（AdminCard と同じ見た目） */
export function AdminLoadState({ error, onReload }: Props) {
  return error ? (
    <InfoBox tone="danger" icon="alert" role="alert">
      <p>{error}</p>
      <Button variant="text" size="sm" onClick={onReload}>
        もう一度読み込む
      </Button>
    </InfoBox>
  ) : (
    <p className={styles.loading} role="status">
      読み込み中…
    </p>
  )
}
