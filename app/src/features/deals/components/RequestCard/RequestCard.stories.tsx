import type { Meta, StoryObj } from '@storybook/react-vite'
import { RequestCard } from './RequestCard'

const now = Date.parse('2026-10-08T03:40:00Z')

const meta = {
  title: '取引/RequestCard',
  component: RequestCard,
  args: {
    now,
    onApprove: () => {},
    onReject: () => {},
    request: {
      transactionId: 'tx-1',
      requestedAt: '2026-10-07T03:40:00Z',
      approvalDeadline: '2026-10-09T03:40:00Z',
      buyer: { departmentName: '経済学部', admissionYear: 2026, completedDealCount: 0 },
    },
  },
} satisfies Meta<typeof RequestCard>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

/** 期限が近い（1時間未満） */
export const NearDeadline: Story = {
  args: {
    request: {
      transactionId: 'tx-2',
      requestedAt: '2026-10-06T04:00:00Z',
      approvalDeadline: '2026-10-08T04:00:00Z',
      buyer: { departmentName: '理工学部', admissionYear: 2025, completedDealCount: 2 },
    },
  },
}

/** 学部が分からない購入者 */
export const UnknownDepartment: Story = {
  args: {
    request: {
      transactionId: 'tx-3',
      requestedAt: '2026-10-07T03:40:00Z',
      approvalDeadline: '2026-10-09T03:40:00Z',
      buyer: { departmentName: null, admissionYear: 2024, completedDealCount: 1 },
    },
  },
}

/** 承認・却下の送信中 */
export const Disabled: Story = { args: { disabled: true } }
