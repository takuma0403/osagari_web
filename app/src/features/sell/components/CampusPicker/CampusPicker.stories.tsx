import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import type { Campus } from '../../api'
import { CampusPicker } from './CampusPicker'

function Demo({
  initial,
  error,
  disabled,
}: {
  initial: Campus | ''
  error?: string
  disabled?: boolean
}) {
  const [value, setValue] = useState(initial)
  return <CampusPicker value={value} onChange={setValue} error={error} disabled={disabled} />
}

const meta = {
  title: '出品/CampusPicker',
  component: Demo,
  args: { initial: 'honjo' },
} satisfies Meta<typeof Demo>

export default meta
type Story = StoryObj<typeof meta>

export const Selected: Story = {}

export const Empty: Story = { args: { initial: '' } }

export const WithError: Story = {
  args: { initial: '', error: '受け渡し希望キャンパスを選んでください' },
}

export const Disabled: Story = { args: { disabled: true } }
