import type { Meta, StoryObj } from '@storybook/react-vite'
import { TextAreaField } from './TextAreaField'

const meta = {
  title: '入力/TextAreaField',
  component: TextAreaField,
  args: { label: '説明', rows: 3, defaultValue: '1年前期で使いました。書き込みはありません。' },
} satisfies Meta<typeof TextAreaField>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const WithError: Story = { args: { defaultValue: '', error: '説明を入力してください' } }
