import { formatPrice } from '../../lib/format'
import {
  campusOptions,
  categoryOptions,
  conditionOptions,
  isOptionValue,
  labelOf,
} from '../../lib/itemOptions'
import type { ItemListQuery } from './api'

/** 一覧・検索の条件（BROWSE）。listItems の条件から続きの取得用（cursor・limit）を除いたもの */
export type ItemSearchConditions = Omit<ItemListQuery, 'cursor' | 'limit'>

/** キーワード・授業名の長さの上限（API の maxLength） */
export const SEARCH_TEXT_MAX = 100

/** URL のクエリに並べる順（同じ条件なら同じ文字列になるように固定する） */
const keys = [
  'keyword',
  'category',
  'courseName',
  'priceMin',
  'priceMax',
  'campus',
  'condition',
] as const satisfies readonly (keyof ItemSearchConditions)[]

const text = (value: string | null) => value?.trim().slice(0, SEARCH_TEXT_MAX) || undefined

const price = (value: string | null) =>
  value && /^\d{1,9}$/.test(value) ? Number(value) : undefined

/**
 * URL のクエリを検索条件にする。条件は URL に持たせ、戻る・再読み込み・共有で同じ結果を出す。
 * 知らない値（存在しないカテゴリ、数字でない価格など）は、その条件がないものとして捨てる。
 */
export function parseSearchConditions(params: URLSearchParams): ItemSearchConditions {
  const category = params.get('category') ?? ''
  const campus = params.get('campus') ?? ''
  const condition = params.get('condition') ?? ''
  const conditions: ItemSearchConditions = {
    keyword: text(params.get('keyword')),
    category: isOptionValue(categoryOptions, category) ? category : undefined,
    courseName: text(params.get('courseName')),
    priceMin: price(params.get('priceMin')),
    priceMax: price(params.get('priceMax')),
    campus: isOptionValue(campusOptions, campus) ? campus : undefined,
    condition: isOptionValue(conditionOptions, condition) ? condition : undefined,
  }
  return Object.fromEntries(
    Object.entries(conditions).filter(([, v]) => v !== undefined),
  ) as ItemSearchConditions
}

/** 検索条件を URL のクエリ（API の query にも使う）にする。指定のない条件は省く */
export function toSearchParams(conditions: ItemSearchConditions): URLSearchParams {
  const params = new URLSearchParams()
  for (const key of keys) {
    const value = conditions[key]
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  return params
}

/** 絞り込み条件のチップ。押すと without の条件で検索し直す */
export type FilterChip = { id: string; label: string; without: ItemSearchConditions }

function without(
  conditions: ItemSearchConditions,
  ...removed: (keyof ItemSearchConditions)[]
): ItemSearchConditions {
  const rest = { ...conditions }
  for (const key of removed) delete rest[key]
  return rest
}

/** 価格帯の表示（¥1,000〜¥2,000、¥1,000〜、〜¥2,000） */
export function formatPriceRange(min?: number, max?: number): string {
  return `${min !== undefined ? formatPrice(min) : ''}〜${max !== undefined ? formatPrice(max) : ''}`
}

/**
 * 検索結果の上に出す絞り込み条件のチップ（Search）。キーワードは検索欄に出すのでチップにしない。
 * 価格の下限・上限は1つのチップにまとめ、外すと両方を外す。
 */
export function filterChips(conditions: ItemSearchConditions): FilterChip[] {
  const { category, courseName, priceMin, priceMax, campus, condition } = conditions
  const chips: FilterChip[] = []
  if (category)
    chips.push({
      id: 'category',
      label: labelOf(categoryOptions, category),
      without: without(conditions, 'category'),
    })
  if (courseName)
    chips.push({
      id: 'courseName',
      label: `授業名：${courseName}`,
      without: without(conditions, 'courseName'),
    })
  if (priceMin !== undefined || priceMax !== undefined)
    chips.push({
      id: 'price',
      label: formatPriceRange(priceMin, priceMax),
      without: without(conditions, 'priceMin', 'priceMax'),
    })
  if (campus)
    chips.push({
      id: 'campus',
      label: labelOf(campusOptions, campus),
      without: without(conditions, 'campus'),
    })
  if (condition)
    chips.push({
      id: 'condition',
      label: labelOf(conditionOptions, condition),
      without: without(conditions, 'condition'),
    })
  return chips
}
