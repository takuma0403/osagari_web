import {
  campusOptions,
  categoryOptions,
  categoryTint,
  conditionOptions,
  labelOf,
} from '../../lib/itemOptions'
import type { ItemSummary } from './api'

/** 商品ページの固定URL（BROWSE） */
export const itemPath = (shareId: string) => `/items/${encodeURIComponent(shareId)}`

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
    note:
      isTextbook && item.hasWriting != null
        ? item.hasWriting
          ? '書き込みあり'
          : '書き込みなし'
        : labelOf(conditionOptions, item.condition),
    tint: categoryTint[item.category],
    imageUrl: item.thumbnail?.url,
  }
}
