import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { sampleCards } from '../storyData'
import { CardReviewPanel } from './CardReviewPanel'

// 学生証の画像の代わり（外部に取りに行かないよう data URI にする）
const sampleImage =
  'data:image/svg+xml,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="320" height="200"><rect width="320" height="200" fill="#dceef6"/><text x="160" y="105" font-size="16" text-anchor="middle" fill="#566178">学生証の画像</text></svg>',
  )

const meta = {
  title: '管理/CardReviewPanel',
  component: CardReviewPanel,
  args: {
    card: sampleCards[0],
    image: { status: 'ready', url: sampleImage },
    onReloadImage: () => {},
    views: [
      { viewedAt: '2026-10-08T06:20:00Z', adminStudentNumber: '24101032' },
      { viewedAt: '2026-10-07T01:02:00Z', adminStudentNumber: '25238117' },
    ],
    deciding: null,
    onDecide: () => {},
  },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof CardReviewPanel>

export default meta
type Story = StoryObj<typeof meta>

/** 番号の入力前（承認はまだ押せない） */
export const Default: Story = {}

/** 番号がメールの学籍番号と一致（デザインどおり） */
export const Matched: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByLabelText('学生証に書かれた番号'), '26141023')
    await expect(canvas.getByRole('button', { name: '承認する' })).toBeEnabled()
  },
}

/** 番号が一致しない */
export const Mismatched: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByLabelText('学生証に書かれた番号'), '26141024')
    await expect(canvas.getByRole('button', { name: '承認する' })).toBeDisabled()
  },
}

/** 画像の閲覧URLを発行中 */
export const ImageLoading: Story = { args: { image: { status: 'loading' }, views: undefined } }

/** 画像の閲覧URLを発行できなかった */
export const ImageError: Story = {
  args: { image: { status: 'error', message: '見つかりません' }, views: [] },
}

/** 承認を送信中 */
export const Deciding: Story = { args: { deciding: 'approve' } }

/** 承認・差し戻しに失敗した */
export const DecisionError: Story = {
  args: { decisionError: 'サーバーでエラーが発生しました。時間をおいてもう一度お試しください' },
}

/** 学部コードが表にない */
export const UnknownDepartment: Story = { args: { card: sampleCards[3] } }
