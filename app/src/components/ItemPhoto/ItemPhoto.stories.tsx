import type { Meta, StoryObj } from '@storybook/react-vite'
import { ItemPhoto } from './ItemPhoto'

const meta = {
  title: '商品/ItemPhoto',
  component: ItemPhoto,
  args: { tint: 'textbook' },
  argTypes: {
    tint: { control: 'inline-radio', options: ['textbook', 'appliance', 'furniture', 'game'] },
  },
} satisfies Meta<typeof ItemPhoto>

export default meta
type Story = StoryObj<typeof meta>

export const Placeholder: Story = {
  decorators: [
    (Story) => (
      <div style={{ width: 120, height: 120, display: 'grid' }}>
        <Story />
      </div>
    ),
  ],
}

export const Tints: Story = {
  render: () => (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 72px)', gap: 8 }}>
      {(['textbook', 'appliance', 'furniture', 'game'] as const).map((tint) => (
        <div key={tint} style={{ height: 72, display: 'grid' }}>
          <ItemPhoto tint={tint} />
        </div>
      ))}
    </div>
  ),
}
