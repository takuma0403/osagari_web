import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from '../../../../components/Button/Button'
import { EmailField } from '../../../../components/EmailField/EmailField'
import { AuthFormLayout } from './AuthFormLayout'

const meta = {
  title: '認証/AuthFormLayout',
  component: AuthFormLayout,
  parameters: { layout: 'fullscreen' },
  args: {
    backTo: '/login',
    step: { current: 1, total: 3 },
    title: '新規登録',
    lead: '大学のメールアドレスに認証コードを送ります。',
    children: (
      <>
        <EmailField value="" onChange={() => {}} />
        <Button fullWidth disabled>
          認証コードを送る
        </Button>
      </>
    ),
  },
} satisfies Meta<typeof AuthFormLayout>

export default meta
type Story = StoryObj<typeof meta>

/** スマホ（SignUp.dc.html など）。白地に、ヘッダー・見出し・入力を縦に並べる */
export const Default: Story = {}

/** PC 幅。灰色の地の中央に最大 440px の白いカードを置く（DS-1） */
export const Pc: Story = {
  globals: { viewport: { value: 'pc1280', isRotated: false } },
}
