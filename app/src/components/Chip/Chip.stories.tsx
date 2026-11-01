import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { Chip } from './Chip'

const meta = {
  title: '基本/Chip',
  component: Chip,
  args: { label: '教科書・参考書' },
} satisfies Meta<typeof Chip>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Selected: Story = { args: { selected: true } }

const categories = ['すべて', '教科書・参考書', '家具・家電', 'ゲーム・その他']

export const Categories: Story = {
  render: function Render() {
    const [current, setCurrent] = useState(categories[0])
    return (
      <div style={{ display: 'flex', gap: 8, overflowX: 'auto' }}>
        {categories.map((c) => (
          <Chip key={c} label={c} selected={c === current} onClick={() => setCurrent(c)} />
        ))}
      </div>
    )
  },
}

export const Filters: Story = {
  render: function Render() {
    const [conditions, setConditions] = useState(['教科書・参考書', '本庄', '〜¥2,000'])
    return (
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        <Chip label="絞り込み" icon="filter" />
        {conditions.map((c) => (
          <Chip
            key={c}
            label={c}
            onRemove={() => setConditions((prev) => prev.filter((x) => x !== c))}
          />
        ))}
      </div>
    )
  },
}
