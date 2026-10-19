import { getDeviceId } from '../lib/deviceId'
import type { components } from './schema.gen'

export type ApiErrorBody = components['schemas']['Error']

/**
 * API-1 の共通エラーコード（server/internal/apperr と同じ）。機能固有のコードは string として受ける。
 * `network_error` だけはフロント専用で、サーバーに届かなかった（オフライン・CORS・DNS など）ことを表す。
 */
export type ApiErrorCode =
  | 'bad_request'
  | 'unauthorized'
  | 'forbidden'
  | 'not_found'
  | 'method_not_allowed'
  | 'invalid_status_transition'
  | 'too_many_requests'
  | 'internal_error'
  | 'network_error'
  | (string & {})

// 応答本文が Error スキーマでないとき（プロキシの HTML、空の本文など）に使う既定値。文言はサーバーの apperr と揃える
const fallbacks: Record<number, { code: ApiErrorCode; message: string }> = {
  400: { code: 'bad_request', message: '入力内容に誤りがあります' },
  401: { code: 'unauthorized', message: 'ログインが必要です' },
  403: { code: 'forbidden', message: 'この操作は許可されていません' },
  404: { code: 'not_found', message: '見つかりません' },
  405: { code: 'method_not_allowed', message: 'この操作には対応していません' },
  409: {
    code: 'invalid_status_transition',
    message: '状態が変わったため操作できませんでした。画面を更新してください',
  },
  429: { code: 'too_many_requests', message: 'しばらく時間をおいてからもう一度お試しください' },
  500: {
    code: 'internal_error',
    message: 'サーバーでエラーが発生しました。時間をおいてもう一度お試しください',
  },
}

const networkFallback = {
  code: 'network_error',
  message: '通信できませんでした。接続を確認してもう一度お試しください',
} as const

/** status から既定のコードと文言を決める。表にない 4xx は bad_request、5xx は internal_error（API-1） */
function fallbackFor(status: number): { code: ApiErrorCode; message: string } {
  return fallbacks[status] ?? (status < 500 ? fallbacks[400] : fallbacks[500])
}

function isErrorBody(value: unknown): value is ApiErrorBody {
  if (typeof value !== 'object' || value === null) return false
  const v = value as Record<string, unknown>
  return typeof v.code === 'string' && typeof v.message === 'string'
}

/**
 * API 呼び出しの失敗を表す。code と message は必ず入っており、message はそのまま利用者に表示できる。
 * status はサーバーの HTTP ステータス（通信自体に失敗したときは 0）。
 */
export class ApiError extends Error {
  readonly status: number
  readonly code: ApiErrorCode

  constructor(status: number, code: ApiErrorCode, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }

  /** エラー応答から作る。本文が Error スキーマでなければ status から補う */
  static fromResponse(status: number, body: unknown): ApiError {
    const { code, message } = isErrorBody(body) ? body : fallbackFor(status)
    return new ApiError(status, code, message)
  }

  static network(): ApiError {
    return new ApiError(0, networkFallback.code, networkFallback.message)
  }
}

export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError
}

type UnauthorizedListener = () => void

const unauthorizedListeners = new Set<UnauthorizedListener>()

/**
 * セッションが無効（401 unauthorized）の応答を受けたときに呼ぶ処理を登録する。戻り値で登録を解除する。
 * ログイン状態（features/auth/session）が未ログインに戻すために使う。
 * ログインのパスワード違い（401 invalid_credentials）は code が違うので呼ばない（AUTHN-3）。
 */
export function onUnauthorized(listener: UnauthorizedListener): () => void {
  unauthorizedListeners.add(listener)
  return () => {
    unauthorizedListeners.delete(listener)
  }
}

/**
 * /api 配下を呼ぶ fetch の薄いラッパー。Cookie は同一オリジンの __session だけを送り、
 * 端末識別子を X-Device-Id ヘッダーで付ける（AUTHN-3）。
 * 2xx 以外と通信の失敗は ApiError を投げる。401 unauthorized なら onUnauthorized の処理を呼んでから投げる。204 など本文のない成功は undefined を返す。
 */
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res: Response
  try {
    res = await fetch(`/api${path}`, {
      ...init,
      credentials: 'same-origin',
      headers: { Accept: 'application/json', 'X-Device-Id': getDeviceId(), ...init.headers },
    })
  } catch (err) {
    // 呼び出し側が中断したときは AbortError をそのまま返す
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw ApiError.network()
  }

  const text = await res.text()
  let body: unknown = undefined
  if (text !== '') {
    try {
      body = JSON.parse(text)
    } catch {
      body = null
    }
  }

  if (!res.ok) {
    const err = ApiError.fromResponse(res.status, body)
    if (err.code === 'unauthorized') unauthorizedListeners.forEach((listener) => listener())
    throw err
  }
  return body as T
}

/** 本文を JSON にして POST する（request の薄いラッパー） */
export function postJson<T>(path: string, body: unknown, signal?: AbortSignal): Promise<T> {
  return request<T>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal,
  })
}
