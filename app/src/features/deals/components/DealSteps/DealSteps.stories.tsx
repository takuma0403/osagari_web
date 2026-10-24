import type { Meta, StoryObj } from '@storybook/react-vite'
import { DealSteps } from './DealSteps'

const meta = {
  title: '取引/DealSteps',
  component: DealSteps,
  args: { status: 'requested' },
} satisfies Meta<typeof DealSteps>

export default meta
type Story = StoryObj<typeof meta>

export const Requested: Story = {}

export const InDeal: Story = { args: { status: 'in_deal' } }

export const Completed: Story = { args: { status: 'completed' } }

/** 下書き・取り下げなど、取引の段階に当たらないとき */
export const Withdrawn: Story = { args: { status: 'withdrawn' } }
