import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { Button } from '../Button/Button'
import { Icon } from '../Icon/Icon'
import { BottomSheet } from './BottomSheet'

const meta = {
  title: 'フィードバック/BottomSheet',
  component: BottomSheet,
  args: {
    open: true,
    onClose: () => {},
    title: 'リンクをコピー',
    description: 'リンクと、商品の簡単な説明のテキストがコピーされます。',
  },
  render: function Render(args) {
    const [open, setOpen] = useState(args.open)
    return (
      <>
        <Button variant="ghost" onClick={() => setOpen(true)}>
          開く
        </Button>
        <BottomSheet {...args} open={open} onClose={() => setOpen(false)}>
          <Button fullWidth icon="copy">
            リンクをコピーする
          </Button>
          <Button variant="text" size="sm" onClick={() => setOpen(false)}>
            閉じる
          </Button>
        </BottomSheet>
      </>
    )
  },
} satisfies Meta<typeof BottomSheet>

export default meta
type Story = StoryObj<typeof meta>

export const Share: Story = {}

export const DealConfirmed: Story = {
  args: {
    title: '取引が成立しました',
    description:
      '代金 ¥1,500 は運営が預かっています。Teamsで相手に連絡して受け渡しを決めてください。',
    handle: false,
    leading: (
      <span
        style={{
          display: 'grid',
          placeItems: 'center',
          width: 52,
          height: 52,
          borderRadius: 'var(--radius-pill)',
          background: 'var(--color-primary-soft)',
          color: 'var(--color-primary)',
        }}
      >
        <Icon name="check" size={28} strokeWidth={2.2} />
      </span>
    ),
  },
}
