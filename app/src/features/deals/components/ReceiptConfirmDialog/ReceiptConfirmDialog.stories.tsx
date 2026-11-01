import type { Meta, StoryObj } from '@storybook/react-vite'
import { ReceiptConfirmDialog } from './ReceiptConfirmDialog'

const meta = {
  title: '取引/ReceiptConfirmDialog',
  component: ReceiptConfirmDialog,
  args: { open: true, price: 1500, onConfirm: () => {}, onCancel: () => {} },
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ReceiptConfirmDialog>

export default meta
type Story = StoryObj<typeof meta>

/** 完了リンク・完了コードから（DealCompleteEnter） */
export const Default: Story = {}

/** QR の読み取りから（DealScan）。カメラ画面の上に出す */
export const OnCamera: Story = {
  args: { itemTitle: '線形代数入門', overlay: 'dark' },
}

/** 完了に失敗したとき */
export const Failed: Story = {
  args: { error: '受け渡しを完了できませんでした。もう一度お試しください' },
}
