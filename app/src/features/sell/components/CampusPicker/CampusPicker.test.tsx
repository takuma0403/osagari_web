import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CampusPicker } from './CampusPicker'

describe('CampusPicker', () => {
  it('ラジオボタンのグループとして選択を伝える', async () => {
    const onChange = vi.fn()
    render(<CampusPicker value="honjo" onChange={onChange} />)
    const group = screen.getByRole('group', { name: '受け渡し希望キャンパス' })
    expect(group).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: '本庄' })).toBeChecked()
    await userEvent.click(screen.getByRole('radio', { name: '鍋島' }))
    expect(onChange).toHaveBeenCalledWith('nabeshima')
  })

  it('エラーを role="alert" で出す', () => {
    render(
      <CampusPicker value="" onChange={vi.fn()} error="受け渡し希望キャンパスを選んでください" />,
    )
    expect(screen.getByRole('group')).toHaveAccessibleDescription(
      '受け渡し希望キャンパスを選んでください',
    )
  })
})
