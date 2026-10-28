package service

import (
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// 管理画面（ADMIN）の操作に共通する処理。

var errAdminListQuery = apperr.BadRequest("一覧の条件を確認してください")

// requireAdmin は管理者でなければ、存在しないパスと同じ 404 を返す（ミドルウェアと二重のチェック。ADR 0015）。
func requireAdmin(admins domain.AdminList, u *domain.User) error {
	if !admins.Contains(u) {
		return apperr.NotFound("")
	}
	return nil
}

// AdminPageQuery は管理画面の一覧の続きの位置と件数。
type AdminPageQuery struct {
	// Cursor は前のページの NextCursor
	Cursor *string
	// Limit は1回に返す件数。nil なら domain.AdminListDefaultLimit
	Limit *int
}

// parse は続きの位置と、1件多く読む件数（続きがあるかを確かめるため）を返す。不正なら 400 bad_request。
func (q AdminPageQuery) parse() (*domain.AdminCursor, int, error) {
	limit := domain.AdminListDefaultLimit
	if q.Limit != nil {
		limit = *q.Limit
	}
	if limit < 1 || limit > domain.AdminListMaxLimit {
		return nil, 0, errAdminListQuery
	}
	if q.Cursor == nil {
		return nil, limit, nil
	}
	at, id, err := decodeCursor(*q.Cursor)
	if err != nil {
		return nil, 0, errItemListCursor
	}
	return &domain.AdminCursor{At: at, ID: id}, limit, nil
}

// adminPage は limit + 1 件読んだ rows を limit 件に切り、続きがあれば次の位置を返す。key は並びの日時と ID。
func adminPage[T any](rows []T, limit int, key func(T) (time.Time, string)) ([]T, *string) {
	if len(rows) <= limit {
		return rows, nil
	}
	rows = rows[:limit]
	next := encodeCursor(key(rows[limit-1]))
	return rows, &next
}
