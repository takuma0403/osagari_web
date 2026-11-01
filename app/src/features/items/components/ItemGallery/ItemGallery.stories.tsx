import type { Meta, StoryObj } from '@storybook/react-vite'
import { ItemGallery } from './ItemGallery'

// 写真の代わりに色だけの画像を使う（外部の画像を読み込まない）
const swatch = (color: string) =>
  `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="390" height="340"><rect width="100%" height="100%" fill="${color}"/></svg>`)}`

const meta = {
  title: '商品/ItemGallery',
  component: ItemGallery,
  args: { tint: 'textbook', imageUrls: [] },
  decorators: [
    (Story) => (
      <div style={{ width: 390, display: 'flex', flexDirection: 'column' }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof ItemGallery>

export default meta
type Story = StoryObj<typeof meta>

export const NoPhoto: Story = {}

export const OnePhoto: Story = {
  args: { imageUrls: [swatch('#9cc9cb')] },
}

export const SeveralPhotos: Story = {
  args: { imageUrls: [swatch('#9cc9cb'), swatch('#f2c94c'), swatch('#a8322d'), swatch('#1f7a80')] },
}
