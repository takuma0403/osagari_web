import type { Meta, StoryObj } from '@storybook/react-vite'
import { sampleUsers } from '../storyData'
import { UserDetailPanel } from './UserDetailPanel'

const meta = {
  title: '管理/UserDetailPanel',
  component: UserDetailPanel,
  args: {
    user: sampleUsers[1],
    running: null,
    error: null,
    onChangeExpiresOn: () => {},
    onSuspend: async () => true,
    onUnsuspend: () => {},
    onDelete: async () => true,
  },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof UserDetailPanel>

export default meta
type Story = StoryObj<typeof meta>

/** 有効・退学申請あり（デザインどおり） */
export const Default: Story = {}

/** 停止中は「停止を解除」を出す */
export const Suspended: Story = { args: { user: sampleUsers[2] } }

/** 失効したアカウントは操作できない */
export const Expired: Story = { args: { user: sampleUsers[4] } }

/** 操作の実行中はほかの操作を押せない */
export const Running: Story = { args: { running: 'unsuspend', user: sampleUsers[2] } }

/** 操作に失敗 */
export const Error: Story = { args: { error: '今日以降の日付にしてください' } }
