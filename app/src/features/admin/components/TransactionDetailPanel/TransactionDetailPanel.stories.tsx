import type { Meta, StoryObj } from '@storybook/react-vite'
import { sampleTransactions } from '../storyData'
import { TransactionDetailPanel } from './TransactionDetailPanel'

const meta = {
  title: '管理/TransactionDetailPanel',
  component: TransactionDetailPanel,
  args: {
    transaction: sampleTransactions[0],
    running: false,
    error: null,
    onCancel: async () => true,
  },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof TransactionDetailPanel>

export default meta
type Story = StoryObj<typeof meta>

/** 取引中・購入者がキャンセルを申請（デザインどおり） */
export const Default: Story = {}

/** 購入希望（成立前なので料率はまだない） */
export const Requested: Story = { args: { transaction: sampleTransactions[2] } }

/** 受け渡し完了後はキャンセルできない */
export const Completed: Story = { args: { transaction: sampleTransactions[3] } }

/** 双方の同意でキャンセル済み */
export const Cancelled: Story = { args: { transaction: sampleTransactions[4] } }

/** キャンセルに失敗 */
export const Error: Story = { args: { error: '読み込めませんでした。もう一度お試しください' } }
