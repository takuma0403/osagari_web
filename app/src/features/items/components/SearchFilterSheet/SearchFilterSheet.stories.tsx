import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { SearchFilterSheet } from './SearchFilterSheet'

const meta = {
  title: '一覧・検索/SearchFilterSheet',
  component: SearchFilterSheet,
  args: { conditions: {}, onApply: () => {}, onClose: () => {} },
} satisfies Meta<typeof SearchFilterSheet>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}

export const Selected: Story = {
  args: { conditions: { category: 'textbook', priceMax: 2000, campus: 'honjo' } },
}

/** 価格の下限が上限より大きいまま検索しようとしたとき */
export const PriceError: Story = {
  args: { conditions: { priceMin: 3000, priceMax: 2000 } },
  play: async () => {
    const body = within(document.body)
    await userEvent.click(body.getByRole('button', { name: 'この条件で検索' }))
    await expect(body.getByRole('alert')).toHaveTextContent('上限は下限以上にしてください')
  },
}
