import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { SegmentedTabs } from './SegmentedTabs'

type Kind = 'buy' | 'sell'

function Harness() {
  const [value, setValue] = useState<Kind>('buy')
  return (
    <SegmentedTabs
      label="取引の種類"
      options={[
        { value: 'buy', label: '購入', panelId: 'panel-buy' },
        { value: 'sell', label: '出品', panelId: 'panel-sell' },
      ]}
      value={value}
      onChange={setValue}
    />
  )
}

describe('SegmentedTabs', () => {
  it('tablist の中に、選択中を aria-selected で示すタブを並べる', () => {
    render(<Harness />)
    expect(screen.getByRole('tablist', { name: '取引の種類' })).toBeInTheDocument()
    const buy = screen.getByRole('tab', { name: '購入' })
    expect(buy).toHaveAttribute('aria-selected', 'true')
    expect(buy).toHaveAttribute('aria-controls', 'panel-buy')
    expect(buy).toHaveAttribute('tabindex', '0')
    expect(screen.getByRole('tab', { name: '出品' })).toHaveAttribute('tabindex', '-1')
  })

  it('押したタブを選択する', async () => {
    render(<Harness />)
    await userEvent.click(screen.getByRole('tab', { name: '出品' }))
    expect(screen.getByRole('tab', { name: '出品' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: '購入' })).toHaveAttribute('aria-selected', 'false')
  })

  it('左右キーで選択とフォーカスを移し、端では反対側へ回る', async () => {
    render(<Harness />)
    await userEvent.tab()
    expect(screen.getByRole('tab', { name: '購入' })).toHaveFocus()

    await userEvent.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: '出品' })).toHaveFocus()
    expect(screen.getByRole('tab', { name: '出品' })).toHaveAttribute('aria-selected', 'true')

    await userEvent.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: '購入' })).toHaveAttribute('aria-selected', 'true')
  })
})
