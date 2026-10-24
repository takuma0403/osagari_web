import type { Meta, StoryObj } from '@storybook/react-vite'
import { DealConfirmSheet } from './DealConfirmSheet'

const meta = {
  title: '取引/DealConfirmSheet',
  component: DealConfirmSheet,
  parameters: { layout: 'fullscreen' },
  args: { open: true, price: 1500, studentNumber: '26141023', onAcknowledge: () => {} },
} satisfies Meta<typeof DealConfirmSheet>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

/** 「確認しました」の送信中 */
export const Acknowledging: Story = { args: { acknowledging: true } }

/** 「確認しました」の送信に失敗した */
export const Failed: Story = {
  args: { error: '通信できませんでした。接続を確認してもう一度お試しください' },
}
