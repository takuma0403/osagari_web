import { Link, Navigate, type RouteObject } from 'react-router'
import { PlaceholderPage } from './components/PlaceholderPage/PlaceholderPage'
import { AdminLayout } from './layouts/AdminLayout/AdminLayout'
import { MobileLayout } from './layouts/MobileLayout/MobileLayout'

// 画面ごとの仮ページ。各画面の issue で features/<機能>/pages の画面に差し替える
const placeholder = (path: string, title: string): RouteObject => ({
  path,
  element: <PlaceholderPage title={title} />,
})

/**
 * 全画面のルート（FE）。商品ページの共有リンク（/items/:shareId）を含め、すべて SPA のルートとして扱う。
 * ルートの一覧は docs/tech/frontend.md の FE-1 にも載せている。変えるときは両方を直す。
 */
export const routes: RouteObject[] = [
  {
    // ログイン・登録（AUTH-1）。下部ナビなし
    element: <MobileLayout />,
    children: [
      placeholder('/login', 'ログイン'),
      placeholder('/signup', '新規登録'),
      placeholder('/signup/code', '認証コード入力（新規登録）'),
      placeholder('/signup/password', 'パスワード設定'),
      placeholder('/password-reset', 'パスワード再設定'),
      placeholder('/password-reset/code', '認証コード入力（パスワード再設定）'),
      placeholder('/password-reset/password', '新しいパスワードの設定'),
      placeholder('/recheck', 'ログインの確認'),
    ],
  },
  {
    // 利用者向けの画面。下部ナビ（BottomNav）は DS-3 の部品ができたら bottomNav に渡す
    element: <MobileLayout />,
    children: [
      placeholder('/', '商品一覧'),
      placeholder('/search', '検索結果'),
      placeholder('/items/:shareId', '商品詳細'),
      placeholder('/items/:shareId/request', '購入希望・決済'),
      placeholder('/items/:shareId/edit', '出品の編集'),
      placeholder('/sell', '出品'),
      placeholder('/deals', '取引一覧'),
      placeholder('/deals/:dealId', '取引詳細'),
      placeholder('/deals/:dealId/scan', 'QR読み取り'),
      placeholder('/deals/:dealId/complete', '受け渡しの完了'),
      placeholder('/mypage', 'マイページ'),
      placeholder('/mypage/student-card', '学生証登録'),
      placeholder('/mypage/sales', '売上管理'),
    ],
  },
  {
    // 管理画面（ADMIN）。PC 幅のレイアウト
    path: '/admin',
    element: <AdminLayout />,
    children: [
      { index: true, element: <Navigate to="/admin/student-cards" replace /> },
      placeholder('student-cards', '学生証審査'),
      placeholder('users', 'ユーザー'),
      placeholder('items', '出品・取引'),
      placeholder('settings', '設定'),
    ],
  },
  {
    element: <MobileLayout />,
    children: [
      {
        path: '*',
        element: (
          <PlaceholderPage title="ページが見つかりません">
            <Link to="/">商品一覧へ戻る</Link>
          </PlaceholderPage>
        ),
      },
    ],
  },
]
