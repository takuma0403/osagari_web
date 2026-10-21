import { formatPrice } from './format'

type ShareContent = {
  title: string
  /** 円の整数 */
  price: number
  /** 価格の後ろの括弧に入れる状態（書き込みなし・目立った傷なしなど） */
  note: string
  url: string
}

/**
 * 共有リンクに添える文「【お下がり】商品名 ¥価格（状態） URL」（BROWSE, FE）。
 * SPA のためリンクのプレビュー（OGP）が商品ごとに出ないので、貼るだけで内容が伝わるようにする。
 */
export function shareText({ title, price, note, url }: ShareContent): string {
  return `【お下がり】${title} ${formatPrice(price)}（${note}） ${url}`
}

/** 文をクリップボードにコピーする。使えない環境・拒否されたときは reject する */
export function copyText(text: string): Promise<void> {
  if (!navigator.clipboard) return Promise.reject(new Error('clipboard unavailable'))
  return navigator.clipboard.writeText(text)
}
