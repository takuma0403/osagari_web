import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SelectField } from './SelectField'

const options = [
  { value: 'new_like', label: '新品同様' },
  { value: 'good', label: '目立った傷なし' },
]

describe('SelectField', () => {
  it('ラベルと結び付け、選んだ値で onChange を呼ぶ', async () => {
    const onChange = vi.fn()
    render(
      <SelectField
        label="状態"
        options={options}
        value=""
        placeholder="選択してください"
        onChange={onChange}
      />,
    )
    const select = screen.getByLabelText('状態')
    expect(select).toHaveValue('')
    await userEvent.selectOptions(select, '目立った傷なし')
    expect(onChange).toHaveBeenCalledWith('good')
  })

  it('エラーを role="alert" で示し、選択欄と結び付ける', () => {
    render(
      <SelectField
        label="状態"
        options={options}
        value=""
        onChange={vi.fn()}
        error="状態を選んでください"
      />,
    )
    const select = screen.getByLabelText('状態')
    expect(select).toHaveAttribute('aria-invalid', 'true')
    expect(select).toHaveAccessibleDescription('状態を選んでください')
  })
})
