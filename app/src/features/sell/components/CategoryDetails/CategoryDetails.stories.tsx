import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { emptySellForm, type SellErrors, type SellForm } from '../../form'
import { CategoryDetails } from './CategoryDetails'

function Demo({ initial, errors = {} }: { initial: Partial<SellForm>; errors?: SellErrors }) {
  const [form, setForm] = useState<SellForm>({ ...emptySellForm, ...initial })
  return (
    <CategoryDetails
      form={form}
      onChange={(key, value) => setForm((f) => ({ ...f, [key]: value }))}
      errors={errors}
    />
  )
}

const meta = {
  title: '出品/CategoryDetails',
  component: Demo,
  args: { initial: { category: 'textbook', courseName: '線形代数学I' } },
} satisfies Meta<typeof Demo>

export default meta
type Story = StoryObj<typeof meta>

export const Textbook: Story = {}

export const TextbookWithError: Story = {
  args: {
    initial: { category: 'textbook', isbn: '9784000000002' },
    errors: { isbn: 'ISBN を確認してください（10桁か13桁）' },
  },
}

export const FurnitureAppliance: Story = {
  args: {
    initial: { category: 'furniture_appliance', size: '幅60×奥行45cm', operationChecked: true },
  },
}
