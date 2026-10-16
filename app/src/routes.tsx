import { Link, Navigate, type RouteObject } from 'react-router'
import { BottomNav } from './components/BottomNav/BottomNav'
import { GuestOnly, RequireAuth } from './features/auth/guards'
import { CodePage } from './features/auth/pages/CodePage'
import { LoginPage } from './features/auth/pages/LoginPage'
import { PasswordPage } from './features/auth/pages/PasswordPage'
import { SignUpPage } from './features/auth/pages/SignUpPage'
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
    // ログイン・登録（AUTH-1）。ログイン済みなら開こうとしていたページか商品一覧へ送る。下部ナビなし
    element: <GuestOnly />,
    children: [
      {
        element: <MobileLayout />,
        children: [
          { path: '/login', element: <LoginPage /> },
          { path: '/signup', element: <SignUpPage /> },
          { path: '/signup/code', element: <CodePage flow="signup" /> },
          { path: '/signup/password', element: <PasswordPage flow="signup" /> },
          { path: '/password-reset', element: <SignUpPage flow="passwordReset" /> },
          { path: '/password-reset/code', element: <CodePage flow="passwordReset" /> },
          { path: '/password-reset/password', element: <PasswordPage flow="passwordReset" /> },
          placeholder('/recheck', 'ログインの確認'),
        ],
      },
    ],
  },
  {
    // 利用規約・プライバシーポリシー（内容は未決。product/open-questions）。ログインの有無にかかわらず見られる
    element: <MobileLayout />,
    children: [placeholder('/terms', '利用規約'), placeholder('/privacy', 'プライバシーポリシー')],
  },
  {
    // ここから下はログインが必要（AUTHN-5）。未ログインならログイン画面へ送り、ログイン後に戻す。
    // 共有リンク（/items/:shareId）を未ログインで開いたときの表示は未決のため、決まるまではログインを求める
    element: <RequireAuth />,
    children: [
      {
        // 利用者向けの画面のうち、下部ナビ（BottomNav）を出すもの。デザインでナビがあるのは Main・Search・DealList・MyPage
        element: <MobileLayout bottomNav={<BottomNav />} />,
        children: [
          placeholder('/', '商品一覧'),
          placeholder('/search', '検索結果'),
          placeholder('/deals', '取引一覧'),
          placeholder('/mypage', 'マイページ'),
        ],
      },
      {
        // 利用者向けの画面のうち、下部ナビを出さないもの（詳細・入力・取引の操作など）
        element: <MobileLayout />,
        children: [
          placeholder('/items/:shareId', '商品詳細'),
          placeholder('/items/:shareId/request', '購入希望・決済'),
          placeholder('/items/:shareId/edit', '出品の編集'),
          placeholder('/sell', '出品'),
          placeholder('/deals/:dealId', '取引詳細'),
          placeholder('/deals/:dealId/scan', 'QR読み取り'),
          placeholder('/deals/:dealId/complete', '受け渡しの完了'),
          placeholder('/mypage/student-card', '学生証登録'),
          placeholder('/mypage/sales', '売上管理'),
        ],
      },
      {
        // 管理画面（ADMIN）。PC 幅のレイアウト。管理者かどうかの判定は管理者権限の issue で足す
        path: '/admin',
        element: <AdminLayout />,
        children: [
          { index: true, element: <Navigate to="/admin/student-cards" replace /> },
          placeholder('student-cards', '学生証審査'),
          placeholder('users', 'ユーザー'),
          placeholder('items', '出品・取引'),
          placeholder('settings', '設定（手数料率）'),
        ],
      },
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
