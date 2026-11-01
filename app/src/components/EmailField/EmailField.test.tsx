import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { EmailField } from './EmailField'

function Controlled({ initial = '' }: { initial?: string }) {
  const [value, setValue] = useState(initial)
  return <EmailField value={value} onChange={setValue} />
}

describe('EmailField', () => {
  it('ラベル付きの数字入力に、固定のドメインを続けて表示する', () => {
    render(<Controlled />)
    const input = screen.getByLabelText('大学のメールアドレス')
    expect(input).toHaveAttribute('inputmode', 'numeric')
    expect(input).toHaveAttribute('autocomplete', 'username')
    expect(screen.getByText('@edu.cc.saga-u.ac.jp')).toBeInTheDocument()
  })

  it('数字以外は受け付けず、8桁までにする', async () => {
    render(<Controlled />)
    const input = screen.getByLabelText('大学のメールアドレス')
    await userEvent.type(input, '26a2-38001999')
    expect(input).toHaveValue('26238001')
  })

  it('アドレスごと貼り付けたら学籍番号だけを残す', async () => {
    render(<Controlled />)
    const input = screen.getByLabelText('大学のメールアドレス')
    await userEvent.click(input)
    await userEvent.paste('26238001@edu.cc.saga-u.ac.jp')
    expect(input).toHaveValue('26238001')
  })

  it('エラーを表示できる', () => {
    render(<EmailField value="2623" onChange={() => {}} error="学籍番号8桁を入力してください" />)
    expect(screen.getByRole('alert')).toHaveTextContent('学籍番号8桁を入力してください')
  })
})
