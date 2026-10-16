/** 端末識別子を保存する localStorage のキー（AUTHN-3） */
export const DEVICE_ID_STORAGE_KEY = 'osagari.deviceId'

// localStorage が使えない（プライベートモードなどで例外になる）ときに、このページの間だけ使う値
let fallbackId: string | undefined

/**
 * この端末の識別子を返す（AUTHN-3）。初回は乱数で作って localStorage に保存し、以後は同じ値を返す。
 * Firebase Hosting は `__session` 以外の Cookie を落とすため、Cookie ではなく localStorage に置き、
 * API には `X-Device-Id` ヘッダーで送る。保存できないときは毎回「初めての端末」として扱われる。
 */
export function getDeviceId(): string {
  try {
    const saved = localStorage.getItem(DEVICE_ID_STORAGE_KEY)
    if (saved) return saved
    const id = crypto.randomUUID()
    localStorage.setItem(DEVICE_ID_STORAGE_KEY, id)
    return id
  } catch {
    fallbackId ??= crypto.randomUUID()
    return fallbackId
  }
}
