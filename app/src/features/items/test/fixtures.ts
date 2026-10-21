import type { ItemList, ItemSummary } from '../api'

/** 一覧・検索のテストで使う商品 */
export function summary(overrides: Partial<ItemSummary> = {}): ItemSummary {
  return {
    shareId: 'abc123',
    status: 'published',
    title: '線形代数入門',
    category: 'textbook',
    condition: 'good',
    price: 1500,
    campus: 'honjo',
    thumbnail: null,
    courseName: '線形代数学I',
    hasWriting: false,
    publishedAt: '2026-10-22T04:00:00Z',
    ...overrides,
  }
}

export function page(
  items: ItemSummary[],
  nextCursor: string | null = null,
  total = items.length,
): ItemList {
  return { items, total, nextCursor }
}

export const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

/** URL（/api/items?...）ごとに応答を返し分ける fetch を差し込む。呼ばれた URL は calls で見る */
export function stubItemsFetch(responses: Record<string, () => Response | Promise<Response>>) {
  const fetchMock = vi.fn(async (url: string) => {
    const handler = responses[url]
    if (!handler) throw new Error(`想定していない呼び出し: ${url}`)
    return handler()
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}
