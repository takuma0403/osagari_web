import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { Button } from '../Button/Button'
import { ConfirmDialog } from './ConfirmDialog'

const meta = {
  title: 'フィードバック/ConfirmDialog',
  component: ConfirmDialog,
  args: {
    open: true,
    title: '商品を受け取りましたか？',
    children: (
      <>
        「受け取った」を押すと <b style={{ fontFamily: 'var(--font-num)' }}>¥1,500</b>{' '}
        が出品者に支払われ、取引が完了します。完了後はアプリでキャンセル・返金できません。
      </>
    ),
    confirmLabel: '受け取った',
    onConfirm: () => {},
    cancelLabel: 'まだ受け取っていない',
    onCancel: () => {},
  },
  render: function Render(args) {
    const [open, setOpen] = useState(args.open)
    return (
      <>
        <Button variant="ghost" onClick={() => setOpen(true)}>
          開く
        </Button>
        <ConfirmDialog
          {...args}
          open={open}
          onConfirm={() => setOpen(false)}
          onCancel={() => setOpen(false)}
        />
      </>
    )
  },
} satisfies Meta<typeof ConfirmDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const OnCamera: Story = {
  args: { overlay: 'dark' },
  decorators: [
    (Story) => (
      <div style={{ minHeight: 600, background: 'var(--color-dark)' }}>
        <Story />
      </div>
    ),
  ],
}
