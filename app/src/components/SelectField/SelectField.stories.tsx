import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { SelectField } from './SelectField'

const options = [
  { value: 'new_like', label: '新品同様' },
  { value: 'good', label: '目立った傷なし' },
  { value: 'fair', label: 'やや傷あり' },
]

function Demo({ initial, error }: { initial: string; error?: string }) {
  const [value, setValue] = useState(initial)
  return (
    <SelectField
      label="状態"
      options={options}
      value={value}
      onChange={setValue}
      placeholder="選択してください"
      error={error}
    />
  )
}

const meta = {
  title: '入力/SelectField',
  component: Demo,
  args: { initial: 'good' },
} satisfies Meta<typeof Demo>

export default meta
type Story = StoryObj<typeof meta>

export const Selected: Story = {}

export const Empty: Story = { args: { initial: '' } }

export const WithError: Story = { args: { initial: '', error: '状態を選んでください' } }
