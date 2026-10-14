import type { Meta, StoryObj } from '@storybook/react-vite'
import { DefinitionList } from './DefinitionList'

const meta = {
  title: '基本/DefinitionList',
  component: DefinitionList,
  args: {
    items: [
      { term: '授業名', description: '線形代数学I' },
      { term: '担当教員', description: '[担当教員名]' },
      { term: 'ISBN', description: <span style={{ fontFamily: 'var(--font-num)' }}>[ISBN]</span> },
      { term: '受け渡し', description: '本庄キャンパス' },
    ],
  },
} satisfies Meta<typeof DefinitionList>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Boxed: Story = {
  args: {
    size: 'sm',
    boxed: true,
    items: [
      { term: '学籍番号', description: <b style={{ fontFamily: 'var(--font-num)' }}>26238001</b> },
      { term: '学部・学科', description: '理工学部 理工学科' },
      { term: '有効期限', description: '2030年3月31日' },
    ],
  },
}
