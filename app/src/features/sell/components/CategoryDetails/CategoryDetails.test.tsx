import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { emptySellForm, type SellForm } from '../../form'
import type { StartScan } from '../IsbnScanner/startIsbnScan'
import { CategoryDetails } from './CategoryDetails'

function setup(form: Partial<SellForm>, startScan?: StartScan) {
  const onChange = vi.fn()
  render(
    <CategoryDetails
      form={{ ...emptySellForm, ...form }}
      onChange={onChange}
      errors={{}}
      startScan={startScan}
    />,
  )
  return onChange
}

describe('CategoryDetails', () => {
  it('教科書では授業名・担当教員・ISBN・書き込みの有無を出す', async () => {
    const onChange = setup({ category: 'textbook' })
    expect(screen.getByRole('group', { name: '教科書の情報' })).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('授業名'), 'A')
    expect(onChange).toHaveBeenCalledWith('courseName', 'A')
    await userEvent.click(screen.getByRole('checkbox', { name: '書き込みあり' }))
    expect(onChange).toHaveBeenCalledWith('hasWriting', true)
    expect(screen.queryByLabelText('サイズ')).not.toBeInTheDocument()
  })

  it('家具・家電ではサイズと動作確認を出す', () => {
    setup({ category: 'furniture_appliance' })
    expect(screen.getByRole('group', { name: '家具・家電の情報' })).toBeInTheDocument()
    expect(screen.getByLabelText('サイズ')).toBeInTheDocument()
    expect(screen.queryByLabelText('ISBN')).not.toBeInTheDocument()
  })

  it('ゲーム・その他と未選択では何も出さない', () => {
    setup({ category: 'game_other' })
    expect(screen.queryByRole('group')).not.toBeInTheDocument()
  })

  it('バーコードを読み取ったら ISBN に入れる', async () => {
    let emit: (text: string) => void = () => {}
    const startScan: StartScan = async (_video, onText) => {
      emit = onText
      return { stop: () => {} }
    }
    const onChange = setup({ category: 'textbook' }, startScan)
    await userEvent.click(screen.getByRole('button', { name: 'バーコードを読み取る' }))
    expect(screen.getByRole('dialog', { name: 'バーコードを読み取る' })).toBeInTheDocument()
    act(() => emit('9784000000000'))
    expect(onChange).toHaveBeenCalledWith('isbn', '9784000000000')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
