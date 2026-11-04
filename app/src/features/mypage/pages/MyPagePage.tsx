import { Link, useSearchParams } from 'react-router'
import { Avatar } from '../../../components/Avatar/Avatar'
import { Button } from '../../../components/Button/Button'
import { Chip } from '../../../components/Chip/Chip'
import { InfoBox } from '../../../components/InfoBox/InfoBox'
import { SegmentedTabs } from '../../../components/SegmentedTabs/SegmentedTabs'
import { SelectField } from '../../../components/SelectField/SelectField'
import { departmentLabel } from '../../../lib/department'
import { formatDate } from '../../../lib/format'
import { categoryTint } from '../../../lib/itemOptions'
import type { CurrentUser } from '../../auth/api'
import { useSession } from '../../auth/session'
import { HistoryRow } from '../components/HistoryRow/HistoryRow'
import { useMyPageHistory, type MyPageHistoryState } from '../hooks/useMyPageHistory'
import {
  ownItemDisplay,
  ownStatusLabels,
  presentStatuses,
  purchaseDisplay,
  purchaseStatusLabels,
  sellerStatusLabels,
  sortByUpdatedAt,
  sortOrderOptions,
  type SortOrder,
} from '../myPageDisplay'
import styles from './MyPagePage.module.css'

type Tab = 'sold' | 'bought'

const panelId = 'mypage-history-panel'

/** URL の ?tab=bought なら購入した商品、それ以外は出品した商品 */
const tabOf = (tab: string | null): Tab => (tab === 'bought' ? 'bought' : 'sold')

/** URL の ?order=asc なら古い順、それ以外は新しい順 */
const orderOf = (order: string | null): SortOrder => (order === 'asc' ? 'asc' : 'desc')

/**
 * マイページ（AUTH-3, ITEM、MyPage）。プロフィールとアカウントの有効期限、出品と購入の履歴、売上管理・学生証の登録・利用規約・プライバシーポリシーへの導線を出す。
 * プロフィールはセッションの本人の情報（GET /me）、出品は GET /me/items、購入は購入側の取引のうち取引中・完了（ADR 0029・0030）。
 * 選んでいるタブ・状態の絞り込み・並び順は URL のクエリに持つ（行の先から戻ったときに同じ表示を出す。ADR 0031）。
 */
export function MyPagePage() {
  const { user } = useSession()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = tabOf(searchParams.get('tab'))
  const status = searchParams.get('status')
  const order = orderOf(searchParams.get('order'))
  const { state, reload } = useMyPageHistory()

  /** クエリの1つを変える。既定の値（null）なら外す */
  const changeParam = (name: string, value: string | null) =>
    setSearchParams(
      (params) => {
        if (value === null) params.delete(name)
        else params.set(name, value)
        return params
      },
      { replace: true },
    )

  // 状態はタブごとに違うので、タブを替えたら絞り込みを外す（並び順は残す）
  const changeTab = (next: Tab) =>
    setSearchParams(
      (params) => {
        params.delete('status')
        if (next === 'bought') params.set('tab', 'bought')
        else params.delete('tab')
        return params
      },
      { replace: true },
    )

  const count = (n: (s: Extract<MyPageHistoryState, { status: 'ready' }>) => number) =>
    state.status === 'ready' ? ` ${n(state)}` : ''
  const tabs = [
    { value: 'sold', label: `出品した商品${count((s) => s.items.length)}`, panelId },
    { value: 'bought', label: `購入した商品${count((s) => s.purchases.length)}`, panelId },
  ] as const satisfies { value: Tab; label: string; panelId: string }[]

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>マイページ</h1>
      {user && <Profile user={user} />}
      <div className={styles.tabs}>
        <SegmentedTabs
          label="履歴の種類"
          options={tabs}
          value={tab}
          onChange={changeTab}
          variant="underline"
        />
      </div>
      <div id={panelId} role="tabpanel" className={styles.panel}>
        <History
          state={state}
          tab={tab}
          status={status}
          order={order}
          onStatusChange={(next) => changeParam('status', next)}
          onOrderChange={(next) => changeParam('order', next === 'asc' ? 'asc' : null)}
          onReload={reload}
        />
      </div>
      <nav aria-label="設定" className={styles.settings}>
        <Link to="/mypage/sales" className={styles.setting}>
          売上管理<span className={styles.chevron}>›</span>
        </Link>
        <Link to="/mypage/student-card" className={styles.setting}>
          学生証の登録
          <span
            className={[styles.cardStatus, user?.sellerStatus === 'pending' && styles.pending]
              .filter(Boolean)
              .join(' ')}
          >
            {user && `${sellerStatusLabels[user.sellerStatus]} `}›
          </span>
        </Link>
        <Link to="/terms" className={styles.setting}>
          利用規約<span className={styles.chevron}>›</span>
        </Link>
        <Link to="/privacy" className={styles.setting}>
          プライバシーポリシー<span className={styles.chevron}>›</span>
        </Link>
      </nav>
    </div>
  )
}

/** 本人の学籍番号・学部・入学年度と、アカウントの有効期限（AUTH-2・3） */
function Profile({ user }: { user: CurrentUser }) {
  return (
    <section aria-label="プロフィール" className={styles.profile}>
      <div className={styles.identity}>
        <Avatar departmentName={user.departmentName} size="lg" inverse />
        <div className={styles.names}>
          <span className={styles.studentNumber}>{user.studentNumber}</span>
          <span className={styles.department}>
            {departmentLabel(user.departmentName)} · {user.admissionYear}年度入学
          </span>
        </div>
      </div>
      <dl className={styles.expiry}>
        <dt>アカウント有効期限</dt>
        <dd>{formatDate(user.expiresOn)}</dd>
      </dl>
    </section>
  )
}

const empty = {
  sold: { message: '出品した商品はまだありません。', to: '/sell', link: '出品する' },
  bought: { message: '購入した商品はまだありません。', to: '/', link: '商品一覧を見る' },
} as const

type HistoryProps = {
  state: MyPageHistoryState
  tab: Tab
  /** 絞り込む状態。null か一覧にない状態ならすべて */
  status: string | null
  order: SortOrder
  onStatusChange: (status: string | null) => void
  onOrderChange: (order: SortOrder) => void
  onReload: () => void
}

/**
 * 選んでいるタブの履歴。状態の絞り込み（一覧にある状態だけをチップに出す）と、最後に動いた日時の並べ替えができる（ADR 0031）。
 */
function History({
  state,
  tab,
  status,
  order,
  onStatusChange,
  onOrderChange,
  onReload,
}: HistoryProps) {
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
        <Button variant="text" size="sm" onClick={onReload}>
          もう一度読み込む
        </Button>
      </InfoBox>
    )
  }

  const list = tab === 'sold' ? state.items : state.purchases
  if (list.length === 0) {
    return (
      <div className={styles.empty}>
        <p>{empty[tab].message}</p>
        <Link to={empty[tab].to} className={styles.emptyLink}>
          {empty[tab].link}
        </Link>
      </div>
    )
  }

  const labels: Record<string, string> = tab === 'sold' ? ownStatusLabels : purchaseStatusLabels
  const statuses = presentStatuses(labels, list)
  const selected = status && statuses.includes(status) ? status : null
  const pick = <T extends { status: string; updatedAt: string }>(rows: T[]) =>
    sortByUpdatedAt(
      rows.filter((row) => !selected || row.status === selected),
      order,
    )

  const rows =
    tab === 'sold'
      ? pick(state.items).map((item) => (
          <li key={item.id}>
            <HistoryRow
              title={item.title}
              price={item.price}
              // 自分の出品の一覧にはカテゴリがないので、写真がないときは同じ地の色にする
              tint="textbook"
              imageUrl={item.thumbnail?.url}
              {...ownItemDisplay(item)}
            />
          </li>
        ))
      : pick(state.purchases).map((entry) => (
          <li key={entry.transactionId}>
            <HistoryRow
              title={entry.item.title}
              price={entry.price}
              tint={categoryTint[entry.item.category]}
              imageUrl={entry.item.thumbnail?.url}
              {...purchaseDisplay(entry)}
            />
          </li>
        ))

  return (
    <>
      <div className={styles.toolbar}>
        <div className={styles.statuses} role="group" aria-label="状態で絞り込む">
          <Chip label="すべて" selected={!selected} onClick={() => onStatusChange(null)} />
          {statuses.map((s) => (
            <Chip
              key={s}
              label={labels[s]}
              selected={selected === s}
              onClick={() => onStatusChange(s)}
            />
          ))}
        </div>
        <SelectField
          label="並び順"
          hideLabel
          options={sortOrderOptions}
          value={order}
          onChange={(next) => onOrderChange(next === 'asc' ? 'asc' : 'desc')}
          className={styles.order}
        />
      </div>
      <ul className={styles.list}>{rows}</ul>
    </>
  )
}
