import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConfirmDialog } from './ConfirmDialog'

function renderDialog(props: Partial<React.ComponentProps<typeof ConfirmDialog>> = {}) {
  const onConfirm = vi.fn()
  const onCancel = vi.fn()
  render(
    <ConfirmDialog
      open
      title="商品を受け取りましたか？"
      confirmLabel="受け取った"
      onConfirm={onConfirm}
      cancelLabel="まだ受け取っていない"
      onCancel={onCancel}
      {...props}
    >
      「受け取った」を押すと ¥1,500 が出品者に支払われ、取引が完了します。
    </ConfirmDialog>,
  )
  return { onConfirm, onCancel }
}

describe('ConfirmDialog', () => {
  it('見出しを名前にした dialog に、説明と2つのボタンを表示する', () => {
    renderDialog()
    const dialog = screen.getByRole('dialog', { name: '商品を受け取りましたか？' })
    expect(dialog).toHaveTextContent('¥1,500 が出品者に支払われ')
    expect(screen.getByRole('button', { name: '受け取った' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'まだ受け取っていない' })).toBeInTheDocument()
  })

  it('開いた直後は確定ボタンにフォーカスを置かない', () => {
    renderDialog()
    expect(screen.getByRole('button', { name: '受け取った' })).not.toHaveFocus()
  })

  it('確定・取り消しのボタンでそれぞれ呼ばれる', async () => {
    const { onConfirm, onCancel } = renderDialog()
    await userEvent.click(screen.getByRole('button', { name: '受け取った' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    await userEvent.click(screen.getByRole('button', { name: 'まだ受け取っていない' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('Esc は取り消しとして扱う', async () => {
    const { onConfirm, onCancel } = renderDialog()
    await userEvent.keyboard('{Escape}')
    expect(onCancel).toHaveBeenCalled()
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('取り消しを省くと確定ボタンだけになり、Esc・背面の幕では閉じない', async () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog open title="試験運用中です" confirmLabel="確認した" onConfirm={onConfirm}>
        実際のお金は動きません。
      </ConfirmDialog>,
    )
    expect(screen.getAllByRole('button')).toHaveLength(1)
    await userEvent.keyboard('{Escape}')
    await userEvent.click(screen.getByTestId('modal-overlay'))
    expect(screen.getByRole('dialog', { name: '試験運用中です' })).toBeInTheDocument()
    expect(onConfirm).not.toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', { name: '確認した' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('confirmDisabled のときは確定できない', () => {
    renderDialog({ confirmDisabled: true })
    expect(screen.getByRole('button', { name: '受け取った' })).toBeDisabled()
  })
})
