import type { Meta, StoryObj } from '@storybook/react-vite'
import { sampleItems } from '../storyData'
import { ItemDetailPanel } from './ItemDetailPanel'

const meta = {
  title: '管理/ItemDetailPanel',
  component: ItemDetailPanel,
  args: { item: sampleItems[0], running: false, error: null, onSuspend: async () => true },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof ItemDetailPanel>

export default meta
type Story = StoryObj<typeof meta>

/** 公開中（デザインどおり） */
export const Default: Story = {}

/** 停止中・取引完了などは停止できない */
export const NotSuspendable: Story = { args: { item: sampleItems[3] } }

/** 停止の実行中 */
export const Running: Story = { args: { running: true } }

/** 停止に失敗 */
export const Error: Story = { args: { error: '読み込めませんでした。もう一度お試しください' } }
