import type { Meta, StoryObj } from '@storybook/react-vite'
import { ItemRow } from '../../../../components/ItemRow/ItemRow'
import type { ItemSummary } from '../../api'
import type { ItemListState } from '../../hooks/useItemList'
import { rowProps } from '../../itemDisplay'
import { ItemListView } from './ItemListView'

const item = (shareId: string, title: string): ItemSummary => ({
  shareId,
  status: 'published',
  title,
  category: 'textbook',
  condition: 'good',
  price: 1500,
  campus: 'honjo',
  thumbnail: null,
  courseName: '線形代数学I',
  hasWriting: false,
  publishedAt: '2026-10-22T04:00:00Z',
})

const ready = (overrides: Partial<Extract<ItemListState, { status: 'ready' }>> = {}) =>
  ({
    status: 'ready',
    items: [item('a1', '線形代数入門'), item('a2', 'やさしい線形代数')],
    total: 12,
    nextCursor: 'c1',
    loadingMore: false,
    moreError: null,
    ...overrides,
  }) satisfies ItemListState

const meta = {
  title: '一覧・検索/ItemListView',
  component: ItemListView,
  args: {
    state: ready(),
    emptyMessage: '条件に合う商品が見つかりませんでした',
    onRetry: () => {},
    onLoadMore: () => {},
    children: (items: ItemSummary[]) => (
      <ul style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {items.map((it) => (
          <li key={it.shareId}>
            <ItemRow {...rowProps(it)} />
          </li>
        ))}
      </ul>
    ),
  },
  decorators: [
    (Story) => (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof ItemListView>

export default meta
type Story = StoryObj<typeof meta>

export const HasMore: Story = {}

export const LoadingMore: Story = { args: { state: ready({ loadingMore: true }) } }

export const LoadMoreError: Story = {
  args: { state: ready({ moreError: '読み込めませんでした。もう一度お試しください' }) },
}

export const AllLoaded: Story = { args: { state: ready({ nextCursor: null, total: 2 }) } }

export const Loading: Story = { args: { state: { status: 'loading' } } }

export const Empty: Story = { args: { state: ready({ items: [], total: 0, nextCursor: null }) } }

export const Error: Story = {
  args: { state: { status: 'error', message: '読み込めませんでした。もう一度お試しください' } },
}
