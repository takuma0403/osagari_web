import { act, render, screen } from '@testing-library/react'
import { IsbnScanner } from './IsbnScanner'
import type { StartScan } from './startIsbnScan'

/** カメラの代わり。emit で読み取った値を流す */
function fakeCamera() {
  let emit: (text: string) => void = () => {}
  const stop = vi.fn()
  const startScan: StartScan = vi.fn(async (_video, onText) => {
    emit = onText
    return { stop }
  })
  return { startScan, stop, emit: (text: string) => act(() => emit(text)) }
}

describe('IsbnScanner', () => {
  it('書籍のバーコードを読み取ったら onDetect に渡し、カメラを止める', async () => {
    const camera = fakeCamera()
    const onDetect = vi.fn()
    render(<IsbnScanner open onClose={vi.fn()} onDetect={onDetect} startScan={camera.startScan} />)
    await act(async () => {})

    camera.emit('4901234567894')
    expect(onDetect).not.toHaveBeenCalled()
    expect(screen.getByRole('status')).toHaveTextContent('ISBN ではないバーコードです')

    camera.emit('9784000000000')
    expect(onDetect).toHaveBeenCalledWith('9784000000000')
    expect(camera.stop).toHaveBeenCalled()
  })

  it('閉じたらカメラを止める', async () => {
    const camera = fakeCamera()
    const { rerender } = render(
      <IsbnScanner open onClose={vi.fn()} onDetect={vi.fn()} startScan={camera.startScan} />,
    )
    await act(async () => {})
    rerender(
      <IsbnScanner
        open={false}
        onClose={vi.fn()}
        onDetect={vi.fn()}
        startScan={camera.startScan}
      />,
    )
    expect(camera.stop).toHaveBeenCalled()
  })

  it('カメラを使えないときは ISBN の入力を案内する', async () => {
    const startScan: StartScan = () => Promise.reject(new DOMException('denied', 'NotAllowedError'))
    render(<IsbnScanner open onClose={vi.fn()} onDetect={vi.fn()} startScan={startScan} />)
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'カメラの使用が許可されていません。ブラウザの設定で許可するか、ISBN を入力してください',
    )
  })
})
