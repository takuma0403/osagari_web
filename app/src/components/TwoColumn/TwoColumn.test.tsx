import { render, screen } from '@testing-library/react'
import { TwoColumn } from './TwoColumn'

describe('TwoColumn', () => {
  it('start・end の順に中身を出す', () => {
    render(<TwoColumn start={<p>左</p>} end={<p>右</p>} />)
    const left = screen.getByText('左')
    expect(left.compareDocumentPosition(screen.getByText('右'))).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
  })

  it('end がなければ start だけを出す', () => {
    const { container } = render(<TwoColumn start={<p>左</p>} />)
    expect(container.firstElementChild?.children).toHaveLength(1)
  })
})
