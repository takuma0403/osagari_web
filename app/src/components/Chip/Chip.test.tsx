import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Chip } from './Chip'

describe('Chip', () => {
  it('選択中かどうかを aria-pressed で示す', () => {
    render(
      <>
        <Chip label="すべて" selected />
        <Chip label="家具・家電" selected={false} />
      </>,
    )
    expect(screen.getByRole('button', { name: 'すべて' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: '家具・家電' })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  })

  it('押すと onClick が呼ばれる', async () => {
    const onClick = vi.fn()
    render(<Chip label="絞り込み" icon="filter" onClick={onClick} />)
    await userEvent.click(screen.getByRole('button', { name: '絞り込み' }))
    expect(onClick).toHaveBeenCalled()
  })

  it('絞り込み条件のチップは「〇〇を外す」ボタンになる', async () => {
    const onRemove = vi.fn()
    render(<Chip label="本庄" onRemove={onRemove} />)
    await userEvent.click(screen.getByRole('button', { name: '本庄を外す' }))
    expect(onRemove).toHaveBeenCalled()
  })
})
