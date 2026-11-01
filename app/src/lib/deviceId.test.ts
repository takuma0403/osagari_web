import { DEVICE_ID_STORAGE_KEY, getDeviceId } from './deviceId'

describe('getDeviceId', () => {
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('初回は識別子を作って localStorage に保存する', () => {
    const id = getDeviceId()
    expect(id).toMatch(/^[0-9a-f-]{36}$/)
    expect(localStorage.getItem(DEVICE_ID_STORAGE_KEY)).toBe(id)
  })

  it('保存済みの識別子をそのまま返す', () => {
    localStorage.setItem(DEVICE_ID_STORAGE_KEY, 'saved-id')
    expect(getDeviceId()).toBe('saved-id')
  })

  it('localStorage が使えないときも、ページの間は同じ識別子を返す', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    const first = getDeviceId()
    expect(first).toMatch(/^[0-9a-f-]{36}$/)
    expect(getDeviceId()).toBe(first)
  })
})
