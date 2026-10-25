import type { Meta, StoryObj } from '@storybook/react-vite'
import { CancelStatus } from './CancelStatus'

const meta = {
  title: '取引/CancelStatus',
  component: CancelStatus,
  args: { kind: 'requested', onAction: () => {} },
} satisfies Meta<typeof CancelStatus>

export default meta
type Story = StoryObj<typeof meta>

/** 自分が申請中 */
export const Requested: Story = {}

/** 取り下げの送信中 */
export const Withdrawing: Story = { args: { busy: true } }

/** 相手の申請が届いている */
export const Received: Story = { args: { kind: 'received' } }
