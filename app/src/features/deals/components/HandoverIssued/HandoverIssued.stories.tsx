import type { Meta, StoryObj } from '@storybook/react-vite'
import { HandoverIssued } from './HandoverIssued'

const in24Hours = () => new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()

const meta = {
  title: '取引/HandoverIssued',
  component: HandoverIssued,
  args: {
    kind: 'link',
    value: 'https://example.com/deals/tx1/complete#t=3f9a1c7e5b2d4a6f8e0c1b3d5f7a9c2e',
    expiresAt: in24Hours(),
  },
} satisfies Meta<typeof HandoverIssued>

export default meta
type Story = StoryObj<typeof meta>

/** 完了リンク */
export const Link: Story = {}

/** 6桁の完了コード */
export const Code: Story = { args: { kind: 'code', value: '582017' } }
