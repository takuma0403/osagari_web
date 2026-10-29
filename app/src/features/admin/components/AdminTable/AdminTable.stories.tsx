import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { Badge } from '../../../../components/Badge/Badge'
import { userStatusDisplay } from '../../adminDisplay'
import type { AdminUser } from '../../api'
import { sampleUsers } from '../storyData'
import { AdminTable, type AdminColumn } from './AdminTable'

const columns: AdminColumn<AdminUser>[] = [
  { header: '学籍番号', cell: (u) => u.studentNumber, tone: 'num' },
  {
    header: '状態',
    cell: (u) => (
      <Badge tone={userStatusDisplay[u.status].tone}>{userStatusDisplay[u.status].label}</Badge>
    ),
  },
  { header: 'キャンセル', cell: (u) => u.cancelCount, tone: 'num' },
]

const meta = {
  title: '管理/AdminTable',
  component: AdminTable<AdminUser>,
  args: {
    label: 'ユーザー一覧',
    columns,
    rows: sampleUsers,
    rowKey: (u) => u.id,
    selectedKey: 'u2',
    onSelect: () => {},
    empty: '条件に合うユーザーはいません。',
    hasMore: false,
    onLoadMore: () => {},
    loadingMore: false,
    moreError: null,
  },
  parameters: { layout: 'padded' },
  render: function Render(args) {
    const [selectedKey, setSelectedKey] = useState(args.selectedKey)
    return <AdminTable {...args} selectedKey={selectedKey} onSelect={setSelectedKey} />
  },
} satisfies Meta<typeof AdminTable<AdminUser>>

export default meta
type Story = StoryObj<typeof meta>

/** 2行目を選択中（デザインどおり） */
export const Default: Story = {}

/** 続きがある */
export const HasMore: Story = { args: { hasMore: true } }

/** 続きを読み込み中 */
export const LoadingMore: Story = { args: { hasMore: true, loadingMore: true } }

/** 続きの読み込みに失敗 */
export const MoreError: Story = {
  args: { hasMore: true, moreError: '読み込めませんでした。もう一度お試しください' },
}

/** 0件 */
export const Empty: Story = { args: { rows: [] } }
