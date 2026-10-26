package service_test

import (
	"context"
	"slices"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

// listFixture は取引一覧の各区分に出る取引をそろえる（時計を1分ずつ進めながら、この順に作る）。
//   - r1, r2：商品1への購入者・購入者2の購入希望（承認待ち）
//   - d2：商品2の購入者の取引（取引中）
//   - x3：商品3の購入者の購入希望（却下）
//   - c4：商品4の購入者の取引（完了）
//   - k5：商品5の購入者2の取引（キャンセル）
type listFixture struct {
	*dealFixture
	r1, r2, d2, x3, c4, k5 *service.TransactionView
}

func newListFixture(t *testing.T) *listFixture {
	t.Helper()
	ctx := context.Background()
	f := &listFixture{dealFixture: newDealFixture(t)}
	requestOn := func(u *domain.User, it *service.ItemView) *service.TransactionView {
		t.Helper()
		v, err := f.deals.CreateRequest(ctx, u, it.ShareID, domain.PaymentMethodCard)
		if err != nil {
			t.Fatal(err)
		}
		f.tick()
		return v
	}
	approveOn := func(u *domain.User) *service.TransactionView {
		t.Helper()
		v := requestOn(u, f.publish(t))
		if _, err := f.deals.Approve(ctx, f.seller, v.ID); err != nil {
			t.Fatal(err)
		}
		f.tick()
		return v
	}

	f.r1 = requestOn(f.buyer, f.item)
	f.r2 = requestOn(f.buyer2, f.item)
	f.d2 = approveOn(f.buyer)
	f.x3 = requestOn(f.buyer, f.publish(t))
	if _, err := f.deals.Reject(ctx, f.seller, f.x3.ID); err != nil {
		t.Fatal(err)
	}
	f.tick()
	f.c4 = approveOn(f.buyer)
	if err := f.txs.Complete(ctx, f.c4.ID, f.buyer.ID, domain.HandoverMethodQR, f.clock.Now()); err != nil {
		t.Fatal(err)
	}
	f.tick()
	f.k5 = approveOn(f.buyer2)
	for _, u := range []*domain.User{f.buyer2, f.seller} {
		if _, err := f.deals.Cancel(ctx, u, f.k5.ID); err != nil {
			t.Fatal(err)
		}
		f.tick()
	}
	return f
}

func (f *listFixture) list(t *testing.T, u *domain.User, q service.TransactionListQuery) *service.TransactionListPage {
	t.Helper()
	page, err := f.deals.List(context.Background(), u, q)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

// entryIDs は行を取引IDの並びにする。商品ごとの承認待ちの行は "item:<商品ID>" にする。
func entryIDs(page *service.TransactionListPage) []string {
	out := make([]string, len(page.Entries))
	for i, e := range page.Entries {
		if e.TransactionID == nil {
			out[i] = "item:" + *e.ItemID
			continue
		}
		out[i] = *e.TransactionID
	}
	return out
}

// 当事者の取引だけを、role・phase の区分ごとに最後に動いた順で返す（DEAL-1, ADR 0028）。
func TestDealList振り分け(t *testing.T) {
	f := newListFixture(t)
	q := func(role domain.TransactionRole, phase domain.TransactionPhase) service.TransactionListQuery {
		return service.TransactionListQuery{Role: role, Phase: phase}
	}
	buyer, seller := domain.TransactionRoleBuyer, domain.TransactionRoleSeller
	active, ended := domain.TransactionPhaseActive, domain.TransactionPhaseEnded

	tests := []struct {
		name string
		user *domain.User
		q    service.TransactionListQuery
		want []string
	}{
		{"購入者の進行中", f.buyer, q(buyer, active), []string{f.d2.ID, f.r1.ID}},
		{"購入者の終了（却下を含む）", f.buyer, q(buyer, ended), []string{f.c4.ID, f.x3.ID}},
		{"購入者2の進行中", f.buyer2, q(buyer, active), []string{f.r2.ID}},
		{"購入者2の終了", f.buyer2, q(buyer, ended), []string{f.k5.ID}},
		{"出品者の進行中（購入希望は商品ごと）", f.seller, q(seller, active), []string{f.d2.ID, "item:" + f.item.ID}},
		{"出品者の終了（却下は出さない）", f.seller, q(seller, ended), []string{f.k5.ID, f.c4.ID}},
		{"購入者として出品の取引を見ない", f.seller, q(buyer, active), []string{}},
		{"出品者として購入の取引を見ない", f.buyer, q(seller, ended), []string{}},
		{"当事者でない人の進行中", f.stranger, q(buyer, active), []string{}},
		{"当事者でない人の出品", f.stranger, q(seller, active), []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := f.list(t, tt.user, tt.q)
			if got := entryIDs(page); !slices.Equal(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
			if page.NextCursor != nil {
				t.Errorf("続きがないのに NextCursor = %v", *page.NextCursor)
			}
		})
	}
}

// 行の項目：承認期限は購入側の購入希望だけ、商品IDは出品側だけ、商品ごとの行は件数と今の価格（DEAL-1, ADR 0028）。
func TestDealList項目(t *testing.T) {
	f := newListFixture(t)

	bought := f.list(t, f.buyer, service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: domain.TransactionPhaseActive}).Entries
	d2, r1 := bought[0], bought[1]
	if r1.Status != domain.TransactionStatusRequested || r1.ApprovalDeadline == nil || !r1.ApprovalDeadline.Equal(f.r1.ApprovalDeadline) ||
		r1.Price != f.r1.Price || r1.ItemID != nil || r1.RequestCount != nil || !r1.UpdatedAt.Equal(f.r1.CreatedAt) {
		t.Errorf("購入側の購入希望: got = %+v", r1)
	}
	if d2.Status != domain.TransactionStatusInDeal || d2.ApprovalDeadline != nil || d2.ItemID != nil {
		t.Errorf("購入側の取引中: got = %+v", d2)
	}
	for _, e := range bought {
		if e.Item.Title == "" || len(e.Item.Images) != 1 || len(e.Item.ImageURLs) != 1 || e.Item.ImageURLs[0].URL == "" {
			t.Errorf("サムネイル: got = %+v", e.Item)
		}
	}

	sold := f.list(t, f.seller, service.TransactionListQuery{Role: domain.TransactionRoleSeller, Phase: domain.TransactionPhaseActive}).Entries
	d2, group := sold[0], sold[1]
	if d2.TransactionID == nil || *d2.TransactionID != f.d2.ID || d2.ItemID == nil || *d2.ItemID != f.d2.ItemID || d2.ApprovalDeadline != nil {
		t.Errorf("出品側の取引中: got = %+v", d2)
	}
	if group.Status != domain.TransactionStatusRequested || group.ItemID == nil || *group.ItemID != f.item.ID ||
		group.RequestCount == nil || *group.RequestCount != 2 || group.Price != *f.item.Price ||
		group.ApprovalDeadline != nil || !group.UpdatedAt.Equal(f.r2.CreatedAt) || group.Item.ShareID != f.item.ShareID {
		t.Errorf("出品側の商品ごとの購入希望: got = %+v", group)
	}

	ended := f.list(t, f.seller, service.TransactionListQuery{Role: domain.TransactionRoleSeller, Phase: domain.TransactionPhaseEnded}).Entries
	if c4 := ended[1]; c4.Status != domain.TransactionStatusCompleted || c4.CompletedAt == nil {
		t.Errorf("完了: got = %+v", c4)
	}
	if k5 := ended[0]; k5.Status != domain.TransactionStatusCancelled || k5.CompletedAt != nil {
		t.Errorf("キャンセル: got = %+v", k5)
	}
}

// 終了した取引は cursor で続きを取る。条件が不正なら 400（ADR 0028）。
func TestDealListページ分け(t *testing.T) {
	ctx := context.Background()
	f := newListFixture(t)
	one := 1
	ended := service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: domain.TransactionPhaseEnded, Limit: &one}

	first := f.list(t, f.buyer, ended)
	if got := entryIDs(first); !slices.Equal(got, []string{f.c4.ID}) || first.NextCursor == nil {
		t.Fatalf("1ページ目: got = %v, next = %v", got, first.NextCursor)
	}
	ended.Cursor = first.NextCursor
	second := f.list(t, f.buyer, ended)
	if got := entryIDs(second); !slices.Equal(got, []string{f.x3.ID}) || second.NextCursor != nil {
		t.Errorf("2ページ目: got = %v, next = %v", got, second.NextCursor)
	}

	zero, over, broken := 0, domain.TransactionListMaxLimit+1, "!!"
	tests := []struct {
		name string
		q    service.TransactionListQuery
	}{
		{"立場が不正", service.TransactionListQuery{Role: "admin", Phase: domain.TransactionPhaseActive}},
		{"区分が不正", service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: "all"}},
		{"進行中に cursor", service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: domain.TransactionPhaseActive, Cursor: first.NextCursor}},
		{"件数が0", service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: domain.TransactionPhaseEnded, Limit: &zero}},
		{"件数が上限を超える", service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: domain.TransactionPhaseEnded, Limit: &over}},
		{"壊れた cursor", service.TransactionListQuery{Role: domain.TransactionRoleBuyer, Phase: domain.TransactionPhaseEnded, Cursor: &broken}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.deals.List(ctx, f.buyer, tt.q)
			wantCode(t, err, apperr.CodeBadRequest)
		})
	}
}
