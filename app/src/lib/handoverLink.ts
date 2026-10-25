/** URL で受け渡すトークンの種類。6桁のコードは手入力なので URL にしない */
export type HandoverUrlKind = 'qr' | 'link'

/** フラグメントの名前。受け渡しQRは q、完了リンクは t（ADR 0025） */
const fragmentKey: Record<HandoverUrlKind, string> = { qr: 'q', link: 't' }

const completePath = (dealId: string) => `/deals/${encodeURIComponent(dealId)}/complete`

/**
 * 受け渡しQR・完了リンクの URL `{origin}/deals/{取引ID}/complete#q=<トークン>`（QR）・`#t=<トークン>`（リンク）を組み立てる（DEAL-3）。
 * 中身は取引IDと使い捨てのトークンだけ。フラグメントに載せるのは、サーバーのアクセスログやリファラーに残さないため（ADR 0024）
 */
export function buildHandoverUrl(
  origin: string,
  dealId: string,
  kind: HandoverUrlKind,
  token: string,
): string {
  return `${origin}${completePath(dealId)}#${fragmentKey[kind]}=${encodeURIComponent(token)}`
}

/** 完了の画面の location.hash（`#q=…`・`#t=…`）から種類とトークンを取り出す。どちらでもなければ null */
export function parseHandoverFragment(
  hash: string,
): { kind: HandoverUrlKind; token: string } | null {
  const params = new URLSearchParams(hash.replace(/^#/, ''))
  for (const kind of ['qr', 'link'] as const) {
    const token = params.get(fragmentKey[kind])
    if (token) return { kind, token }
  }
  return null
}

/** アプリ内で読み取った QR の文字列が、この取引の受け渡しQR（`/deals/{dealId}/complete#q=…`）ならトークンを返す。違えば null */
export function parseHandoverQr(text: string, dealId: string): string | null {
  let url: URL
  try {
    url = new URL(text)
  } catch {
    return null
  }
  if (url.pathname !== completePath(dealId)) return null
  const parsed = parseHandoverFragment(url.hash)
  return parsed?.kind === 'qr' ? parsed.token : null
}
