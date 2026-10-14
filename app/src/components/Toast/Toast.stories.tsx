import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { Button } from '../Button/Button'
import { Toast } from './Toast'

const meta = {
  title: 'フィードバック/Toast',
  component: Toast,
  args: { message: 'リンクをコピーしました' },
} satisfies Meta<typeof Toast>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Interactive: Story = {
  render: function Render() {
    const [message, setMessage] = useState('')
    const show = () => {
      setMessage('リンクをコピーしました')
      setTimeout(() => setMessage(''), 2000)
    }
    return (
      <>
        <Button variant="ghost" icon="copy" onClick={show}>
          リンクをコピー
        </Button>
        <Toast message={message} />
      </>
    )
  },
}
