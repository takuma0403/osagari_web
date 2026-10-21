import { isValidIsbn, normalizeIsbn } from '../../lib/isbn'
import type { Campus, ItemCategory, ItemCondition, ItemInput, OwnItem } from './api'

/** 出品の入力の制限（ITEM。server の domain と同じ値） */
export const ITEM_TITLE_MAX = 100
export const ITEM_DESCRIPTION_MAX = 2000
export const ITEM_DETAIL_MAX = 100
export const ITEM_PRICE_MIN = 100
export const ITEM_PRICE_MAX = 100000
export const ITEM_IMAGES_MAX = 10

export const categoryOptions: { value: ItemCategory; label: string }[] = [
  { value: 'textbook', label: '教科書・参考書' },
  { value: 'furniture_appliance', label: '家具・家電' },
  { value: 'game_other', label: 'ゲーム・その他' },
]

export const conditionOptions: { value: ItemCondition; label: string }[] = [
  { value: 'new_like', label: '新品同様' },
  { value: 'good', label: '目立った傷なし' },
  { value: 'fair', label: 'やや傷あり' },
]

export const campusOptions: { value: Campus; label: string }[] = [
  { value: 'honjo', label: '本庄' },
  { value: 'nabeshima', label: '鍋島' },
]

/** 入力欄の値。未選択・未入力は空文字で持ち、送るときに省く */
export type SellForm = {
  title: string
  category: ItemCategory | ''
  condition: ItemCondition | ''
  /** 数字だけの文字列 */
  price: string
  description: string
  campus: Campus | ''
  courseName: string
  instructor: string
  isbn: string
  hasWriting: boolean
  size: string
  operationChecked: boolean
}

export type SellField = keyof SellForm | 'photos'

/** 項目ごとのエラー文（server の domain と同じ文言） */
export type SellErrors = Partial<Record<SellField, string>>

export const emptySellForm: SellForm = {
  title: '',
  category: '',
  condition: '',
  price: '',
  description: '',
  campus: '',
  courseName: '',
  instructor: '',
  isbn: '',
  hasWriting: false,
  size: '',
  operationChecked: false,
}

/** 編集する商品から入力欄の値を作る */
export function formFromItem(item: OwnItem): SellForm {
  return {
    title: item.title,
    category: item.category ?? '',
    condition: item.condition ?? '',
    price: item.price == null ? '' : String(item.price),
    description: item.description ?? '',
    campus: item.campus ?? '',
    courseName: item.textbook?.courseName ?? '',
    instructor: item.textbook?.instructor ?? '',
    isbn: item.textbook?.isbn ?? '',
    hasWriting: item.textbook?.hasWriting ?? false,
    size: item.furnitureAppliance?.size ?? '',
    operationChecked: item.furnitureAppliance?.operationChecked ?? false,
  }
}

/** 価格の欄の値を数値にする。数字がなければ undefined */
export function parsePrice(price: string): number | undefined {
  return price === '' ? undefined : Number(price)
}

const optional = (value: string) => value.trim() || undefined

/**
 * 入力欄の値を API の入力にする（ITEM）。空の任意項目は省く。
 * カテゴリと合わない追加項目（家具・家電の ISBN など）はサーバーが受け付けないので送らない。
 */
export function toItemInput(form: SellForm, imageObjectKeys: string[]): ItemInput {
  const isbn = normalizeIsbn(form.isbn)
  return {
    title: form.title.trim(),
    category: form.category || undefined,
    condition: form.condition || undefined,
    price: parsePrice(form.price),
    description: optional(form.description),
    campus: form.campus || undefined,
    imageObjectKeys,
    textbook:
      form.category === 'textbook'
        ? {
            courseName: optional(form.courseName),
            instructor: optional(form.instructor),
            isbn: isbn || undefined,
            hasWriting: form.hasWriting,
          }
        : undefined,
    furnitureAppliance:
      form.category === 'furniture_appliance'
        ? { size: optional(form.size), operationChecked: form.operationChecked }
        : undefined,
  }
}

const length = (value: string) => [...value.trim()].length

/**
 * 入力を確かめる（ITEM, ADR 0016）。下書き（publish: false）はタイトルだけ必須で、入力した項目の形式と範囲を確かめる。
 * 公開（publish: true）はすべての共通項目と写真1枚以上が必須。最終的な検証はサーバーが行う。
 */
export function validateSellForm(form: SellForm, photoCount: number, publish: boolean): SellErrors {
  const errors: SellErrors = {}
  const price = parsePrice(form.price)

  if (length(form.title) === 0) errors.title = 'タイトルを入力してください'
  else if (length(form.title) > ITEM_TITLE_MAX)
    errors.title = 'タイトルは100文字以内で入力してください'

  if (price !== undefined && (price < ITEM_PRICE_MIN || price > ITEM_PRICE_MAX))
    errors.price = '価格は100円から100,000円の範囲で入力してください'
  if (length(form.description) > ITEM_DESCRIPTION_MAX)
    errors.description = '説明は2,000文字以内で入力してください'

  if (form.category === 'textbook') {
    if (length(form.courseName) > ITEM_DETAIL_MAX)
      errors.courseName = '授業名は100文字以内で入力してください'
    if (length(form.instructor) > ITEM_DETAIL_MAX)
      errors.instructor = '担当教員は100文字以内で入力してください'
    const isbn = normalizeIsbn(form.isbn)
    if (isbn !== '' && !isValidIsbn(isbn)) errors.isbn = 'ISBN を確認してください（10桁か13桁）'
  }
  if (form.category === 'furniture_appliance' && length(form.size) > ITEM_DETAIL_MAX)
    errors.size = 'サイズは100文字以内で入力してください'

  if (photoCount > ITEM_IMAGES_MAX) errors.photos = '写真は10枚までです'

  if (publish) {
    if (photoCount === 0) errors.photos = '写真を1枚以上追加してください'
    if (!form.category) errors.category = 'カテゴリを選んでください'
    if (!form.condition) errors.condition = '状態を選んでください'
    if (price === undefined) errors.price = '価格を入力してください'
    if (length(form.description) === 0) errors.description = '説明を入力してください'
    if (!form.campus) errors.campus = '受け渡し希望キャンパスを選んでください'
  }
  return errors
}
