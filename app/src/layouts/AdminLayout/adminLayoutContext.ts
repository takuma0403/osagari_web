import { useOutletContext } from 'react-router'

/** 件数のバッジを出せる管理メニューの行（AdminCard のサイドバー） */
export type AdminMenuBadgeKey = 'studentCards'

export type AdminLayoutContext = {
  /** メニューの件数を設定する。0 か undefined ならバッジを出さない */
  setMenuBadge: (key: AdminMenuBadgeKey, count: number | undefined) => void
}

/** AdminLayout の内側の画面から、レイアウト（管理メニュー）を操作する */
export function useAdminLayout(): AdminLayoutContext {
  return useOutletContext<AdminLayoutContext>()
}
