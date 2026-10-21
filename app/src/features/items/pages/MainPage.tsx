import { useNavigate, useSearchParams } from 'react-router'
import { AppLogo } from '../../../components/AppLogo/AppLogo'
import { Chip } from '../../../components/Chip/Chip'
import { ItemCard } from '../../../components/ItemCard/ItemCard'
import { categoryOptions } from '../../../lib/itemOptions'
import { ItemListView } from '../components/ItemListView/ItemListView'
import { SearchField } from '../components/SearchField/SearchField'
import { useItemList } from '../hooks/useItemList'
import { cardProps } from '../itemDisplay'
import { parseSearchConditions, toSearchParams } from '../searchConditions'
import styles from './MainPage.module.css'

/**
 * 商品一覧（BROWSE、Main）。公開中の商品を新着順に2列で並べ、カテゴリで切り替える。
 * 選んだカテゴリは URL（/?category=）に持ち、戻ったときに同じ一覧を出す。
 * 検索欄で確定すると、選んでいるカテゴリを引き継いで検索結果へ移る。
 * デザインの右上のお知らせ（ベル）は、通知の機能が要件にないため置かない。
 */
export function MainPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  // 一覧で使う条件はカテゴリだけ
  const { category } = parseSearchConditions(searchParams)
  const { state, loadMore, reload } = useItemList({ category })

  const selectCategory = (next: typeof category) =>
    setSearchParams(toSearchParams({ category: next }), { replace: true })

  const search = (keyword: string) =>
    navigate({ pathname: '/search', search: toSearchParams({ keyword, category }).toString() })

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div className={styles.brand}>
          <AppLogo size={32} />
          <span className={styles.appName}>佐賀大お下がりフリマ</span>
        </div>
        <SearchField placeholder="教科書名・授業名・ISBNで検索" onSearch={search} />
        <div className={styles.categories} role="group" aria-label="カテゴリ">
          <Chip label="すべて" selected={!category} onClick={() => selectCategory(undefined)} />
          {categoryOptions.map((o) => (
            <Chip
              key={o.value}
              label={o.label}
              selected={category === o.value}
              onClick={() => selectCategory(o.value)}
            />
          ))}
        </div>
      </header>
      <section className={styles.body} aria-labelledby="main-title">
        <div className={styles.titleRow}>
          <h1 id="main-title" className={styles.title}>
            新着の出品
          </h1>
          <span className={styles.order}>新着順</span>
        </div>
        <ItemListView
          state={state}
          emptyMessage="まだ出品がありません"
          onRetry={reload}
          onLoadMore={loadMore}
        >
          {(items) => (
            <ul className={styles.grid}>
              {items.map((item) => (
                <li key={item.shareId}>
                  <ItemCard {...cardProps(item)} />
                </li>
              ))}
            </ul>
          )}
        </ItemListView>
      </section>
    </div>
  )
}
