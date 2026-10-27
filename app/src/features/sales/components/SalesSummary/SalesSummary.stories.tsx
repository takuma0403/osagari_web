import type { Meta, StoryObj } from '@storybook/react-vite'
import { SalesSummary } from './SalesSummary'

const meta = {
  title: '売上/SalesSummary',
  component: SalesSummary,
  args: {
    availableAmount: 7830,
    awaitingHandoverAmount: 720,
    paidOutAmount: 3150,
    onPayout: () => {},
  },
} satisfies Meta<typeof SalesSummary>

export default meta
type Story = StoryObj<typeof meta>

/** 出金できる売上がある */
export const Available: Story = {}

/** 出金できる売上がない（ボタンは押せない） */
export const NothingToPayOut: Story = {
  args: { availableAmount: 0, awaitingHandoverAmount: 720, paidOutAmount: 0 },
}

/** 出金の申請中 */
export const Paying: Story = {
  args: { payoutDisabled: true },
}
