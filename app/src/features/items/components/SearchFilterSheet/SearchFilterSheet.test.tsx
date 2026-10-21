import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SearchFilterSheet, type FilterConditions } from './SearchFilterSheet'

function setup(conditions: FilterConditions = {}) {
  const onApply = vi.fn()
  const onClose = vi.fn()
  render(<SearchFilterSheet conditions={conditions} onApply={onApply} onClose={onClose} />)
  return { onApply, onClose, user: userEvent.setup() }
}

const pressed = (name: string) => screen.getByRole('button', { name, pressed: true })

describe('SearchFilterSheet', () => {
  it('いまの条件を選んだ状態で開く', () => {
    setup({ category: 'textbook', courseName: '線形代数学I', priceMax: 2000, campus: 'honjo' })
    expect(pressed('教科書・参考書')).toBeInTheDocument()
    expect(pressed('本庄')).toBeInTheDocument()
    // 状態は指定なし
    expect(screen.getAllByRole('button', { name: '指定なし', pressed: true })).toHaveLength(1)
    expect(screen.getByLabelText('授業名')).toHaveValue('線形代数学I')
    expect(screen.getByLabelText('価格の上限')).toHaveValue('2000')
  })

  it('選んだ条件を、指定のないものを除いて渡す', async () => {
    const { onApply, user } = setup()
    await user.click(screen.getByRole('button', { name: 'ゲーム・その他' }))
    await user.type(screen.getByLabelText('授業名'), '  基礎数学 ')
    await user.type(screen.getByLabelText('価格の上限'), '1,5００円')
    await user.click(screen.getByRole('button', { name: 'やや傷あり' }))
    await user.click(screen.getByRole('button', { name: 'この条件で検索' }))
    expect(onApply).toHaveBeenCalledWith({
      category: 'game_other',
      courseName: '基礎数学',
      priceMax: 15,
      condition: 'fair',
    })
  })

  it('下限が上限より大きいと検索せず、価格帯にエラーを出す', async () => {
    const { onApply, user } = setup()
    await user.type(screen.getByLabelText('価格の下限'), '3000')
    await user.type(screen.getByLabelText('価格の上限'), '2000')
    await user.click(screen.getByRole('button', { name: 'この条件で検索' }))
    expect(onApply).not.toHaveBeenCalled()
    expect(screen.getByRole('alert')).toHaveTextContent('上限は下限以上にしてください')
    expect(screen.getByLabelText('価格の上限')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('価格の下限')).toHaveAccessibleDescription(
      '上限は下限以上にしてください',
    )

    // 価格を直したらエラーを消す
    await user.clear(screen.getByLabelText('価格の下限'))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('「クリア」で選択を外す（確定するまでは検索しない）', async () => {
    const { onApply, user } = setup({ category: 'textbook', priceMin: 500, campus: 'nabeshima' })
    await user.click(screen.getByRole('button', { name: 'クリア' }))
    expect(onApply).not.toHaveBeenCalled()
    expect(pressed('すべて')).toBeInTheDocument()
    expect(screen.getByLabelText('価格の下限')).toHaveValue('')
    await user.click(screen.getByRole('button', { name: 'この条件で検索' }))
    expect(onApply).toHaveBeenCalledWith({})
  })

  it('Esc で閉じる', async () => {
    const { onClose, user } = setup()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
