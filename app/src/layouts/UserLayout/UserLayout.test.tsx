import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { MobileLayout } from './MobileLayout'

function renderLayout(bottomNav?: ReactNode) {
  const router = createMemoryRouter([
    {
      element: <MobileLayout bottomNav={bottomNav} />,
      children: [{ path: '/', element: <p>本文</p> }],
    },
  ])
  render(<RouterProvider router={router} />)
}

describe('MobileLayout', () => {
  it('子ルートを main の中に表示する', () => {
    renderLayout()
    expect(screen.getByRole('main')).toHaveTextContent('本文')
  })

  it('bottomNav を渡したときだけ下部に表示する', () => {
    renderLayout(<nav aria-label="下部ナビ" />)
    expect(screen.getByRole('navigation', { name: '下部ナビ' })).toBeInTheDocument()
  })

  it('bottomNav がなければナビを出さない', () => {
    renderLayout()
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  })
})
