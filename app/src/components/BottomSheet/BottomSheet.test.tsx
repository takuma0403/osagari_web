import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BottomSheet } from './BottomSheet'

describe('BottomSheet', () => {
  it('見出しを名前にした dialog として、説明と中身を表示する', () => {
    render(
      <BottomSheet
        open
        onClose={() => {}}
        title="リンクをコピー"
        description="リンクと、商品の簡単な説明のテキストがコピーされます。"
      >
        <button>コピーする</button>
      </BottomSheet>,
    )
    const dialog = screen.getByRole('dialog', { name: 'リンクをコピー' })
    expect(dialog).toHaveTextContent('リンクと、商品の簡単な説明のテキストがコピーされます。')
    expect(screen.getByRole('button', { name: 'コピーする' })).toBeInTheDocument()
  })

  it('閉じているときは何も出さない', () => {
    render(<BottomSheet open={false} onClose={() => {}} title="リンクをコピー" />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('Esc で onClose が呼ばれる', async () => {
    const onClose = vi.fn()
    render(<BottomSheet open onClose={onClose} title="リンクをコピー" />)
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  it('見出しの右に action を置ける', () => {
    render(
      <BottomSheet open onClose={() => {}} title="絞り込み" action={<button>クリア</button>} />,
    )
    const dialog = screen.getByRole('dialog', { name: '絞り込み' })
    expect(within(dialog).getByRole('button', { name: 'クリア' })).toBeInTheDocument()
  })
})
