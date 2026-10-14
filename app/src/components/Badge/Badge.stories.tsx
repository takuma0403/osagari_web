import type { Meta, StoryObj } from '@storybook/react-vite'
import { Badge } from './Badge'

const meta = {
  title: '基本/Badge',
  component: Badge,
  args: { children: '取引中', tone: 'primary', size: 'sm' },
  argTypes: {
    tone: {
      control: 'inline-radio',
      options: ['primary', 'warning', 'danger', 'neutral', 'surface'],
    },
    size: { control: 'inline-radio', options: ['sm', 'md'] },
  },
} satisfies Meta<typeof Badge>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Tones: Story = {
  render: () => (
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        gap: 8,
        padding: 8,
        background: 'var(--color-bg)',
      }}
    >
      <Badge>取引中</Badge>
      <Badge tone="warning">購入希望 2件</Badge>
      <Badge tone="danger">停止中</Badge>
      <Badge tone="neutral">終了</Badge>
      <Badge tone="surface">教科書</Badge>
      <Badge tone="warning" size="md">
        決済はモック
      </Badge>
    </div>
  ),
}
