import type { Meta, StoryObj } from '@storybook/react-vite'
import { ItemRow } from './ItemRow'

const meta = {
  title: '商品/ItemRow',
  component: ItemRow,
  args: {
    to: '/items/abc123',
    title: '線形代数入門',
    subtitle: '線形代数学I · 理工学部',
    price: 1500,
    note: '書き込みなし',
    tint: 'textbook',
  },
} satisfies Meta<typeof ItemRow>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const List: Story = {
  render: () => (
    <div
      style={{
        display: 'grid',
        gap: 10,
        padding: 'var(--space-page)',
        background: 'var(--color-bg)',
      }}
    >
      <ItemRow
        to="/items/1"
        title="線形代数入門"
        subtitle="線形代数学I · 理工学部"
        price={1500}
        note="書き込みなし"
        tint="textbook"
      />
      <ItemRow
        to="/items/2"
        title="線形代数 演習ノート付き"
        subtitle="線形代数学II"
        price={800}
        note="書き込みあり"
        tint="textbook"
      />
      <ItemRow to="/items/3" title="一人暮らし用 冷蔵庫 2ドア 138L" price={6000} tint="appliance" />
    </div>
  ),
}
