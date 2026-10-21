import { Navigate, type RouteObject } from 'react-router'
import { BottomNav } from './components/BottomNav/BottomNav'
import { NotFoundPage } from './components/NotFoundPage/NotFoundPage'
import { AdminCardPage } from './features/admin/pages/AdminCardPage'
import { RequireAdmin } from './features/admin/RequireAdmin'
import { GuestOnly, RequireAuth } from './features/auth/guards'
import { CodePage } from './features/auth/pages/CodePage'
import { LoginPage } from './features/auth/pages/LoginPage'
import { PasswordPage } from './features/auth/pages/PasswordPage'
import { RecheckPage } from './features/auth/pages/RecheckPage'
import { SignUpPage } from './features/auth/pages/SignUpPage'
import { DetailPage } from './features/items/pages/DetailPage'
import { MainPage } from './features/items/pages/MainPage'
import { SearchPage } from './features/items/pages/SearchPage'
import { SellPage } from './features/sell/pages/SellPage'
import { StudentCardPage } from './features/student-card/pages/StudentCardPage'
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
          { path: '/recheck', element: <RecheckPage /> },
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
    // 共有リンク（/items/:shareId）も未ログインならログインを求め、ログイン後に商品ページへ戻す（ADR 0020）
    element: <RequireAuth />,
    children: [
      {
        // 利用者向けの画面のうち、下部ナビ（BottomNav）を出すもの。デザインでナビがあるのは Main・Search・DealList・MyPage
        element: <MobileLayout bottomNav={<BottomNav />} />,
        children: [
          { path: '/', element: <MainPage /> },
          { path: '/search', element: <SearchPage /> },
          placeholder('/deals', '取引一覧'),
          placeholder('/mypage', 'マイページ'),
        ],
      },
      {
        // 利用者向けの画面のうち、下部ナビを出さないもの（詳細・入力・取引の操作など）
        element: <MobileLayout />,
        children: [
          { path: '/items/:shareId', element: <DetailPage /> },
          placeholder('/items/:shareId/request', '購入希望・決済'),
          { path: '/sell', element: <SellPage /> },
          { path: '/sell/:itemId', element: <SellPage /> },
          placeholder('/deals/:dealId', '取引詳細'),
          placeholder('/deals/:dealId/scan', 'QR読み取り'),
          placeholder('/deals/:dealId/complete', '受け渡しの完了'),
          { path: '/mypage/student-card', element: <StudentCardPage /> },
          placeholder('/mypage/sales', '売上管理'),
        ],
      },
      {
        // 管理画面（ADMIN）。PC 幅のレイアウト。管理者でなければ知らないパスと同じ「ページが見つかりません」を出す（ADR 0015）
        element: <RequireAdmin />,
        children: [
          {
            path: '/admin',
            element: <AdminLayout />,
            children: [
              { index: true, element: <Navigate to="/admin/student-cards" replace /> },
              { path: 'student-cards', element: <AdminCardPage /> },
              placeholder('users', 'ユーザー'),
              placeholder('items', '出品・取引'),
              placeholder('settings', '設定（手数料率）'),
            ],
          },
        ],
      },
    ],
  },
  {
    element: <MobileLayout />,
    children: [{ path: '*', element: <NotFoundPage /> }],
  },
]
