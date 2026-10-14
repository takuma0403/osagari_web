import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { EmailField } from './EmailField'

const meta = {
  title: '入力/EmailField',
  component: EmailField,
  args: { value: '26238001', onChange: () => {} },
  render: function Render(args) {
    const [value, setValue] = useState(args.value)
    return <EmailField {...args} value={value} onChange={setValue} />
  },
} satisfies Meta<typeof EmailField>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Empty: Story = { args: { value: '' } }

export const WithError: Story = { args: { value: '2623', error: '学籍番号8桁を入力してください' } }
