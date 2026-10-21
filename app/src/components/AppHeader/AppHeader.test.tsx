import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { Badge } from '../Badge/Badge'
import { AppHeader } from './AppHeader'

function renderHeader(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

describe('AppHeader', () => {
  it('タイトルを h1 で、行き先のある戻るボタンをリンクで表示する', () => {
    renderHeader(<AppHeader title="受け渡し" backTo="/deals" right={<Badge>取引中</Badge>} />)
    expect(screen.getByRole('heading', { level: 1, name: '受け渡し' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '戻る' })).toHaveAttribute('href', '/deals')
    expect(screen.getByText('取引中')).toBeInTheDocument()
  })

  it('onBack を渡すと戻るボタンになり、押すと呼ばれる', async () => {
    const onBack = vi.fn()
    renderHeader(<AppHeader title="売上管理" onBack={onBack} />)
    await userEvent.click(screen.getByRole('button', { name: '戻る' }))
    expect(onBack).toHaveBeenCalled()
  })

  it('行き先も onBack もなければ戻るボタンを出さない', () => {
    renderHeader(<AppHeader title="取引" />)
    expect(screen.queryByRole('link', { name: '戻る' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '戻る' })).not.toBeInTheDocument()
  })

  it('closable では戻るボタンを「閉じる」にする', () => {
    renderHeader(<AppHeader title="出品する" backTo="/" closable />)
    expect(screen.getByRole('link', { name: '閉じる' })).toHaveAttribute('href', '/')
    expect(screen.queryByRole('link', { name: '戻る' })).not.toBeInTheDocument()
  })

  it('bordered={false} では白地と下線を付けない', () => {
    renderHeader(<AppHeader backTo="/signup" bordered={false} />)
    expect(screen.getByRole('banner').className).not.toMatch(/bordered/)
  })
})
