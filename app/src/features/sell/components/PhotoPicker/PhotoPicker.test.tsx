import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PhotoPicker } from './PhotoPicker'

const photos = [
  { id: 'a', url: 'blob:a', objectKey: 'u1/a' },
  { id: 'b', url: 'blob:b', objectKey: 'u1/b' },
]

function setup(props: Partial<Parameters<typeof PhotoPicker>[0]> = {}) {
  const handlers = { onAdd: vi.fn(), onRemove: vi.fn(), onMove: vi.fn() }
  render(<PhotoPicker photos={photos} {...handlers} {...props} />)
  return handlers
}

describe('PhotoPicker', () => {
  it('選んだ写真をまとめて onAdd に渡す', async () => {
    const { onAdd } = setup()
    const files = [
      new File(['1'], '1.jpg', { type: 'image/jpeg' }),
      new File(['2'], '2.jpg', { type: 'image/jpeg' }),
    ]
    await userEvent.upload(screen.getByLabelText('商品の写真'), files)
    expect(onAdd).toHaveBeenCalledWith(files)
  })

  it('写真を押すとシートで並べ替え・削除を選べる。先頭の写真は前へ移動できない', async () => {
    const { onMove, onRemove } = setup()
    await userEvent.click(screen.getByRole('button', { name: '写真1を並べ替え・削除' }))
    const sheet = screen.getByRole('dialog', { name: '写真1' })
    expect(within(sheet).getByRole('button', { name: '前へ移動' })).toBeDisabled()
    await userEvent.click(within(sheet).getByRole('button', { name: '後ろへ移動' }))
    expect(onMove).toHaveBeenCalledWith('a', 1)

    await userEvent.click(screen.getByRole('button', { name: '写真2を並べ替え・削除' }))
    await userEvent.click(screen.getByRole('button', { name: '削除する' }))
    expect(onRemove).toHaveBeenCalledWith('b')
  })

  it('アップロード中の写真は押せない', () => {
    setup({ photos: [{ id: 'a', url: 'blob:a' }] })
    expect(screen.getByRole('button', { name: '写真1（アップロード中）' })).toBeDisabled()
  })

  it('10枚そろうと「写真を追加」を出さない', () => {
    const ten = Array.from({ length: 10 }, (_, i) => ({
      id: `${i}`,
      url: `blob:${i}`,
      objectKey: `u1/${i}`,
    }))
    setup({ photos: ten })
    expect(screen.queryByRole('button', { name: '写真を追加' })).not.toBeInTheDocument()
    expect(screen.getByText('10枚まで（10/10）。1枚目が表紙になります')).toBeInTheDocument()
  })

  it('エラーを role="alert" で出す', () => {
    setup({ error: '写真を1枚以上追加してください' })
    expect(screen.getByRole('alert')).toHaveTextContent('写真を1枚以上追加してください')
  })
})
