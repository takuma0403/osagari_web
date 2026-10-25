import type { Meta, StoryObj } from '@storybook/react-vite'
import { CancelSheet } from './CancelSheet'

const meta = {
  title: '取引/CancelSheet',
  component: CancelSheet,
  parameters: { layout: 'fullscreen' },
  args: { open: true, side: 'request', price: 1500, onSubmit: () => {}, onClose: () => {} },
} satisfies Meta<typeof CancelSheet>

export default meta
type Story = StoryObj<typeof meta>

/** 自分から申請する */
export const Request: Story = {}

/** 相手の申請に同意を求められた */
export const Agree: Story = { args: { side: 'agree' } }

/** 申請・同意の送信中 */
export const Submitting: Story = { args: { submitting: true } }

/** 送信に失敗した */
export const Failed: Story = {
  args: { side: 'agree', error: '通信できませんでした。接続を確認してもう一度お試しください' },
}
