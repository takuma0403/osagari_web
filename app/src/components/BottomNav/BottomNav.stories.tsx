import type { Meta, StoryObj } from '@storybook/react-vite'
import { BottomNav } from './BottomNav'

// いまのパスは .storybook/preview.tsx のルーターに parameters.path で渡す
const meta = {
  title: 'レイアウト/BottomNav',
  component: BottomNav,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof BottomNav>

export default meta
type Story = StoryObj<typeof meta>

export const Home: Story = { parameters: { path: '/' } }

export const Search: Story = { parameters: { path: '/search' } }

export const Deals: Story = { parameters: { path: '/deals/42' } }
