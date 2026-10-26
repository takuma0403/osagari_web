const yen = new Intl.NumberFormat('ja-JP')

/**
 * 金額（円の整数）を `¥1,500` の形にする（PAY-2）。表示は Outfit フォント（--font-num）で行う。
 * 金額の計算はサーバーの値を使い、ここでは表示だけを担う。
 */
export function formatPrice(amount: number): string {
  return `¥${yen.format(amount)}`
}

/**
 * 日付（`2030-03-31` の形の文字列）を `2030年3月31日` の形にする。
 * 時刻を持たない日付なので、タイムゾーンで日がずれないよう文字列のまま分解する。
 */
export function formatDate(isoDate: string): string {
  const [year, month, day] = isoDate.split('-').map(Number)
  return `${year}年${month}月${day}日`
}

/** 日時（ISO 8601）を端末の時刻で `10/8 14:02` の形にする。提出・審査した日時など、年を省いてよい表示に使う */
export function formatShortDateTime(isoDateTime: string): string {
  const d = new Date(isoDateTime)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${hh}:${mm}`
}

/** 日時（ISO 8601）を端末の日付で `9/28` の形にする。取引一覧の受け渡し日など、日だけでよい表示に使う */
export function formatMonthDay(isoDateTime: string): string {
  const d = new Date(isoDateTime)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

/** 残り秒数を `9:41` の形（分:秒）にする。認証コードの有効期限や再送までの待ち時間に使う */
export function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.ceil(seconds))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

/** 残り秒数を `23:58` の形（時:分。分の端数は切り上げ）にする。完了リンク・完了コードの有効期限（24時間、DEAL-3）に使う */
export function formatCountdownHours(seconds: number): string {
  const m = Math.max(0, Math.ceil(seconds / 60))
  return `${Math.floor(m / 60)}:${String(m % 60).padStart(2, '0')}`
}

/** 6桁の確認コード・完了コードを、読み上げやすいよう `381 902` のように3桁で区切る（DEAL-3） */
export function formatHandoverCode(code: string): string {
  return code.length === 6 ? `${code.slice(0, 3)} ${code.slice(3)}` : code
}

const HOUR_MS = 60 * 60 * 1000
const MINUTE_MS = 60 * 1000

/**
 * 期限までの残りを `24時間` の形にする（端数は切り捨て）。1時間未満は `42分`、過ぎていたら `0分`。
 * 購入希望の承認期限（DEAL-1、48時間）の表示に使う。
 */
export function formatRemaining(deadline: string, now: number = Date.now()): string {
  const ms = Math.max(0, new Date(deadline).getTime() - now)
  if (ms >= HOUR_MS) return `${Math.floor(ms / HOUR_MS)}時間`
  return `${Math.floor(ms / MINUTE_MS)}分`
}
