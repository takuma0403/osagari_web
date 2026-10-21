import type { components } from '../api/schema.gen'
import type { PhotoTint } from '../components/ItemPhoto/ItemPhoto'

type ItemCategory = components['schemas']['ItemCategory']
type ItemCondition = components['schemas']['ItemCondition']
type Campus = components['schemas']['Campus']

/** 選択肢の値と表示名 */
export type Option<T extends string> = { value: T; label: string }

/** カテゴリ（ITEM, glossary の ItemCategory）。出品の選択・一覧のチップ・カードの表示で使う */
export const categoryOptions: Option<ItemCategory>[] = [
  { value: 'textbook', label: '教科書・参考書' },
  { value: 'furniture_appliance', label: '家具・家電' },
  { value: 'game_other', label: 'ゲーム・その他' },
]

/** 商品の状態（ITEM） */
export const conditionOptions: Option<ItemCondition>[] = [
  { value: 'new_like', label: '新品同様' },
  { value: 'good', label: '目立った傷なし' },
  { value: 'fair', label: 'やや傷あり' },
]

/** 受け渡し希望キャンパス（ITEM） */
export const campusOptions: Option<Campus>[] = [
  { value: 'honjo', label: '本庄' },
  { value: 'nabeshima', label: '鍋島' },
]

/** 値の表示名。options にない値はそのまま返す */
export function labelOf<T extends string>(options: Option<T>[], value: T): string {
  return options.find((o) => o.value === value)?.label ?? value
}

/** 値が options のどれかか（URL のクエリなど、外から来た文字列の確認に使う） */
export function isOptionValue<T extends string>(options: Option<T>[], value: string): value is T {
  return options.some((o) => o.value === value)
}

/**
 * 写真がないときの地の色（DS-2 の --color-cat-*）。家具・家電は家具の色を使い、家電の色は使わない
 * （教科書の色と見分けやすくするため。glossary の注記）
 */
export const categoryTint: Record<ItemCategory, PhotoTint> = {
  textbook: 'textbook',
  furniture_appliance: 'furniture',
  game_other: 'game',
}
