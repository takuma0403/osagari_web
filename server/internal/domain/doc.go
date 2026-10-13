// Package domain はエンティティと列挙値を置く。名前は docs/glossary.md に揃える。
//
// エンティティの構造体（User・Item・Transaction など）は、そのテーブルを作るマイグレーションと同じ issue で追加する。
// 商品のカテゴリ（ItemCategory）と状態（ItemCondition）は値の一覧が未確定なので、出品（ITEM）の実装時に決める。
package domain

import "slices"

// enumValid は列挙値 v が values に含まれるかを返す。各列挙型の Valid から使う。
func enumValid[T ~string](v T, values []T) bool {
	return slices.Contains(values, v)
}
