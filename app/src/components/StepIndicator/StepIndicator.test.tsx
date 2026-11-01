import { render, screen } from '@testing-library/react'
import { StepIndicator } from './StepIndicator'

describe('StepIndicator', () => {
  it('いまの手順と全体の数を表示する', () => {
    render(<StepIndicator current={2} total={3} />)
    expect(screen.getByText('STEP 2 / 3')).toBeInTheDocument()
  })
})
