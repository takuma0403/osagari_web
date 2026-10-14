import { render, screen } from '@testing-library/react'
import { InfoBox } from './InfoBox'

describe('InfoBox', () => {
  it('文を色調付きの枠で表示する', () => {
    render(<InfoBox tone="warning">変更すると、ほかの端末ではログアウトされます。</InfoBox>)
    const text = screen.getByText('変更すると、ほかの端末ではログアウトされます。')
    expect(text.parentElement?.className).toMatch(/warning/)
  })

  it('role を渡すと、エラーとして読み上げられる', () => {
    render(
      <InfoBox tone="danger" icon="lock" role="alert">
        15分間ログインできません。
      </InfoBox>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('15分間ログインできません。')
  })
})
