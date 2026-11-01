import type { Meta, StoryObj } from '@storybook/react-vite'
import { HistoryRow } from './HistoryRow'

const meta = {
  title: 'マイページ/HistoryRow',
  component: HistoryRow,
  args: {
    to: '/sell/i-1/requests',
    title: '線形代数入門',
    price: 1500,
    status: '購入希望 3件',
    tone: 'warning',
    tint: 'textbook',
  },
} satisfies Meta<typeof HistoryRow>

export default meta
type Story = StoryObj<typeof meta>

/** 購入希望が届いている出品 */
export const Requested: Story = {}

/** 取引中 */
export const InDeal: Story = {
  args: {
    to: '/items/def456',
    title: 'ローテーブル 折りたたみ',
    price: 800,
    status: '取引中',
    tone: 'primary',
    tint: 'furniture',
  },
}

/** 価格を入れていない下書き */
export const DraftWithoutPrice: Story = {
  args: { to: '/sell/i-3', title: '電気スタンド', price: null, status: '下書き', tone: 'neutral' },
}

/** 管理者が停止した出品 */
export const Suspended: Story = {
  args: { to: '/sell/i-4', title: '有機化学 演習', price: 700, status: '停止中', tone: 'danger' },
}
