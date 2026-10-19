import { Link } from 'react-router'
import { PlaceholderPage } from '../PlaceholderPage/PlaceholderPage'

/**
 * 「ページが見つかりません」の画面。知らないパスと、管理者でない人が開いた管理画面で同じものを出し、
 * 管理画面があることを区別できないようにする（ADR 0015）
 */
export function NotFoundPage() {
  return (
    <PlaceholderPage title="ページが見つかりません">
      <Link to="/">商品一覧へ戻る</Link>
    </PlaceholderPage>
  )
}
