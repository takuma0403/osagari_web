import type { Meta, StoryObj } from '@storybook/react-vite'
import { InfoBox } from './InfoBox'

const meta = {
  title: '基本/InfoBox',
  component: InfoBox,
  args: {
    tone: 'info',
    icon: 'shield',
    children:
      '前回の確認から30日たったか、はじめての端末からのログインのため、本人確認をお願いしています。',
  },
  argTypes: {
    tone: { control: 'inline-radio', options: ['info', 'warning', 'danger', 'neutral'] },
  },
} satisfies Meta<typeof InfoBox>

export default meta
type Story = StoryObj<typeof meta>

export const Info: Story = {}

export const Warning: Story = {
  args: {
    tone: 'warning',
    icon: undefined,
    children: '変更すると、ほかの端末ではログアウトされます。',
  },
}

export const Danger: Story = {
  args: {
    tone: 'danger',
    icon: 'lock',
    role: 'alert',
    children: 'パスワードを10回間違えたため、15分間ログインできません。',
  },
}

export const Neutral: Story = {
  args: {
    tone: 'neutral',
    icon: undefined,
    children: 'メールが届かないときは、迷惑メールフォルダも確認してください。',
  },
}
