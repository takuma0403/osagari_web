package service

import (
	"context"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// OwnItemSummaryView はマイページの出品の1行（AUTH-3, ADR 0029）。
type OwnItemSummaryView struct {
	// Item は商品。写真は1枚目だけ（閲覧用の署名付きURLつき）
	Item *ItemView
	// RequestCount は承認待ちの購入希望の件数
	RequestCount int
}

// ListOwn は本人のすべての出品（下書き・取り下げ・停止を含む）を、最後に更新した順に返す（マイページ。AUTH-3, ADR 0029）。
// 学生証の承認がなくても、出品がなければ空を返す。
func (s *ItemService) ListOwn(ctx context.Context, u *domain.User) ([]*OwnItemSummaryView, error) {
	items, err := s.d.Items.ListBySeller(ctx, u.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	// 承認待ちの件数は取引一覧と同じ集計を使う（ADR 0028）
	groups, err := s.d.Transactions.ListRequestGroupsBySeller(ctx, u.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	counts := make(map[string]int, len(groups))
	for _, g := range groups {
		counts[g.ItemID] = g.Count
	}
	out := make([]*OwnItemSummaryView, 0, len(items))
	for _, it := range items {
		v, err := s.view(ctx, it)
		if err != nil {
			return nil, err
		}
		out = append(out, &OwnItemSummaryView{Item: v, RequestCount: counts[it.ID]})
	}
	return out, nil
}
