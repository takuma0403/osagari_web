package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
)

var errTransactionListQuery = apperr.BadRequest("取引一覧の条件を確認してください")

// TransactionListQuery は取引一覧の入力（ADR 0028）。
type TransactionListQuery struct {
	Role  domain.TransactionRole
	Phase domain.TransactionPhase
	// Cursor は前のページの NextCursor（終了した取引だけ）
	Cursor *string
	// Limit は1回に返す件数（終了した取引だけ）。nil なら domain.TransactionListDefaultLimit
	Limit *int
}

// TransactionListPage は取引一覧の1ページ。
type TransactionListPage struct {
	Entries []*TransactionListEntry
	// NextCursor は続きを取るときに渡す値。続きがなければ nil（進行中は常に nil）
	NextCursor *string
}

// TransactionListEntry は取引一覧の1行。ふつうは取引1件、出品側の承認待ちの購入希望は商品ごとに1行（ADR 0028）。
// 学籍番号・手数料は持たない（DEAL-1）。
type TransactionListEntry struct {
	// TransactionID は取引のID。出品側の商品ごとの承認待ちの行は nil
	TransactionID *string
	// ItemID は商品のID。出品側だけ入る（購入希望の一覧に使う）
	ItemID *string
	Status domain.TransactionStatus
	// Item は商品。写真は1枚目だけ（閲覧用の署名付きURLつき）
	Item *ItemView
	// Price は取引の金額（円）。出品側の商品ごとの行は今の商品価格
	Price int64
	// RequestCount は承認待ちの購入希望の件数。出品側の商品ごとの行だけ入る
	RequestCount *int
	// ApprovalDeadline は承認期限。購入側の購入希望だけ入る
	ApprovalDeadline *time.Time
	CompletedAt      *time.Time
	// UpdatedAt は最後に動いた日時。出品側の商品ごとの行は最新の購入希望を出した日時
	UpdatedAt time.Time
}

// List は本人が role として当事者の取引を、phase の区分で最後に動いた順に返す（取引一覧。DEAL-1, DEAL-3, DEAL-4, ADR 0028）。
// 進行中は全件、終了は cursor で続きを取る。条件が不正なら 400 bad_request。
func (s *DealService) List(ctx context.Context, u *domain.User, q TransactionListQuery) (*TransactionListPage, error) {
	if !q.Role.Valid() || !q.Phase.Valid() {
		return nil, errTransactionListQuery
	}
	var after *domain.TransactionCursor
	limit := 0 // 進行中は全件（購入希望は48時間で期限切れになり、取引中は商品ごとに1件なので件数が限られる）
	if q.Phase == domain.TransactionPhaseActive {
		if q.Cursor != nil {
			return nil, errTransactionListQuery
		}
	} else {
		limit = domain.TransactionListDefaultLimit
		if q.Limit != nil {
			limit = *q.Limit
		}
		if limit < 1 || limit > domain.TransactionListMaxLimit {
			return nil, errTransactionListQuery
		}
		if q.Cursor != nil {
			at, id, err := decodeCursor(*q.Cursor)
			if err != nil {
				return nil, errItemListCursor
			}
			after = &domain.TransactionCursor{UpdatedAt: at, ID: id}
		}
	}

	// 終了は1件多く読み、続きがあるかを確かめる
	read := limit
	if read > 0 {
		read++
	}
	trs, err := s.d.Transactions.ListByParty(ctx, q.Role, u.ID, domain.TransactionListStatuses(q.Role, q.Phase), after, read)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	page := &TransactionListPage{}
	if limit > 0 && len(trs) > limit {
		trs = trs[:limit]
		last := trs[limit-1]
		next := encodeCursor(last.UpdatedAt, last.ID)
		page.NextCursor = &next
	}
	var groups []domain.TransactionRequestGroup
	if q.Role == domain.TransactionRoleSeller && q.Phase == domain.TransactionPhaseActive {
		if groups, err = s.d.Transactions.ListRequestGroupsBySeller(ctx, u.ID); err != nil {
			return nil, apperr.Internal(err)
		}
	}

	ids := make([]string, 0, len(trs)+len(groups))
	for _, tr := range trs {
		ids = append(ids, tr.ItemID)
	}
	for _, g := range groups {
		ids = append(ids, g.ItemID)
	}
	items, err := s.d.Items.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	now := s.d.Clock.Now()
	item := func(id string) (*ItemView, error) {
		it, ok := items[id]
		if !ok {
			return nil, apperr.Internal(errors.New("service: 取引の商品が見つかりません"))
		}
		return newItemView(ctx, s.d.Storage, s.d.Bucket, it, now)
	}

	page.Entries = make([]*TransactionListEntry, 0, len(trs)+len(groups))
	for _, tr := range trs {
		v, err := item(tr.ItemID)
		if err != nil {
			return nil, err
		}
		e := &TransactionListEntry{
			TransactionID: &tr.ID, Status: tr.Status, Item: v, Price: tr.Price,
			CompletedAt: tr.CompletedAt, UpdatedAt: tr.UpdatedAt,
		}
		if q.Role == domain.TransactionRoleSeller {
			e.ItemID = &tr.ItemID
		} else if tr.Status == domain.TransactionStatusRequested {
			e.ApprovalDeadline = &tr.ApprovalDeadline
		}
		page.Entries = append(page.Entries, e)
	}
	for _, g := range groups {
		v, err := item(g.ItemID)
		if err != nil {
			return nil, err
		}
		// 購入希望を受け付けている商品は公開時に価格がそろっている（ADR 0016）
		if v.Price == nil {
			return nil, apperr.Internal(errors.New("service: 購入希望のある商品に価格がありません"))
		}
		page.Entries = append(page.Entries, &TransactionListEntry{
			ItemID: &g.ItemID, Status: domain.TransactionStatusRequested, Item: v, Price: *v.Price,
			RequestCount: &g.Count, UpdatedAt: g.LatestRequestedAt,
		})
	}
	// 出品側の進行中は、取引中と商品ごとの購入希望を合わせて最後に動いた順にする
	slices.SortStableFunc(page.Entries, func(a, b *TransactionListEntry) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	return page, nil
}
