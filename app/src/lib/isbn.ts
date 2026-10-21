/** ISBN からハイフンと空白を除き、大文字にする（ISBN-10 の末尾の X。サーバーの保存形式と同じ） */
export function normalizeIsbn(value: string): string {
  return value.replace(/[-\s]/g, '').toUpperCase()
}

/**
 * ハイフンを除いた ISBN-10 か ISBN-13 のチェックディジットが正しいかを返す（ITEM）。
 * server の domain.ValidISBN と同じ判定。最終的な検証はサーバーが行う。
 */
export function isValidIsbn(isbn: string): boolean {
  if (/^\d{9}[\dX]$/.test(isbn)) {
    const sum = [...isbn].reduce((acc, c, i) => acc + (c === 'X' ? 10 : Number(c)) * (10 - i), 0)
    return sum % 11 === 0
  }
  if (/^\d{13}$/.test(isbn)) {
    const sum = [...isbn].reduce((acc, c, i) => acc + Number(c) * (i % 2 === 1 ? 3 : 1), 0)
    return sum % 10 === 0
  }
  return false
}

/** バーコード（EAN-13）の値が書籍の ISBN（978・979 で始まる）かを返す */
export function isBookBarcode(value: string): boolean {
  return /^97[89]\d{10}$/.test(value) && isValidIsbn(value)
}
