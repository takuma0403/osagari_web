import type { Meta, StoryObj } from '@storybook/react-vite'
import { HandoverPanel } from './HandoverPanel'

const inMinutes = (minutes: number) => new Date(Date.now() + minutes * 60 * 1000).toISOString()

const meta = {
  title: '取引/HandoverPanel',
  component: HandoverPanel,
  args: {
    url: 'https://example.com/deals/tx1/complete#q=3f9a1c7e5b2d4a6f8e0c1b3d5f7a9c2e',
    code: '381902',
    expiresAt: inMinutes(5),
    onReissue: () => {},
  },
} satisfies Meta<typeof HandoverPanel>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

/** 発行中 */
export const Issuing: Story = { args: { url: null, code: null, expiresAt: null } }

/** 有効期限（5分）が切れた */
export const Expired: Story = { args: { expiresAt: inMinutes(-1) } }
