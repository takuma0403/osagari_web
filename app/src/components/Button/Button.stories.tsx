import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button, ButtonLink } from './Button'

const meta = {
  title: '基本/Button',
  component: Button,
  args: { children: 'ログイン', variant: 'primary', size: 'lg', fullWidth: true },
  argTypes: {
    variant: {
      control: 'inline-radio',
      options: ['primary', 'secondary', 'ghost', 'danger', 'text'],
    },
    size: { control: 'inline-radio', options: ['lg', 'md', 'sm'] },
  },
} satisfies Meta<typeof Button>

export default meta
type Story = StoryObj<typeof meta>

export const Primary: Story = {}

export const Disabled: Story = { args: { disabled: true } }

export const Variants: Story = {
  render: () => (
    <div style={{ display: 'grid', gap: 12 }}>
      <Button fullWidth>確認する</Button>
      <Button variant="secondary" fullWidth>
        新規登録
      </Button>
      <Button variant="ghost" size="md" fullWidth icon="copy">
        リンクをコピー
      </Button>
      <Button variant="danger" size="sm">
        キャンセルを申請
      </Button>
      <Button variant="text" size="sm">
        閉じる
      </Button>
      <Button variant="text" size="sm" disabled>
        コードを再送（0:42 後）
      </Button>
    </div>
  ),
}

export const Sizes: Story = {
  render: () => (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <Button size="lg">52px</Button>
      <Button size="md">48px</Button>
      <Button size="sm">44px</Button>
    </div>
  ),
}

export const AsLink: Story = {
  render: () => (
    <ButtonLink to="/signup" variant="secondary" fullWidth>
      新規登録
    </ButtonLink>
  ),
}
