import { render, screen } from '@testing-library/react'
import { Badge } from './Badge'

describe('Badge', () => {
  it('既定は primary の小さいバッジ', () => {
    render(<Badge>取引中</Badge>)
    const badge = screen.getByText('取引中')
    expect(badge.className).toMatch(/primary/)
    expect(badge.className).toMatch(/sm/)
  })

  it('色調と大きさを切り替えられる', () => {
    render(
      <Badge tone="warning" size="md">
        決済はモック
      </Badge>,
    )
    const badge = screen.getByText('決済はモック')
    expect(badge.className).toMatch(/warning/)
    expect(badge.className).toMatch(/md/)
  })
})
