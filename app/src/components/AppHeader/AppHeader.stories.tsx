import type { Meta, StoryObj } from '@storybook/react-vite'
import { Badge } from '../Badge/Badge'
import { AppHeader } from './AppHeader'

const meta = {
  title: 'レイアウト/AppHeader',
  component: AppHeader,
  parameters: { layout: 'fullscreen' },
  args: { title: '受け渡し', backTo: '/deals' },
} satisfies Meta<typeof AppHeader>

export default meta
type Story = StoryObj<typeof meta>

export const WithBadge: Story = { args: { right: <Badge>取引中</Badge> } }

export const MockBadge: Story = {
  args: { title: '購入希望を出す', right: <Badge tone="warning">決済はモック</Badge> },
}

export const TitleOnly: Story = { args: { title: '取引', backTo: undefined } }

export const FormScreen: Story = { args: { title: undefined, backTo: '/signup', bordered: false } }

export const Closable: Story = { args: { title: '出品する', backTo: '/', closable: true } }

/** QRの読み取り（カメラ画面の暗い地の上） */
export const Dark: Story = {
  args: {
    title: 'QRを読み取る',
    backTo: '/deals/tx1',
    closable: true,
    bordered: false,
    tone: 'dark',
  },
  decorators: [
    (Story) => (
      <div style={{ background: 'var(--color-dark)' }}>
        <Story />
      </div>
    ),
  ],
}
