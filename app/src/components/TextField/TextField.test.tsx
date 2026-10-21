import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TextField } from './TextField'

describe('TextField', () => {
  it('ラベルと入力欄を結び付ける', () => {
    render(<TextField label="タイトル" defaultValue="線形代数入門" />)
    expect(screen.getByLabelText('タイトル')).toHaveValue('線形代数入門')
  })

  it('エラーを role="alert" で示し、入力欄と結び付ける', () => {
    render(<TextField label="パスワード" error="10文字以上にしてください" />)
    const input = screen.getByLabelText('パスワード')
    expect(screen.getByRole('alert')).toHaveTextContent('10文字以上にしてください')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAccessibleDescription('10文字以上にしてください')
  })

  it('補足も入力欄の説明として読み上げる', () => {
    render(<TextField label="パスワード" hint="10文字以上" />)
    expect(screen.getByLabelText('パスワード')).toHaveAccessibleDescription('10文字以上')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('パスワードは表示・非表示を切り替えられる', async () => {
    render(<TextField label="パスワード" type="password" defaultValue="password12" />)
    const input = screen.getByLabelText('パスワード')
    expect(input).toHaveAttribute('type', 'password')

    await userEvent.click(screen.getByRole('button', { name: 'パスワードを表示' }))
    expect(input).toHaveAttribute('type', 'text')

    await userEvent.click(screen.getByRole('button', { name: 'パスワードを隠す' }))
    expect(input).toHaveAttribute('type', 'password')
  })

  it('前後に固定の文字を置ける', () => {
    render(<TextField label="価格" prefix="¥" suffix="円" numeric />)
    expect(screen.getByText('¥')).toBeInTheDocument()
    expect(screen.getByText('円')).toBeInTheDocument()
  })

  it('hideLabel でもラベルは読み上げに残る', () => {
    render(<TextField label="授業名" hideLabel placeholder="授業名" />)
    expect(screen.getByLabelText('授業名')).toBeInTheDocument()
  })

  it('invalid では文を出さずに枠だけエラーにし、呼び出し側の説明と結び付ける', () => {
    render(
      <>
        <TextField label="価格の上限" invalid aria-describedby="price-error" />
        <p id="price-error">上限は下限以上にしてください</p>
      </>,
    )
    const input = screen.getByLabelText('価格の上限')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAccessibleDescription('上限は下限以上にしてください')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
