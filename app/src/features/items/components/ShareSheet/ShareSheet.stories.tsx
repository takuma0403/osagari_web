import type { Meta, StoryObj } from '@storybook/react-vite'
import { ShareSheet } from './ShareSheet'

const meta = {
  title: '商品/ShareSheet',
  component: ShareSheet,
  parameters: { layout: 'fullscreen' },
  args: { open: true, onCopy: () => {}, onClose: () => {} },
} satisfies Meta<typeof ShareSheet>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const CopyFailed: Story = {
  args: { error: 'コピーできませんでした。お使いのブラウザでは使えない可能性があります' },
}
