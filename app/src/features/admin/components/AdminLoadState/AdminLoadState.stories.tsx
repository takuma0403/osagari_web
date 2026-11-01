import type { Meta, StoryObj } from '@storybook/react-vite'
import { AdminLoadState } from './AdminLoadState'

const meta = {
  title: '管理/AdminLoadState',
  component: AdminLoadState,
  args: { error: null, onReload: () => {} },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof AdminLoadState>

export default meta
type Story = StoryObj<typeof meta>

/** 読み込み中 */
export const Loading: Story = {}

/** 読み込みに失敗 */
export const Failed: Story = { args: { error: '読み込めませんでした。もう一度お試しください' } }
