import { useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import { Chip } from '../../../components/Chip/Chip'
import { Icon } from '../../../components/Icon/Icon'
import { ItemRow } from '../../../components/ItemRow/ItemRow'
import { ItemListView } from '../components/ItemListView/ItemListView'
import { SearchField } from '../components/SearchField/SearchField'
import { SearchFilterSheet } from '../components/SearchFilterSheet/SearchFilterSheet'
import { useItemList } from '../hooks/useItemList'
import { rowProps } from '../itemDisplay'
import {
  filterChips,
  parseSearchConditions,
  toSearchParams,
  type ItemSearchConditions,
} from '../searchConditions'
import styles from './SearchPage.module.css'

/**
 * 検索結果（BROWSE、Search）。キーワード・カテゴリ・授業名・価格帯・キャンパス・状態で絞り込み、新着順に並べる。
 * 条件は URL のクエリに持つ（戻る・再読み込みで同じ結果を出す）。条件なしで開いたら公開中の商品をすべて出す。
 */
export function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const conditions = parseSearchConditions(searchParams)
  const { state, loadMore, reload } = useItemList(conditions)
  const [filterOpen, setFilterOpen] = useState(false)

  const searchWith = (next: ItemSearchConditions) => setSearchParams(toSearchParams(next))
  const { keyword, ...filters } = conditions

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div className={styles.searchRow}>
          <Link to="/" className={styles.back} aria-label="戻る">
            <Icon name="back" />
          </Link>
          <SearchField
            // URL のキーワードが変わったら（戻るなど）入れ直す
            key={keyword ?? ''}
            defaultValue={keyword}
            className={styles.search}
            onSearch={(next) => searchWith({ ...conditions, keyword: next || undefined })}
          />
        </div>
        <div className={styles.chips}>
          <Chip label="絞り込み" icon="filter" onClick={() => setFilterOpen(true)} />
          {filterChips(conditions).map((chip) => (
            <Chip key={chip.id} label={chip.label} onRemove={() => searchWith(chip.without)} />
          ))}
        </div>
      </header>
      <section className={styles.body} aria-labelledby="search-title">
        <h1 id="search-title" className="visually-hidden">
          検索結果
        </h1>
        {state.status === 'ready' && (
          <p className={styles.summary} role="status">
            {state.total}件 · 新着順
          </p>
        )}
        <ItemListView
          state={state}
          emptyMessage="条件に合う商品が見つかりませんでした"
          onRetry={reload}
          onLoadMore={loadMore}
        >
          {(items) => (
            <ul className={styles.list}>
              {items.map((item) => (
                <li key={item.shareId}>
                  <ItemRow {...rowProps(item)} />
                </li>
              ))}
            </ul>
          )}
        </ItemListView>
      </section>
      {filterOpen && (
        <SearchFilterSheet
          conditions={filters}
          onClose={() => setFilterOpen(false)}
          onApply={(next) => {
            setFilterOpen(false)
            searchWith({ keyword, ...next })
          }}
        />
      )}
    </div>
  )
}
