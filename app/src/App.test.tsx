import { render, screen } from '@testing-library/react'
import App from './App'

describe('App', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('API の稼働状況を表示する', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: 'ok' }), { status: 200 })),
    )
    render(<App />)
    expect(await screen.findByText('API: ok')).toBeInTheDocument()
  })

  it('API に接続できないときはその旨を表示する', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('network')))
    render(<App />)
    expect(await screen.findByText('API: API に接続できません')).toBeInTheDocument()
  })
})
