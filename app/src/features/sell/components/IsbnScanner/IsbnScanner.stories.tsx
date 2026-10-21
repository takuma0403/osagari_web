import type { Meta, StoryObj } from '@storybook/react-vite'
import { IsbnScanner } from './IsbnScanner'
import type { StartScan } from './startIsbnScan'

// Storybook ではカメラを使わない。起動したまま待つ状態と、カメラを使えない状態を見る
const waiting: StartScan = async () => ({ stop: () => {} })
const denied: StartScan = () => Promise.reject(new DOMException('denied', 'NotAllowedError'))

const meta = {
  title: '出品/IsbnScanner',
  component: IsbnScanner,
  parameters: { layout: 'fullscreen' },
  args: { open: true, onClose: () => {}, onDetect: () => {}, startScan: waiting },
} satisfies Meta<typeof IsbnScanner>

export default meta
type Story = StoryObj<typeof meta>

export const Scanning: Story = {}

export const CameraDenied: Story = { args: { startScan: denied } }
