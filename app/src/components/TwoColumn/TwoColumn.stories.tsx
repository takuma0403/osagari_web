import type { Meta, StoryObj } from '@storybook/react-vite'
import { InfoBox } from '../InfoBox/InfoBox'
import { TwoColumn } from './TwoColumn'

// 親の縦並び（画面の本文）の中に置いて使う。スマホでは縦に、PC 幅では左右に並ぶ
const meta = {
  title: 'レイアウト/TwoColumn',
  component: TwoColumn,
  decorators: [
    (Story) => (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <Story />
      </div>
    ),
  ],
  args: {
    start: <InfoBox icon="info">左（スマホでは上）に置く中身</InfoBox>,
    end: (
      <>
        <InfoBox icon="qr">右（スマホでは下）に置く中身</InfoBox>
        <InfoBox tone="warning" icon="alert">
          カラムの中の間隔は親の gap を引き継ぐ
        </InfoBox>
      </>
    ),
  },
} satisfies Meta<typeof TwoColumn>

export default meta
type Story = StoryObj<typeof meta>

export const Mobile: Story = {}

export const Pc: Story = {
  globals: { viewport: { value: 'pc1280', isRotated: false } },
}

export const NarrowStartPc: Story = {
  args: { variant: 'narrowStart' },
  globals: { viewport: { value: 'pc1280', isRotated: false } },
}
