import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { Button, ButtonLink } from './Button'

describe('Button', () => {
  it('既定は type="button" の主要ボタン（52px）', () => {
    render(<Button>ログイン</Button>)
    const button = screen.getByRole('button', { name: 'ログイン' })
    expect(button).toHaveAttribute('type', 'button')
    expect(button.className).toMatch(/primary/)
    expect(button.className).toMatch(/lg/)
  })

  it('種類と高さを切り替えられる', () => {
    render(
      <Button variant="ghost" size="sm" fullWidth>
        あとで
      </Button>,
    )
    const button = screen.getByRole('button', { name: 'あとで' })
    expect(button.className).toMatch(/ghost/)
    expect(button.className).toMatch(/sm/)
    expect(button.className).toMatch(/fullWidth/)
  })

  it('押すと onClick が呼ばれ、無効のときは呼ばれない', async () => {
    const onClick = vi.fn()
    const { rerender } = render(<Button onClick={onClick}>送る</Button>)
    await userEvent.click(screen.getByRole('button', { name: '送る' }))
    expect(onClick).toHaveBeenCalledTimes(1)

    rerender(
      <Button onClick={onClick} disabled>
        送る
      </Button>,
    )
    await userEvent.click(screen.getByRole('button', { name: '送る' }))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('アイコンは読み上げず、文字だけを名前にする', () => {
    render(<Button icon="copy">リンクをコピー</Button>)
    expect(screen.getByRole('button', { name: 'リンクをコピー' })).toBeInTheDocument()
  })
})

describe('ButtonLink', () => {
  it('ボタンの見た目のリンクになる', () => {
    render(
      <MemoryRouter>
        <ButtonLink to="/signup" variant="secondary">
          新規登録
        </ButtonLink>
      </MemoryRouter>,
    )
    const link = screen.getByRole('link', { name: '新規登録' })
    expect(link).toHaveAttribute('href', '/signup')
    expect(link.className).toMatch(/secondary/)
  })
})
