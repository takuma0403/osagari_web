import { Link, useSearchParams } from 'react-router'
import { Button } from '../../../components/Button/Button'
import { Icon } from '../../../components/Icon/Icon'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { SegmentedTabs } from '../../../components/SegmentedTabs/SegmentedTabs'
import type { TransactionListEntry, TransactionRole } from '../api'
import { DealRow } from '../components/DealRow/DealRow'
import { dealRowDisplay } from '../dealListDisplay'
import { useDealList } from '../hooks/useDealList'
import styles from './DealListPage.module.css'

const panelId = 'deal-list-panel'

const tabs = [
  { value: 'buyer', label: '購入', panelId },
  { value: 'seller', label: '出品', panelId },
] as const satisfies { value: TransactionRole; label: string; panelId: string }[]

const emptyMessages: Record<TransactionRole, string> = {
  buyer: '購入希望を出した商品はまだありません。',
  seller: '出品した商品の取引はまだありません。',
}

/** URL の ?tab=sell なら出品、それ以外は購入 */
const roleOf = (tab: string | null): TransactionRole => (tab === 'sell' ? 'seller' : 'buyer')

/**
 * 取引一覧（DEAL-1・3・4、DealList）。購入と出品を切り替え、それぞれの進行中・終了した取引を新しい順に出す（ADR 0028）。
 * 選んでいるタブは URL のクエリに持つ（取引詳細から戻ったときに同じタブを出す）。
 * 出品側の承認待ちの購入希望は商品ごとの1行で、購入希望の一覧へ進む。終了した取引は「もっと見る」で続きを読み込む。
 */
export function DealListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const role = roleOf(searchParams.get('tab'))
  const { state, loadMore, reload } = useDealList(role)

  const changeRole = (next: TransactionRole) =>
    setSearchParams(next === 'seller' ? { tab: 'sell' } : {}, { replace: true })

  const rows = (entries: TransactionListEntry[]) => (
    <ul className={styles.list}>
      {entries.map((entry) => (
        <li key={entry.transactionId ?? `item-${entry.itemId}`}>
          <DealRow item={entry.item} price={entry.price} {...dealRowDisplay(entry, role)} />
        </li>
      ))}
    </ul>
  )

  const body = () => {
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
          <Button variant="text" size="sm" onClick={reload}>
            もう一度読み込む
          </Button>
        </InfoBox>
      )
    }
    if (state.active.length === 0 && state.ended.length === 0) {
      return (
        <div className={styles.empty}>
          <Icon name="deal" size={40} />
          <p>{emptyMessages[role]}</p>
          <Link to="/" className={styles.emptyLink}>
            商品一覧を見る
          </Link>
        </div>
      )
    }
    return (
      <>
        {state.active.length > 0 && (
          <section className={styles.section} aria-labelledby="deal-list-active">
            <h2 id="deal-list-active" className={styles.sectionTitle}>
              進行中
            </h2>
            {rows(state.active)}
          </section>
        )}
        {state.ended.length > 0 && (
          <section className={styles.section} aria-labelledby="deal-list-ended">
            <h2 id="deal-list-ended" className={styles.sectionTitle}>
              終了した取引
            </h2>
            {rows(state.ended)}
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
                onClick={loadMore}
              >
                {state.loadingMore ? '読み込み中…' : 'もっと見る'}
              </Button>
            )}
          </section>
        )}
      </>
    )
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>取引</h1>
        <div className={styles.tabs}>
          <SegmentedTabs label="取引の種類" options={tabs} value={role} onChange={changeRole} />
        </div>
      </header>
      <div id={panelId} role="tabpanel" className={styles.body}>
        {body()}
      </div>
    </div>
  )
}
