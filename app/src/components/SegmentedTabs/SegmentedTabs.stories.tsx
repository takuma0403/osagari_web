import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { SegmentedTabs } from './SegmentedTabs'

const meta = {
  title: '基本/SegmentedTabs',
  component: SegmentedTabs<string>,
  args: {
    label: '取引の種類',
    options: [
      { value: 'buy', label: '購入' },
      { value: 'sell', label: '出品' },
    ],
    value: 'buy',
    onChange: () => {},
  },
  render: function Render(args) {
    const [value, setValue] = useState(args.value)
    return <SegmentedTabs {...args} value={value} onChange={setValue} />
  },
} satisfies Meta<typeof SegmentedTabs<string>>

export default meta
type Story = StoryObj<typeof meta>

export const DealList: Story = {}

export const AdminDeals: Story = {
  args: {
    label: '表示する一覧',
    options: [
      { value: 'items', label: '出品' },
      { value: 'deals', label: '取引' },
    ],
    value: 'items',
    onBg: true,
  },
  decorators: [
    (Story) => (
      <div style={{ padding: 16, background: 'var(--color-bg)', maxWidth: 280 }}>
        <Story />
      </div>
    ),
  ],
}
