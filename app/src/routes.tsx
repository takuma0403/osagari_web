import { Navigate, type RouteObject } from 'react-router'
import { NotFoundPage } from './components/NotFoundPage/NotFoundPage'
import { AdminCardPage } from './features/admin/pages/AdminCardPage'
import { AdminDealsPage } from './features/admin/pages/AdminDealsPage'
import { AdminSettingsPage } from './features/admin/pages/AdminSettingsPage'
import { AdminUsersPage } from './features/admin/pages/AdminUsersPage'
import { RequireAdmin } from './features/admin/RequireAdmin'
import { GuestOnly, RequireAuth } from './features/auth/guards'
import { CodePage } from './features/auth/pages/CodePage'
import { LoginPage } from './features/auth/pages/LoginPage'
import { PasswordPage } from './features/auth/pages/PasswordPage'
import { RecheckPage } from './features/auth/pages/RecheckPage'
import { SignUpPage } from './features/auth/pages/SignUpPage'
import { DealCompletePage } from './features/deals/pages/DealCompletePage'
import { DealDetailPage } from './features/deals/pages/DealDetailPage'
import { DealListPage } from './features/deals/pages/DealListPage'
import { DealRequestsPage } from './features/deals/pages/DealRequestsPage'
import { DealScanPage } from './features/deals/pages/DealScanPage'
import { RequestPage } from './features/deals/pages/RequestPage'
import { DetailPage } from './features/items/pages/DetailPage'
import { MainPage } from './features/items/pages/MainPage'
import { SearchPage } from './features/items/pages/SearchPage'
import { RootLayout } from './features/legal/components/TrialNotice'
import { LegalPage } from './features/legal/pages/LegalPage'
import { MyPagePage } from './features/mypage/pages/MyPagePage'
import { SalesPage } from './features/sales/pages/SalesPage'
import { SellPage } from './features/sell/pages/SellPage'
import { StudentCardPage } from './features/student-card/pages/StudentCardPage'
import { AdminLayout } from './layouts/AdminLayout/AdminLayout'
import { UserLayout } from './layouts/UserLayout/UserLayout'

/**
 * 全画面のルート（FE）。商品ページの共有リンク（/items/:shareId）を含め、すべて SPA のルートとして扱う。
 * ルートの一覧は docs/tech/frontend.md の FE-1 にも載せている。変えるときは両方を直す。
 */
export const routes: RouteObject[] = [
  {
    // 全画面の上に、試験運用の初回ダイアログを載せる（OVERVIEW、PAY。/terms・/privacy では出さない）
    element: <RootLayout />,
    children: [
      {
        // ログイン・登録（AUTH-1）。ログイン済みなら開こうとしていたページか商品一覧へ送る。下部ナビなし
        element: <GuestOnly />,
        children: [
          {
            element: <UserLayout />,
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
        // 利用規約・プライバシーポリシー（本文の正本は docs/legal）。ログインの有無にかかわらず見られる
        element: <UserLayout />,
        children: [
          { path: '/terms', element: <LegalPage doc="terms" /> },
          { path: '/privacy', element: <LegalPage doc="privacy" /> },
        ],
      },
      {
        // ここから下はログインが必要（AUTHN-5）。未ログインならログイン画面へ送り、ログイン後に戻す。
        // 共有リンク（/items/:shareId）も未ログインならログインを求め、ログイン後に商品ページへ戻す（ADR 0020）
        element: <RequireAuth />,
        children: [
          {
            // 利用者向けの画面のうち、ナビ（BottomNav）を出すもの。デザインでナビがあるのは Main・Search・DealList・MyPage。
            // PC 幅では列を 1080px に広げ、一覧の列数を増やす（DS-1）
            element: <UserLayout nav wide />,
            children: [
              { path: '/', element: <MainPage /> },
              { path: '/search', element: <SearchPage /> },
              { path: '/deals', element: <DealListPage /> },
              { path: '/mypage', element: <MyPagePage /> },
            ],
          },
          {
            // ナビを出さない画面のうち、PC 幅で列を広げて2カラムにするもの（出品・取引の操作。DS-1）
            element: <UserLayout wide />,
            children: [
              { path: '/items/:shareId/request', element: <RequestPage /> },
              { path: '/sell', element: <SellPage /> },
              { path: '/sell/:itemId', element: <SellPage /> },
              { path: '/sell/:itemId/requests', element: <DealRequestsPage /> },
              { path: '/deals/:dealId', element: <DealDetailPage /> },
              { path: '/deals/:dealId/scan', element: <DealScanPage /> },
              { path: '/deals/:dealId/complete', element: <DealCompletePage /> },
            ],
          },
          {
            // ナビを出さず、PC 幅でも 560px の列のままの画面（商品詳細・学生証・売上管理）
            element: <UserLayout />,
            children: [
              { path: '/items/:shareId', element: <DetailPage /> },
              { path: '/mypage/student-card', element: <StudentCardPage /> },
              { path: '/mypage/sales', element: <SalesPage /> },
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
                  { path: 'users', element: <AdminUsersPage /> },
                  { path: 'items', element: <AdminDealsPage /> },
                  { path: 'settings', element: <AdminSettingsPage /> },
                ],
              },
            ],
          },
        ],
      },
      {
        element: <UserLayout />,
        children: [{ path: '*', element: <NotFoundPage /> }],
      },
    ],
  },
]
