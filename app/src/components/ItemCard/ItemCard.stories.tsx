import type { Meta, StoryObj } from '@storybook/react-vite'
import { ItemCard } from './ItemCard'

const meta = {
  title: '商品/ItemCard',
  component: ItemCard,
  args: {
    to: '/items/abc123',
    title: '微分積分学 第3版（書き込み少）',
    price: 1200,
    categoryLabel: '教科書',
    tint: 'textbook',
    campus: '本庄',
    condition: '目立った傷なし',
  },
} satisfies Meta<typeof ItemCard>

export default meta
type Story = StoryObj<typeof meta>

// 1枚だけのときは、390px 幅の2列グリッドで並べたときの幅（約173px）にそろえる
export const Default: Story = {
  decorators: [
    (Story) => (
      <div style={{ width: 173 }}>
        <Story />
      </div>
    ),
  ],
}

const samples = [
  {
    categoryLabel: '教科書',
    tint: 'textbook',
    title: '微分積分学 第3版（書き込み少）',
    price: 1200,
    campus: '本庄',
    condition: '目立った傷なし',
  },
  {
    categoryLabel: '家電',
    tint: 'appliance',
    title: '一人暮らし用 冷蔵庫 2ドア 138L',
    price: 6000,
    campus: '本庄',
    condition: 'やや傷あり',
  },
  {
    categoryLabel: '教科書',
    tint: 'textbook',
    title: '解剖学講義 改訂3版',
    price: 3500,
    campus: '鍋島',
    condition: '書き込みあり',
  },
  {
    categoryLabel: '家具',
    tint: 'furniture',
    title: 'ローテーブル 折りたたみ',
    price: 800,
    campus: '本庄',
    condition: '目立った傷なし',
  },
  {
    categoryLabel: 'ゲーム',
    tint: 'game',
    title: '携帯ゲーム機 本体＋ソフト2本',
    price: 15000,
    campus: '本庄',
    condition: '良好',
  },
  {
    categoryLabel: '教科書',
    tint: 'textbook',
    title: 'ミクロ経済学入門',
    price: 900,
    campus: '本庄',
    condition: '新品同様',
  },
] as const

export const Grid: Story = {
  parameters: { layout: 'fullscreen' },
  render: () => (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
        gap: 'var(--space-gap)',
        padding: 'var(--space-page)',
        background: 'var(--color-bg)',
      }}
    >
      {samples.map((s, i) => (
        <ItemCard key={i} to={`/items/${i}`} {...s} />
      ))}
    </div>
  ),
}
