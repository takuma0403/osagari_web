import type { Meta, StoryObj } from '@storybook/react-vite'
import { QrCode } from './QrCode'

const meta = {
  title: '基本/QrCode',
  component: QrCode,
  args: {
    value: 'https://example.com/deals/tx1/complete#q=3f9a1c7e5b2d4a6f8e0c1b3d5f7a9c2e',
    label: '受け渡し用QRコード',
  },
  decorators: [
    (Story) => (
      <div style={{ width: 240 }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof QrCode>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

/** 期限切れ（読み取らせない） */
export const Dimmed: Story = { args: { dimmed: true } }
