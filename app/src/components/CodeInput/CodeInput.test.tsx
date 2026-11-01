import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { CodeInput } from './CodeInput'

function Controlled(props: Partial<React.ComponentProps<typeof CodeInput>>) {
  const [value, setValue] = useState(props.value ?? '')
  return <CodeInput {...props} value={value} onChange={setValue} />
}

const box = (n: number) => screen.getByRole('textbox', { name: `${n}桁目` })
const code = () => [1, 2, 3, 4, 5, 6].map((n) => (box(n) as HTMLInputElement).value).join('')

describe('CodeInput', () => {
  it('6マスを「n桁目」の名前で、認証コードのまとまりとして表示する', () => {
    render(<Controlled />)
    expect(screen.getByRole('group', { name: '認証コード' })).toBeInTheDocument()
    expect(screen.getAllByRole('textbox')).toHaveLength(6)
    expect(box(1)).toHaveAttribute('autocomplete', 'one-time-code')
    expect(box(1)).toHaveAttribute('inputmode', 'numeric')
  })

  it('showLabel ならまとまりの名前を画面にも出す', () => {
    render(<Controlled label="メールに届いた6桁のコード" showLabel />)
    const legend = screen.getByText('メールに届いた6桁のコード')
    expect(legend.tagName).toBe('LEGEND')
    expect(legend).not.toHaveClass('visually-hidden')
  })

  it('入力すると次のマスへ進み、数字以外は受け付けない', async () => {
    render(<Controlled />)
    await userEvent.click(box(1))
    await userEvent.keyboard('4a81')
    expect(code()).toBe('481')
    expect(box(4)).toHaveFocus()
  })

  it('空のマスで Backspace を押すと、前のマスを消して戻る', async () => {
    render(<Controlled value="481" />)
    await userEvent.click(box(4))
    await userEvent.keyboard('{Backspace}')
    expect(code()).toBe('48')
    expect(box(3)).toHaveFocus()
  })

  it('先のマスを押しても、空いている最初のマスから入力させる', async () => {
    render(<Controlled value="4" />)
    await userEvent.click(box(5))
    expect(box(2)).toHaveFocus()
  })

  it('貼り付けた6桁を各マスに振り分ける', async () => {
    render(<Controlled />)
    await userEvent.click(box(1))
    await userEvent.paste('４８1-234')
    expect(code()).toBe('481234')
    expect(box(6)).toHaveFocus()
  })

  it('エラーを role="alert" で示し、各マスと結び付ける', () => {
    render(<Controlled error="コードが違います。" />)
    expect(screen.getByRole('alert')).toHaveTextContent('コードが違います。')
    expect(box(1)).toHaveAttribute('aria-invalid', 'true')
    expect(box(1)).toHaveAccessibleDescription('コードが違います。')
  })

  it('有効期限と再送ボタンを表示し、待ち時間中は押せない', async () => {
    const onClick = vi.fn()
    const { rerender } = render(
      <CodeInput
        value=""
        onChange={() => {}}
        expiresIn="9:41"
        resend={{ label: 'コードを再送（0:42 後）', onClick, disabled: true }}
      />,
    )
    expect(screen.getByText('9:41')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'コードを再送（0:42 後）' })).toBeDisabled()

    rerender(<CodeInput value="" onChange={() => {}} resend={{ label: 'コードを再送', onClick }} />)
    await userEvent.click(screen.getByRole('button', { name: 'コードを再送' }))
    expect(onClick).toHaveBeenCalled()
  })
})
