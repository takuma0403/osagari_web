import type { Meta, StoryObj } from '@storybook/react-vite'
import { ItemCard } from '../../components/ItemCard/ItemCard'
import { UserLayout } from './UserLayout'

const samples = Array.from({ length: 8 }, (_, i) => ({
  to: `/items/${i}`,
  title: `微分積分学 第${i + 1}版（書き込み少）`,
  price: 1200 + i * 100,
  categoryLabel: '教科書',
  tint: 'textbook' as const,
  campus: '本庄',
  condition: '目立った傷なし',
}))

// 列の幅とナビの位置を見るための枠。グリッドの列数は各画面（MainPage など）の CSS が決めるので、ここでは並べるだけ
const meta = {
  title: 'レイアウト/UserLayout',
  component: UserLayout,
  parameters: { layout: 'fullscreen', path: '/' },
  args: {
    children: (
      <ul
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))',
          gap: 'var(--space-gap)',
          padding: 'var(--space-page)',
        }}
      >
        {samples.map((s) => (
          <li key={s.to}>
            <ItemCard {...s} />
          </li>
        ))}
      </ul>
    ),
  },
} satisfies Meta<typeof UserLayout>

export default meta
type Story = StoryObj<typeof meta>

/** 商品一覧・検索結果の枠（スマホでは下部ナビ） */
export const WideWithNav: Story = { args: { nav: true, wide: true } }

/** 同じ枠の PC 幅。上部の横ナビと、最大 1080px の列 */
export const WideWithNavPc: Story = {
  args: { nav: true, wide: true },
  globals: { viewport: { value: 'pc1280', isRotated: false } },
}

/** フォーム・詳細などの枠の PC 幅。ナビなしで、最大 560px の列 */
export const NarrowPc: Story = {
  globals: { viewport: { value: 'pc1280', isRotated: false } },
}
