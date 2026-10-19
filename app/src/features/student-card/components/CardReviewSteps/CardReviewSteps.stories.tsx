import type { Meta, StoryObj } from '@storybook/react-vite'
import { CardReviewSteps } from './CardReviewSteps'

const meta = {
  title: '学生証/CardReviewSteps',
  component: CardReviewSteps,
  args: {
    card: { status: 'pending', submittedAt: '2026-10-08T05:02:00Z', reviewedAt: null },
  },
  decorators: [
    (Story) => (
      <div style={{ padding: 16, background: 'var(--color-bg)' }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof CardReviewSteps>

export default meta
type Story = StoryObj<typeof meta>

/** 未登録 */
export const None: Story = {
  args: { card: { status: 'none', submittedAt: null, reviewedAt: null } },
}

/** 審査中（デザインどおり） */
export const Pending: Story = {}

/** 差し戻し */
export const Rejected: Story = {
  args: {
    card: {
      status: 'rejected',
      submittedAt: '2026-10-08T05:02:00Z',
      reviewedAt: '2026-10-09T01:30:00Z',
    },
  },
}

/** 承認済み */
export const Approved: Story = {
  args: {
    card: {
      status: 'approved',
      submittedAt: '2026-10-08T05:02:00Z',
      reviewedAt: '2026-10-09T01:30:00Z',
    },
  },
}
