import type { Meta, StoryObj } from '@storybook/react-vite'
import { DealRow } from './DealRow'

const meta = {
  title: '取引/DealRow',
  component: DealRow,
  args: {
    to: '/deals/tx-1',
    item: {
      shareId: 'abc123',
      title: '微分積分学 第3版（書き込み少）',
      category: 'textbook',
      thumbnail: null,
    },
    price: 1200,
    status: '承認待ち',
    tone: 'warning',
    note: 'あと31時間で期限切れ',
  },
} satisfies Meta<typeof DealRow>

export default meta
type Story = StoryObj<typeof meta>

/** 購入側の購入希望（承認待ち） */
export const Requested: Story = {}

/** 取引中 */
export const InDeal: Story = {
  args: {
    item: {
      shareId: 'def456',
      title: '一人暮らし用 冷蔵庫 2ドア 138L',
      category: 'furniture_appliance',
      thumbnail: null,
    },
    price: 6000,
    status: '取引中',
    tone: 'primary',
    note: '受け渡しでQRを表示',
  },
}

/** 出品側の商品ごとの購入希望 */
export const RequestGroup: Story = {
  args: {
    to: '/sell/i-1/requests',
    item: { shareId: 'ghi789', title: '解剖学講義 改訂3版', category: 'textbook', thumbnail: null },
    price: 3500,
    status: '購入希望 3件',
    tone: 'warning',
    note: '相手を選んでください',
  },
}

/** 完了 */
export const Completed: Story = {
  args: {
    item: {
      shareId: 'jkl012',
      title: 'ローテーブル 折りたたみ',
      category: 'furniture_appliance',
      thumbnail: null,
    },
    price: 800,
    status: '完了',
    tone: 'neutral',
    note: '9/28 受け渡し',
  },
}

/** キャンセル */
export const Cancelled: Story = {
  args: {
    item: { shareId: 'mno345', title: '有機化学 演習', category: 'textbook', thumbnail: null },
    price: 700,
    status: 'キャンセル',
    tone: 'neutral',
    note: '双方の同意で取り消し',
  },
}
