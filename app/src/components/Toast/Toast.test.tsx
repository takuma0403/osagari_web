import { render, screen } from '@testing-library/react'
import { Toast } from './Toast'

describe('Toast', () => {
  it('文を role="status" で読み上げる', () => {
    render(<Toast message="リンクをコピーしました" />)
    expect(screen.getByRole('status')).toHaveTextContent('リンクをコピーしました')
  })

  it('文がないときも読み上げ用の領域だけは置いておく', () => {
    render(<Toast />)
    expect(screen.getByRole('status')).toBeEmptyDOMElement()
  })
})
