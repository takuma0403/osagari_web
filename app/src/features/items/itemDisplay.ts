import {
  campusOptions,
  categoryOptions,
  categoryTint,
  conditionOptions,
  labelOf,
} from '../../lib/itemOptions'
import { shareText } from '../../lib/share'
import type { ItemSummary, PublicItem } from './api'

type ItemCategory = ItemSummary['category']
type ItemCondition = ItemSummary['condition']

/** 商品ページの固定URL（BROWSE） */
export const itemPath = (shareId: string) => `/items/${encodeURIComponent(shareId)}`

/**
 * 価格の横に添える状態の注記。教科書で書き込みの有無が分かればそれを、ほかは状態を出す。
 * 検索結果の行と共有文（FE の例「（書き込みなし）」）で使う。
 */
export function conditionNote(
  category: ItemCategory,
  hasWriting: boolean | null | undefined,
  condition: ItemCondition,
): string {
  if (category === 'textbook' && hasWriting != null) {
    return hasWriting ? '書き込みあり' : '書き込みなし'
  }
  return labelOf(conditionOptions, condition)
}

/** 共有リンクに添える文（BROWSE）。origin はアプリの URL の先頭（https://… ） */
export function itemShareText(item: PublicItem, origin: string): string {
  return shareText({
    title: item.title,
    price: item.price,
    note: conditionNote(item.category, item.textbook?.hasWriting, item.condition),
    url: `${origin}${itemPath(item.shareId)}`,
  })
}

/** 一覧のカード（ItemCard）に渡す値 */
export function cardProps(item: ItemSummary) {
  return {
    to: itemPath(item.shareId),
    title: item.title,
    price: item.price,
    categoryLabel: labelOf(categoryOptions, item.category),
    tint: categoryTint[item.category],
    campus: labelOf(campusOptions, item.campus),
    condition: labelOf(conditionOptions, item.condition),
    imageUrl: item.thumbnail?.url,
  }
}

/**
 * 検索結果の行（ItemRow）に渡す値。補足は教科書なら授業名、それ以外はカテゴリ名。
 * 価格の右は、教科書で書き込みの有無が分かればそれを、ほかは状態を出す。
 */
export function rowProps(item: ItemSummary) {
  const isTextbook = item.category === 'textbook'
  return {
    to: itemPath(item.shareId),
    title: item.title,
    subtitle:
      isTextbook && item.courseName ? item.courseName : labelOf(categoryOptions, item.category),
    price: item.price,
    note: conditionNote(item.category, item.hasWriting, item.condition),
    tint: categoryTint[item.category],
    imageUrl: item.thumbnail?.url,
  }
}
