import type { Meta, StoryObj } from '@storybook/react-vite'
import { Icon } from '../Icon/Icon'
import { TextField } from './TextField'

const meta = {
  title: '入力/TextField',
  component: TextField,
  args: { label: 'タイトル', defaultValue: '線形代数入門', size: 'lg' },
  argTypes: { size: { control: 'inline-radio', options: ['lg', 'md', 'sm'] } },
} satisfies Meta<typeof TextField>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Password: Story = {
  args: {
    label: 'パスワード',
    type: 'password',
    defaultValue: 'sagaunifurima',
    autoComplete: 'new-password',
    hint: (
      <span
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          color: 'var(--color-primary-hover)',
          fontWeight: 700,
        }}
      >
        <Icon name="check" size={16} strokeWidth={2.4} />
        10文字以上（13文字）
      </span>
    ),
  },
}

export const WithError: Story = {
  args: {
    label: 'パスワード',
    type: 'password',
    defaultValue: 'short',
    error: '10文字以上にしてください',
  },
}

export const Price: Story = {
  args: {
    label: '価格',
    size: 'md',
    prefix: '¥',
    numeric: true,
    defaultValue: '1500',
    inputMode: 'numeric',
  },
}

export const HiddenLabel: Story = {
  args: { label: '授業名', hideLabel: true, size: 'md', placeholder: '授業名', defaultValue: '' },
}

export const SmallOnPrimary: Story = {
  args: {
    label: '担当教員',
    hideLabel: true,
    size: 'sm',
    tone: 'primary',
    placeholder: '担当教員',
    defaultValue: '',
  },
}
