import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { Modal } from './Modal'

function Harness({ onClose = () => {} }: { onClose?: () => void }) {
  const [open, setOpen] = useState(false)
  const close = () => {
    onClose()
    setOpen(false)
  }
  return (
    <>
      <button onClick={() => setOpen(true)}>開く</button>
      <Modal open={open} onClose={close} labelledBy="t" placement="center">
        <h2 id="t">確認</h2>
        <button>はい</button>
        <button onClick={close}>いいえ</button>
      </Modal>
    </>
  )
}

describe('Modal', () => {
  it('開くと見出しを名前にした dialog を出し、ダイアログ自体にフォーカスを移す', async () => {
    render(<Harness />)
    await userEvent.click(screen.getByRole('button', { name: '開く' }))
    const dialog = screen.getByRole('dialog', { name: '確認' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveFocus()
    expect(document.body.style.overflow).toBe('hidden')
  })

  it('Esc で閉じ、開く前の要素にフォーカスを戻す', async () => {
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)
    const opener = screen.getByRole('button', { name: '開く' })
    await userEvent.click(opener)
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
    expect(document.body.style.overflow).toBe('')
  })

  it('背面の幕を押すと閉じる', async () => {
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)
    await userEvent.click(screen.getByRole('button', { name: '開く' }))
    await userEvent.click(screen.getByTestId('modal-overlay'))
    expect(onClose).toHaveBeenCalled()
  })

  it('Tab でのフォーカスはダイアログの中を巡回する', async () => {
    render(<Harness />)
    await userEvent.click(screen.getByRole('button', { name: '開く' }))
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'はい' })).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'いいえ' })).toHaveFocus()
    await userEvent.tab()
    expect(screen.getByRole('button', { name: 'はい' })).toHaveFocus()
    await userEvent.tab({ shift: true })
    expect(screen.getByRole('button', { name: 'いいえ' })).toHaveFocus()
  })
})
