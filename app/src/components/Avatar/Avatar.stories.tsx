import type { Meta, StoryObj } from '@storybook/react-vite'
import { Avatar } from './Avatar'

const meta = {
  title: '基本/Avatar',
  component: Avatar,
  args: { departmentName: '理工学部' },
} satisfies Meta<typeof Avatar>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

/** 学部が分からないとき */
export const Unknown: Story = { args: { departmentName: null } }
