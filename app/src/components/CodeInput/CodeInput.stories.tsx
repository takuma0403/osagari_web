import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { CodeInput } from './CodeInput'

const meta = {
  title: '入力/CodeInput',
  component: CodeInput,
  args: {
    value: '481',
    onChange: () => {},
    expiresIn: '9:41',
    resend: { label: 'コードを再送（0:42 後）', onClick: () => {}, disabled: true },
  },
  render: function Render(args) {
    const [value, setValue] = useState(args.value)
    return <CodeInput {...args} value={value} onChange={setValue} />
  },
} satisfies Meta<typeof CodeInput>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const WithError: Story = {
  args: { error: 'コードが違います。あと3回間違えるとこのコードは使えなくなります。' },
}

export const CanResend: Story = {
  args: { value: '', resend: { label: 'コードを再送', onClick: () => {} } },
}

export const WithLabel: Story = {
  args: { label: 'メールに届いた6桁のコード', showLabel: true, value: '' },
}
