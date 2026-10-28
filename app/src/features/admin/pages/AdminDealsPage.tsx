import { useState } from 'react'
import { Badge } from '../../../components/Badge/Badge'
import { SegmentedTabs } from '../../../components/SegmentedTabs/SegmentedTabs'
import { Toast } from '../../../components/Toast/Toast'
import { formatMonthDay, formatPrice, formatShortDateTime } from '../../../lib/format'
import { itemStatusDisplay, transactionStatusDisplay, type StatusDisplay } from '../adminDisplay'
import {
  cancelTransactionByAdmin,
  listAdminItems,
  listAdminTransactions,
  suspendItem,
  type AdminItem,
  type AdminItemList,
  type AdminTransaction,
  type AdminTransactionList,
} from '../api'
import { AdminLoadState } from '../components/AdminLoadState/AdminLoadState'
import { AdminTable, type AdminColumn } from '../components/AdminTable/AdminTable'
import { ItemDetailPanel } from '../components/ItemDetailPanel/ItemDetailPanel'
import { TransactionDetailPanel } from '../components/TransactionDetailPanel/TransactionDetailPanel'
import { useAdminAction, type AdminNotice } from '../hooks/useAdminAction'
import { useCursorList } from '../hooks/useCursorList'
import common from './adminPage.module.css'

type Tab = 'items' | 'deals'

const tabs = [
  { value: 'items', label: '出品', panelId: 'admin-items' },
  { value: 'deals', label: '取引', panelId: 'admin-deals' },
] as const satisfies { value: Tab; label: string; panelId: string }[]

const statusBadge = ({ label, tone }: StatusDisplay) => <Badge tone={tone}>{label}</Badge>

const itemColumns: AdminColumn<AdminItem>[] = [
  { header: '商品', cell: (it) => it.title, tone: 'strong' },
  { header: '出品者', cell: (it) => it.sellerStudentNumber, tone: 'num' },
  {
    header: '価格',
    cell: (it) => (it.price === null ? '—' : formatPrice(it.price)),
    tone: 'price',
  },
  { header: 'ステータス', cell: (it) => statusBadge(itemStatusDisplay[it.status]) },
  { header: '出品日', cell: (it) => formatMonthDay(it.createdAt), tone: 'muted' },
]

const dealColumns: AdminColumn<AdminTransaction>[] = [
  { header: '商品', cell: (tx) => tx.item.title, tone: 'strong' },
  {
    header: '出品者 → 購入者',
    cell: (tx) => `${tx.sellerStudentNumber} → ${tx.buyerStudentNumber}`,
    tone: 'num',
  },
  { header: '金額', cell: (tx) => formatPrice(tx.price), tone: 'price' },
  { header: 'ステータス', cell: (tx) => statusBadge(transactionStatusDisplay(tx)) },
  { header: '更新', cell: (tx) => formatShortDateTime(tx.updatedAt), tone: 'muted' },
]

/**
 * 管理：出品・取引（ADMIN, DEAL-4）。デザインは AdminDeals.dc.html。
 * 出品と取引をタブで切り替え、選んだ出品の停止・取引のキャンセルと返金（モック）を行う（ADR 0034）。
 * 一覧の続き・確認・読み込み中などはデザインにないため、既存の部品で表す（ADR 0035）。
 */
export function AdminDealsPage() {
  const [tab, setTab] = useState<Tab>('items')
  const current = tab === 'items' ? tabs[0] : tabs[1]
  return (
    <>
      <div className={common.heading}>
        <h1 className={common.title}>出品・取引</h1>
        <SegmentedTabs label="表示する一覧" options={tabs} value={tab} onChange={setTab} onBg />
      </div>
      <div role="tabpanel" id={current.panelId} aria-label={current.label}>
        {tab === 'items' ? <ItemsBoard /> : <DealsBoard />}
      </div>
    </>
  )
}

function NoticeToast({ notice }: { notice?: AdminNotice }) {
  return <Toast message={notice?.message} icon={notice?.tone === 'alert' ? 'alert' : 'check'} />
}

/** 出品の一覧と、選んだ出品の詳細 */
function ItemsBoard() {
  const list = useCursorList<AdminItem, AdminItemList>(listAdminItems)
  const action = useAdminAction<'suspend'>(list.reload)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const items = list.entries
  const selected = items?.find((it) => it.id === selectedId) ?? items?.[0]

  return (
    <>
      {items ? (
        <div className={common.columns}>
          <AdminTable
            label="出品一覧"
            columns={itemColumns}
            rows={items}
            rowKey={(it) => it.id}
            selectedKey={selected?.id ?? null}
            onSelect={(id) => {
              setSelectedId(id)
              action.clearError()
            }}
            empty="出品はまだありません。"
            hasMore={list.nextCursor !== null}
            onLoadMore={list.loadMore}
            loadingMore={list.loadingMore}
            moreError={list.moreError}
          />
          {selected && (
            <ItemDetailPanel
              key={selected.id}
              item={selected}
              running={action.running !== null}
              error={action.error}
              onSuspend={() =>
                action.run(
                  'suspend',
                  () => suspendItem(selected.id),
                  (item) => {
                    list.replace(item)
                    return `「${item.title}」を停止しました`
                  },
                )
              }
            />
          )}
        </div>
      ) : (
        <AdminLoadState error={list.loadError} onReload={list.reload} />
      )}
      <NoticeToast notice={action.notice} />
    </>
  )
}

/** 取引の一覧と、選んだ取引の詳細 */
function DealsBoard() {
  const list = useCursorList<AdminTransaction, AdminTransactionList>(listAdminTransactions)
  const action = useAdminAction<'cancel'>(list.reload)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const deals = list.entries
  const selected = deals?.find((tx) => tx.id === selectedId) ?? deals?.[0]

  return (
    <>
      {deals ? (
        <div className={common.columns}>
          <AdminTable
            label="取引一覧"
            columns={dealColumns}
            rows={deals}
            rowKey={(tx) => tx.id}
            selectedKey={selected?.id ?? null}
            onSelect={(id) => {
              setSelectedId(id)
              action.clearError()
            }}
            empty="取引はまだありません。"
            hasMore={list.nextCursor !== null}
            onLoadMore={list.loadMore}
            loadingMore={list.loadingMore}
            moreError={list.moreError}
          />
          {selected && (
            <TransactionDetailPanel
              key={selected.id}
              transaction={selected}
              running={action.running !== null}
              error={action.error}
              onCancel={() =>
                action.run(
                  'cancel',
                  () => cancelTransactionByAdmin(selected.id),
                  (tx) => {
                    list.replace(tx)
                    return `「${tx.item.title}」の取引をキャンセルし、返金しました`
                  },
                )
              }
            />
          )}
        </div>
      ) : (
        <AdminLoadState error={list.loadError} onReload={list.reload} />
      )}
      <NoticeToast notice={action.notice} />
    </>
  )
}
