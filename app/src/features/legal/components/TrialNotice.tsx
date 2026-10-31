import { useState } from 'react'
import { Outlet, useLocation } from 'react-router'
import { ConfirmDialog } from '../../../components/ConfirmDialog/ConfirmDialog'
import { acknowledgeTrialNotice, isTrialNoticeAcknowledged } from '../../../lib/trialNotice'

// 規約の本文を読んでいる間は重ねない
const LEGAL_PATHS = ['/terms', '/privacy']

/**
 * フェーズ1は試験運用で決済はモックであることを、初めて開いた端末で一度だけ伝える（OVERVIEW、PAY、利用規約 第２条）。
 * 読まずに閉じないよう、Esc・背面の幕では閉じず「確認した」でだけ閉じる。
 */
export function TrialNotice() {
  const { pathname } = useLocation()
  const [open, setOpen] = useState(() => !isTrialNoticeAcknowledged())

  const handleConfirm = () => {
    acknowledgeTrialNotice()
    setOpen(false)
  }

  return (
    <ConfirmDialog
      open={open && !LEGAL_PATHS.includes(pathname)}
      title="試験運用中です"
      confirmLabel="確認した"
      onConfirm={handleConfirm}
    >
      <p>
        代金・手数料・売上・出金の表示と操作は試験用の記録で、実際のお金の支払い・受け取りは発生しません。
      </p>
      <p>試験運用中は、実際の売買（お金のやり取りを伴う取引）に使わないでください。</p>
    </ConfirmDialog>
  )
}

/** 全画面の親ルート。各画面の上に試験運用の初回ダイアログを載せる */
export function RootLayout() {
  return (
    <>
      <Outlet />
      <TrialNotice />
    </>
  )
}
