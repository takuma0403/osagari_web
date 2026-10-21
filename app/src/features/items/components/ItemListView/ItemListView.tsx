import type { ReactNode } from 'react'
import { Button } from '../../../../components/Button/Button'
import { InfoBox } from '../../../../components/InfoBox/InfoBox'
import type { ItemSummary } from '../../api'
import type { ItemListState } from '../../hooks/useItemList'
import styles from './ItemListView.module.css'

type Props = {
  state: ItemListState
  /** 0件のときの文 */
  emptyMessage: string
  onRetry: () => void
  onLoadMore: () => void
  /** 読み込んだ商品の並べ方（Main は ItemCard の2列、Search は ItemRow の縦並び） */
  children: (items: ItemSummary[]) => ReactNode
}

/**
 * 商品の一覧の読み込み中・エラー・0件・続きの読み込み（BROWSE）をまとめて出す。
 * 続きは末尾の「もっと見る」で読み込む。
 */
export function ItemListView({ state, emptyMessage, onRetry, onLoadMore, children }: Props) {
  if (state.status === 'loading') {
    return (
      <p className={styles.message} role="status">
        読み込み中…
      </p>
    )
  }
  if (state.status === 'error') {
    return (
      <InfoBox tone="danger" icon="alert" role="alert">
        <p>{state.message}</p>
        <Button variant="text" size="sm" onClick={onRetry}>
          もう一度読み込む
        </Button>
      </InfoBox>
    )
  }
  if (state.items.length === 0) {
    return <p className={styles.message}>{emptyMessage}</p>
  }
  return (
    <>
      {children(state.items)}
      {state.moreError && (
        <InfoBox tone="danger" icon="alert" role="alert">
          {state.moreError}
        </InfoBox>
      )}
      {state.nextCursor && (
        <Button
          variant="ghost"
          size="md"
          fullWidth
          disabled={state.loadingMore}
          onClick={onLoadMore}
        >
          {state.loadingMore ? '読み込み中…' : 'もっと見る'}
        </Button>
      )}
    </>
  )
}
