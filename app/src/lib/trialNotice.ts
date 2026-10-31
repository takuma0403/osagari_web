/** 試験運用の初回ダイアログを確認したことを保存する localStorage のキー（OVERVIEW、利用規約 第２条） */
export const TRIAL_NOTICE_STORAGE_KEY = 'osagari.trialNoticeAcknowledged'

/**
 * この端末で試験運用の初回ダイアログを確認済みかを返す。
 * Cookie は `__session` 以外を使わないため localStorage に置く（AUTHN-3）。読めないときは未確認として扱う。
 */
export function isTrialNoticeAcknowledged(): boolean {
  try {
    return localStorage.getItem(TRIAL_NOTICE_STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

/** 確認したことを保存する。保存できない（プライベートモードなど）ときは次に開いたときにまた出る */
export function acknowledgeTrialNotice(): void {
  try {
    localStorage.setItem(TRIAL_NOTICE_STORAGE_KEY, '1')
  } catch {
    // 保存できなくても、このページの間は閉じたままにする（呼び出し側の状態で持つ）
  }
}
