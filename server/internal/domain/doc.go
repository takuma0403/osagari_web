// Package domain はエンティティと列挙値を置く。名前は docs/glossary.md に揃える。
//
// エンティティの構造体（User・Item・Transaction など）は、そのテーブルを使う機能の issue で追加する（認証まわりは user.go）。
package domain

import "slices"

// enumValid は列挙値 v が values に含まれるかを返す。各列挙型の Valid から使う。
func enumValid[T ~string](v T, values []T) bool {
	return slices.Contains(values, v)
}
