import type { Meta, StoryObj } from '@storybook/react-vite'
import { Icon } from './Icon'
import { iconNames } from './iconPaths'

const meta = {
  title: '基本/Icon',
  component: Icon,
  args: { name: 'search', size: 22 },
  argTypes: { name: { control: 'select', options: iconNames } },
} satisfies Meta<typeof Icon>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const All: Story = {
  render: () => (
    <ul style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16 }}>
      {iconNames.map((name) => (
        <li key={name} style={{ display: 'grid', justifyItems: 'center', gap: 4, fontSize: 11 }}>
          <Icon name={name} />
          {name}
        </li>
      ))}
    </ul>
  ),
}
