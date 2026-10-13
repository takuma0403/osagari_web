import { render } from '@testing-library/react'
import { AppLogo } from './AppLogo'

describe('AppLogo', () => {
  it('指定した大きさの枠に、装飾画像としてロゴを表示する', () => {
    const { container } = render(<AppLogo size={30} />)
    const frame = container.firstElementChild as HTMLElement
    expect(frame).toHaveStyle({ width: '30px', height: '30px' })
    expect(container.querySelector('img')).toHaveAttribute('alt', '')
  })
})
