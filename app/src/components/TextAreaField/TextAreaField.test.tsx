import { render, screen } from '@testing-library/react'
import { TextAreaField } from './TextAreaField'

describe('TextAreaField', () => {
  it('ラベルと結び付け、補足とエラーを説明として読み上げる', () => {
    render(
      <TextAreaField
        label="説明"
        defaultValue="1年前期で使いました。"
        hint="2,000文字まで"
        error="説明を入力してください"
      />,
    )
    const textarea = screen.getByLabelText('説明')
    expect(textarea).toHaveValue('1年前期で使いました。')
    expect(textarea).toHaveAttribute('aria-invalid', 'true')
    expect(textarea).toHaveAccessibleDescription('2,000文字まで 説明を入力してください')
    expect(screen.getByRole('alert')).toHaveTextContent('説明を入力してください')
  })
})
