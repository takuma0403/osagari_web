import { useCallback, useState, type FormEvent } from 'react'
import { Badge } from '../../../components/Badge/Badge'
import { Chip } from '../../../components/Chip/Chip'
import { Icon } from '../../../components/Icon/Icon'
import { Toast } from '../../../components/Toast/Toast'
import { departmentLabel } from '../../../lib/department'
import { formatDate } from '../../../lib/format'
import { sellerStatusLabels } from '../../mypage/myPageDisplay'
import { userFilterOptions, userStatusDisplay } from '../adminDisplay'
import {
  deleteUser,
  listAdminUsers,
  suspendUser,
  unsuspendUser,
  updateUserExpiresOn,
  type AdminUser,
  type AdminUserFilter,
  type AdminUserList,
} from '../api'
import { AdminLoadState } from '../components/AdminLoadState/AdminLoadState'
import { AdminTable, type AdminColumn } from '../components/AdminTable/AdminTable'
import { UserDetailPanel, type UserAction } from '../components/UserDetailPanel/UserDetailPanel'
import { useAdminAction } from '../hooks/useAdminAction'
import { useCursorList } from '../hooks/useCursorList'
import common from './adminPage.module.css'
import styles from './AdminUsersPage.module.css'

const columns: AdminColumn<AdminUser>[] = [
  { header: '学籍番号', cell: (u) => <span className={styles.number}>{u.studentNumber}</span> },
  // 一覧は狭いので学部だけを出す（学科は詳細に出す）
  { header: '学部', cell: (u) => departmentLabel(u.departmentName).split(' ')[0] },
  {
    header: '状態',
    cell: (u) => (
      <Badge tone={userStatusDisplay[u.status].tone}>{userStatusDisplay[u.status].label}</Badge>
    ),
  },
  { header: '出品者', cell: (u) => sellerStatusLabels[u.sellerStatus], tone: 'muted' },
  { header: '有効期限', cell: (u) => formatDate(u.expiresOn) },
  { header: 'キャンセル', cell: (u) => u.cancelCount, tone: 'num' },
]

/**
 * 管理：ユーザー（ADMIN, AUTH-3）。デザインは AdminUsers.dc.html。
 * 学籍番号の前方一致と状態で絞り込み、選んだユーザーの有効期限の変更・停止と解除・退学による削除を行う（ADR 0034）。
 * 一覧の続き・確認・読み込み中などはデザインにないため、既存の部品で表す（ADR 0035）。
 */
export function AdminUsersPage() {
  const [keyword, setKeyword] = useState('')
  const [q, setQ] = useState('')
  const [filter, setFilter] = useState<AdminUserFilter>('all')
  const fetchPage = useCallback(
    (cursor: string | undefined, signal: AbortSignal) =>
      listAdminUsers({ q, filter }, cursor, signal),
    [q, filter],
  )
  const list = useCursorList<AdminUser, AdminUserList>(fetchPage)
  const action = useAdminAction<UserAction>(list.reload)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  // 退学申請の件数は先頭の応答の値から、この画面で削除した申請者の数を引く（読み直したら応答の値に戻る）
  const [deleted, setDeleted] = useState<{ head?: AdminUserList; count: number }>({ count: 0 })

  const users = list.entries
  const selected = users?.find((u) => u.id === selectedId) ?? users?.[0]
  const withdrawalCount =
    list.head && list.head.withdrawalRequestCount - (deleted.head === list.head ? deleted.count : 0)

  const search = (e: FormEvent) => {
    e.preventDefault()
    setQ(keyword)
  }

  const select = (id: string) => {
    setSelectedId(id)
    action.clearError()
  }

  const operate = (
    kind: UserAction,
    call: (user: AdminUser) => Promise<AdminUser>,
    message: (user: AdminUser) => string,
  ) => {
    if (!selected) return Promise.resolve(false)
    const before = selected
    return action.run(
      kind,
      () => call(before),
      (user) => {
        list.replace(user)
        if (kind === 'delete' && before.withdrawalRequestedAt)
          setDeleted((prev) => ({
            head: list.head,
            count: prev.head === list.head ? prev.count + 1 : 1,
          }))
        return message(user)
      },
    )
  }

  return (
    <>
      <div className={common.heading}>
        <h1 className={common.title}>ユーザー</h1>
        {withdrawalCount !== undefined && (
          <span className={common.summary}>退学申請 {withdrawalCount}件</span>
        )}
      </div>
      <div className={styles.conditions}>
        <form role="search" className={styles.search} onSubmit={search}>
          <label className={styles.searchBox}>
            <Icon name="search" size={18} />
            <span className="visually-hidden">学籍番号で検索</span>
            <input
              type="search"
              inputMode="numeric"
              enterKeyHint="search"
              maxLength={8}
              placeholder="学籍番号で検索"
              className={styles.searchInput}
              value={keyword}
              // 学籍番号の前方一致なので数字だけを受け付ける
              onChange={(e) => setKeyword(e.target.value.replace(/\D/g, ''))}
            />
          </label>
        </form>
        <div role="group" aria-label="状態で絞り込み" className={styles.filters}>
          {userFilterOptions.map((o) => (
            <Chip
              key={o.value}
              label={o.label}
              selected={filter === o.value}
              onClick={() => setFilter(o.value)}
            />
          ))}
        </div>
      </div>
      {users ? (
        <div className={common.columns}>
          <AdminTable
            label="ユーザー一覧"
            columns={columns}
            rows={users}
            rowKey={(u) => u.id}
            selectedKey={selected?.id ?? null}
            onSelect={select}
            empty="条件に合うユーザーはいません。"
            hasMore={list.nextCursor !== null}
            onLoadMore={list.loadMore}
            loadingMore={list.loadingMore}
            moreError={list.moreError}
          />
          {selected && (
            <UserDetailPanel
              key={selected.id}
              user={selected}
              running={action.running}
              error={action.error}
              onChangeExpiresOn={(expiresOn) =>
                void operate(
                  'expiresOn',
                  (u) => updateUserExpiresOn(u.id, expiresOn),
                  (u) => `${u.studentNumber} の有効期限を${formatDate(u.expiresOn)}に変更しました`,
                )
              }
              onSuspend={() =>
                operate(
                  'suspend',
                  (u) => suspendUser(u.id),
                  (u) => `${u.studentNumber} を停止しました`,
                )
              }
              onUnsuspend={() =>
                void operate(
                  'unsuspend',
                  (u) => unsuspendUser(u.id),
                  (u) => `${u.studentNumber} の停止を解除しました`,
                )
              }
              onDelete={() =>
                operate(
                  'delete',
                  (u) => deleteUser(u.id),
                  (u) => `${u.studentNumber} を削除しました`,
                )
              }
            />
          )}
        </div>
      ) : (
        <AdminLoadState error={list.loadError} onReload={list.reload} />
      )}
      <Toast
        message={action.notice?.message}
        icon={action.notice?.tone === 'alert' ? 'alert' : 'check'}
      />
    </>
  )
}
