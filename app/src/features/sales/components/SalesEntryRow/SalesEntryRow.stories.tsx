import type { Meta, StoryObj } from '@storybook/react-vite'
import { salesEntry } from '../../test/fixtures'
import { SalesEntryRow } from './SalesEntryRow'

const meta = {
  title: '売上/SalesEntryRow',
  component: SalesEntryRow,
  args: { entry: salesEntry() },
} satisfies Meta<typeof SalesEntryRow>

export default meta
type Story = StoryObj<typeof meta>

/** 受け渡しが完了して未出金（確定） */
export const Settled: Story = {}

/** 取引中（受け渡し待ち） */
export const AwaitingHandover: Story = {
  args: {
    entry: salesEntry({
      state: 'awaiting_handover',
      price: 800,
      fee: 40,
      sellerProceeds: 760,
      completedAt: null,
    }),
  },
}

/** 出金済み */
export const PaidOut: Story = {
  args: {
    entry: salesEntry({ state: 'paid_out', paidOutAt: '2026-10-20T03:00:00Z' }),
  },
}

/** 料率が小数（5.25%）のときに成立した取引 */
export const FractionalRate: Story = {
  args: {
    entry: salesEntry({ feeRateBasisPoints: 525, fee: 81, sellerProceeds: 1469 }),
  },
}

/** 商品名が長い */
export const LongTitle: Story = {
  args: {
    entry: salesEntry({
      item: {
        shareId: 'abc123',
        title: '解剖学講義 改訂3版 付録・演習問題集つき 2冊セット',
        category: 'textbook',
        thumbnail: null,
      },
    }),
  },
}
