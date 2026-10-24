import type { Meta, StoryObj } from '@storybook/react-vite'
import { Badge } from '../../../../components/Badge/Badge'
import { DealItemSummary } from './DealItemSummary'

const meta = {
  title: '取引/DealItemSummary',
  component: DealItemSummary,
  args: { title: '線形代数入門', price: 1500, tint: 'textbook' },
} satisfies Meta<typeof DealItemSummary>

export default meta
type Story = StoryObj<typeof meta>

/** 購入希望の入力（Request） */
export const Large: Story = {
  args: { note: '本庄キャンパスで受け渡し' },
}

/** 取引詳細の見出し（DealRequests） */
export const Small: Story = {
  args: { size: 'sm', aside: <Badge tone="warning">あなたが出品</Badge> },
}
