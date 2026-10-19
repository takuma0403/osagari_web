import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { sampleCards } from '../storyData'
import { CardQueue } from './CardQueue'

const meta = {
  title: '管理/CardQueue',
  component: CardQueue,
  args: { cards: sampleCards, selectedId: 'c1', onSelect: () => {} },
  parameters: { layout: 'padded' },
  render: function Render(args) {
    const [selectedId, setSelectedId] = useState(args.selectedId)
    return <CardQueue {...args} selectedId={selectedId} onSelect={setSelectedId} />
  },
} satisfies Meta<typeof CardQueue>

export default meta
type Story = StoryObj<typeof meta>

/** 先頭を選択中（デザインどおり）。学部コードが表にない行を含む */
export const Default: Story = {}

/** まだ何も選んでいない */
export const NoneSelected: Story = { args: { selectedId: null } }
