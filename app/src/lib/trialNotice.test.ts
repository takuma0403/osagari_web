import {
  acknowledgeTrialNotice,
  isTrialNoticeAcknowledged,
  TRIAL_NOTICE_STORAGE_KEY,
} from './trialNotice'

describe('trialNotice', () => {
  afterEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('初めは未確認で、確認すると localStorage に保存する', () => {
    expect(isTrialNoticeAcknowledged()).toBe(false)
    acknowledgeTrialNotice()
    expect(localStorage.getItem(TRIAL_NOTICE_STORAGE_KEY)).toBe('1')
    expect(isTrialNoticeAcknowledged()).toBe(true)
  })

  it('localStorage が使えないときは未確認として扱い、保存しようとしても例外を出さない', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    expect(isTrialNoticeAcknowledged()).toBe(false)
    expect(() => acknowledgeTrialNotice()).not.toThrow()
  })
})
