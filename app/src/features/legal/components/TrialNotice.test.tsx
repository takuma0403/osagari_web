import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, Link, RouterProvider } from 'react-router'
import { TRIAL_NOTICE_STORAGE_KEY } from '../../../lib/trialNotice'
import { RootLayout } from './TrialNotice'

// RootLayout の下に仮の画面を置き、画面の上にダイアログが重なるかを確かめる
function setup(path = '/') {
  const router = createMemoryRouter(
    [
      {
        element: <RootLayout />,
        children: [
          { path: '/', element: <Link to="/terms">利用規約</Link> },
          { path: '/terms', element: <Link to="/">戻る</Link> },
          { path: '/privacy', element: <p>プライバシーポリシー</p> },
        ],
      },
    ],
    { initialEntries: [path] },
  )
  const view = render(<RouterProvider router={router} />)
  return { router, view, user: userEvent.setup() }
}

describe('TrialNotice', () => {
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('初回は利用規約 第２条の要旨をダイアログで出す', () => {
    setup()
    const dialog = screen.getByRole('dialog', { name: '試験運用中です' })
    expect(dialog).toHaveTextContent('実際のお金の支払い・受け取りは発生しません')
    expect(dialog).toHaveTextContent('実際の売買（お金のやり取りを伴う取引）に使わないでください')
  })

  it('「確認した」で閉じて保存し、開き直しても出さない', async () => {
    const { view, user } = setup()
    await user.click(screen.getByRole('button', { name: '確認した' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(localStorage.getItem(TRIAL_NOTICE_STORAGE_KEY)).toBe('1')

    view.unmount()
    setup()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('Esc・背面の幕では閉じない', async () => {
    const { user } = setup()
    await user.keyboard('{Escape}')
    await user.click(screen.getByTestId('modal-overlay'))
    expect(screen.getByRole('dialog', { name: '試験運用中です' })).toBeInTheDocument()
    expect(localStorage.getItem(TRIAL_NOTICE_STORAGE_KEY)).toBeNull()
  })

  it.each(['/terms', '/privacy'])('%s を開いているときは重ねない', (path) => {
    setup(path)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('規約から戻ってきて、まだ確認していなければ出す', async () => {
    const { router } = setup('/terms')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await router.navigate('/')
    expect(await screen.findByRole('dialog', { name: '試験運用中です' })).toBeInTheDocument()
  })

  it('localStorage が使えなくても画面を描き、「確認した」で閉じられる', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    const { user } = setup()
    expect(screen.getByText('利用規約')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '確認した' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
