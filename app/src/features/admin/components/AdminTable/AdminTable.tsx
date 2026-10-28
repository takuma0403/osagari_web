import type { ReactNode } from 'react'
import { Button } from '../../../../components/Button/Button'
import styles from './AdminTable.module.css'

export type AdminColumn<T> = {
  header: string
  cell: (row: T) => ReactNode
  /** 学籍番号・金額は num（Outfit）、日付などは muted */
  tone?: 'num' | 'strong' | 'muted'
}

type Props<T> = {
  /** 一覧の名前（section の aria-label。例 ユーザー一覧） */
  label: string
  /** 先頭の列は行を選ぶボタンにする */
  columns: AdminColumn<T>[]
  rows: T[]
  rowKey: (row: T) => string
  selectedKey: string | null
  onSelect: (key: string) => void
  /** 0件のときの文 */
  empty: string
  /** 続きがあれば「もっと見る」を出す */
  hasMore: boolean
  onLoadMore: () => void
  loadingMore: boolean
  moreError: string | null
}

/**
 * 管理画面の一覧の表（AdminUsers・AdminDeals）。選んだ行は primary-soft の地と左の線で示し、右の詳細パネルに出す。
 * 行全体を押しても選べるが、キーボードと読み上げでは先頭の列のボタン（aria-pressed）で選ぶ。
 */
export function AdminTable<T>({
  label,
  columns,
  rows,
  rowKey,
  selectedKey,
  onSelect,
  empty,
  hasMore,
  onLoadMore,
  loadingMore,
  moreError,
}: Props<T>) {
  const [first, ...rest] = columns
  return (
    <section aria-label={label} className={styles.list}>
      {rows.length === 0 ? (
        <p className={styles.empty}>{empty}</p>
      ) : (
        <div className={styles.scroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                {columns.map((c) => (
                  <th key={c.header} scope="col" className={styles.th}>
                    {c.header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => {
                const key = rowKey(row)
                const selected = key === selectedKey
                return (
                  <tr
                    key={key}
                    className={selected ? `${styles.row} ${styles.selected}` : styles.row}
                    onClick={() => onSelect(key)}
                  >
                    <td
                      className={[styles.td, first.tone && styles[first.tone]]
                        .filter(Boolean)
                        .join(' ')}
                    >
                      <button
                        type="button"
                        aria-pressed={selected}
                        className={styles.select}
                        onClick={(e) => {
                          // 行の onClick と二重に呼ばない
                          e.stopPropagation()
                          onSelect(key)
                        }}
                      >
                        {first.cell(row)}
                      </button>
                    </td>
                    {rest.map((c) => (
                      <td
                        key={c.header}
                        className={[styles.td, c.tone && styles[c.tone]].filter(Boolean).join(' ')}
                      >
                        {c.cell(row)}
                      </td>
                    ))}
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {(hasMore || moreError) && (
        <div className={styles.more}>
          {moreError && (
            <p role="alert" className={styles.moreError}>
              {moreError}
            </p>
          )}
          {hasMore && (
            <Button variant="text" size="sm" onClick={onLoadMore} disabled={loadingMore}>
              {loadingMore ? '読み込み中…' : 'もっと見る'}
            </Button>
          )}
        </div>
      )}
    </section>
  )
}
