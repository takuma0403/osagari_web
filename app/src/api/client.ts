import type { components } from './schema.gen'

export type ApiErrorBody = components['schemas']['Error']

/** API がエラー（2xx 以外）を返したときに投げる。body は Error スキーマ（読めなければ null） */
export class ApiError extends Error {
  readonly status: number
  readonly body: ApiErrorBody | null

  constructor(status: number, body: ApiErrorBody | null) {
    super(body?.message ?? `API エラー (${status})`)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

/** /api 配下を呼ぶ fetch の薄いラッパー。Cookie は同一オリジンの __session だけを送る（AUTHN-3） */
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...init,
    credentials: 'same-origin',
    headers: { Accept: 'application/json', ...init.headers },
  })
  const body: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    throw new ApiError(res.status, body as ApiErrorBody | null)
  }
  return body as T
}
