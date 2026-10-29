import { isApiError } from '../../api/client'

const unexpectedMessage = '読み込めませんでした。もう一度お試しください'

/** API の失敗はサーバーの文を、それ以外は決まった文を出す */
export function messageOf(err: unknown): string {
  return isApiError(err) ? err.message : unexpectedMessage
}
