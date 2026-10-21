import type { Meta, StoryObj } from '@storybook/react-vite'
import { SearchField } from './SearchField'

const meta = {
  title: '一覧・検索/SearchField',
  component: SearchField,
  args: { onSearch: () => {} },
} satisfies Meta<typeof SearchField>

export default meta
type Story = StoryObj<typeof meta>

/** 商品一覧（Main）のヘッダー */
export const Placeholder: Story = { args: { placeholder: '教科書名・授業名・ISBNで検索' } }

/** 検索結果（Search）のヘッダー */
export const WithKeyword: Story = { args: { defaultValue: '線形代数' } }
