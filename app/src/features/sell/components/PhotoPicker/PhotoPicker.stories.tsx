import type { Meta, StoryObj } from '@storybook/react-vite'
import { PhotoPicker } from './PhotoPicker'

// 写真の代わりに単色の画像を使う
const swatch = (color: string) =>
  `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" width="80" height="80"><rect width="80" height="80" fill="${color}"/></svg>`)}`

const meta = {
  title: '出品/PhotoPicker',
  component: PhotoPicker,
  args: {
    photos: [
      { id: 'a', url: swatch('#9cc9cb'), objectKey: 'u1/a' },
      { id: 'b', url: swatch('#dceef6'), objectKey: 'u1/b' },
    ],
    onAdd: () => {},
    onRemove: () => {},
    onMove: () => {},
  },
} satisfies Meta<typeof PhotoPicker>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Empty: Story = { args: { photos: [] } }

export const Uploading: Story = {
  args: {
    photos: [
      { id: 'a', url: swatch('#9cc9cb'), objectKey: 'u1/a' },
      { id: 'b', url: swatch('#dceef6') },
    ],
  },
}

export const WithError: Story = { args: { photos: [], error: '写真を1枚以上追加してください' } }
