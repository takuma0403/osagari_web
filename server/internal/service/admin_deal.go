package service

import (
	"context"
	"errors"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// AdminDealDeps は AdminDealService の依存。
type AdminDealDeps struct {
	Tx           Transactor
	Users        UserStore
	Items        ItemStore
	Transactions TransactionStore
	Payments     payment.Gateway
	// Storage・Bucket は出品の一覧のサムネイルの閲覧用の署名付きURLに使う
	Storage storage.Storage
	// Bucket は商品写真のバケットの名前
	Bucket string
	Admins domain.AdminList
	Clock  clock.Clock
}

// AdminDealService は管理者による出品・取引の確認、出品の停止、取引のキャンセルと返金（モック）を行う（ADMIN, DEAL-4, ADR 0034）。
// どの操作も管理者でなければ 404 にする（ADR 0015）。
type AdminDealService struct {
	d AdminDealDeps
}

func NewAdminDealService(d AdminDealDeps) *AdminDealService {
	return &AdminDealService{d: d}
}

// AdminItemView は管理者に見せる出品。
type AdminItemView struct {
	// Item は商品。写真は1枚目だけ（閲覧用の署名付きURLつき）
	Item                *ItemView
	SellerStudentNumber string
}

// AdminItemPage は管理画面の出品の一覧の1ページ。
type AdminItemPage struct {
	Entries    []*AdminItemView
	NextCursor *string
}

// AdminTransactionView は管理者に見せる取引。管理者には承認前でも当事者の学籍番号を見せる。
type AdminTransactionView struct {
	*domain.Transaction
	Item                *domain.Item
	SellerStudentNumber string
	BuyerStudentNumber  string
}

// AdminTransactionPage は管理画面の取引の一覧の1ページ。
type AdminTransactionPage struct {
	Entries    []*AdminTransactionView
	NextCursor *string
}

// ListItems はすべての出品（下書き・取り下げ・停止を含む）を作った日時の新しい順に返す。
func (s *AdminDealService) ListItems(ctx context.Context, admin *domain.User, q AdminPageQuery) (*AdminItemPage, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	after, limit, err := q.parse()
	if err != nil {
		return nil, err
	}
	items, err := s.d.Items.ListForAdmin(ctx, after, limit+1)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	items, next := adminPage(items, limit, func(it *domain.Item) (time.Time, string) { return it.CreatedAt, it.ID })
	sellerIDs := make([]string, len(items))
	for i, it := range items {
		sellerIDs[i] = it.SellerID
	}
	users, err := s.d.Users.FindByIDs(ctx, sellerIDs)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	page := &AdminItemPage{Entries: make([]*AdminItemView, 0, len(items)), NextCursor: next}
	for _, it := range items {
		v, err := s.itemView(ctx, it, users)
		if err != nil {
			return nil, err
		}
		page.Entries = append(page.Entries, v)
	}
	return page, nil
}

// SuspendItem は下書き・公開中・購入希望あり・取引中の商品を停止にする（ITEM-1, ADR 0034）。
// 購入希望ありなら承認待ちの購入希望をすべて却下して枠を解放する。取引中なら取引は続ける。商品がなければ 404、ほかのステータスなら 409。
func (s *AdminDealService) SuspendItem(ctx context.Context, admin *domain.User, id string) (*AdminItemView, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	it, err := s.d.Items.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errItemNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if !it.CanSuspend() {
		return nil, errItemConflict
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.d.Items.UpdateDealStatus(ctx, it.ID, it.SellerID, domain.ItemSuspendableStatuses, domain.ItemStatusSuspended, now); err != nil {
			return err
		}
		return rejectRequests(ctx, s.d.Transactions, s.d.Payments, it.ID, now)
	})
	if errors.Is(err, repository.ErrConflict) {
		return nil, errItemConflict
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if it, err = s.d.Items.FindByID(ctx, id); err != nil {
		return nil, apperr.Internal(err)
	}
	users, err := s.d.Users.FindByIDs(ctx, []string{it.SellerID})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.itemView(ctx, it, users)
}

// ListTransactions はすべての取引（購入希望を含む）を最後に動いた日時の新しい順に返す。
func (s *AdminDealService) ListTransactions(ctx context.Context, admin *domain.User, q AdminPageQuery) (*AdminTransactionPage, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	after, limit, err := q.parse()
	if err != nil {
		return nil, err
	}
	trs, err := s.d.Transactions.ListForAdmin(ctx, after, limit+1)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	trs, next := adminPage(trs, limit, func(tr *domain.Transaction) (time.Time, string) { return tr.UpdatedAt, tr.ID })
	views, err := s.transactionViews(ctx, trs)
	if err != nil {
		return nil, err
	}
	return &AdminTransactionPage{Entries: views, NextCursor: next}, nil
}

// CancelTransaction はトラブル対応として取引中の取引をキャンセルし、代金を購入者に返金する（モック。DEAL-4 の例外、ADR 0034）。
// 同意日時は入れず、キャンセル回数に数えない。商品は取引中なら公開中に戻す。取引がなければ 404、取引中でなければ 409。
func (s *AdminDealService) CancelTransaction(ctx context.Context, admin *domain.User, id string) (*AdminTransactionView, error) {
	if err := requireAdmin(s.d.Admins, admin); err != nil {
		return nil, err
	}
	tr, err := s.findTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.d.Transactions.CancelBySystem(ctx, tr.ID, now); err != nil {
			return err
		}
		return refundCancelled(ctx, s.d.Items, s.d.Payments, tr, domain.ItemStatusPublished, now)
	})
	if err := dealError(err); err != nil {
		return nil, err
	}
	if tr, err = s.findTransaction(ctx, id); err != nil {
		return nil, err
	}
	views, err := s.transactionViews(ctx, []*domain.Transaction{tr})
	if err != nil {
		return nil, err
	}
	return views[0], nil
}

func (s *AdminDealService) findTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	tr, err := s.d.Transactions.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errTransactionNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return tr, nil
}

func (s *AdminDealService) itemView(ctx context.Context, it *domain.Item, users map[string]*domain.User) (*AdminItemView, error) {
	seller, ok := users[it.SellerID]
	if !ok {
		return nil, apperr.Internal(errors.New("service: 商品の出品者が見つかりません"))
	}
	v, err := newItemView(ctx, s.d.Storage, s.d.Bucket, it, s.d.Clock.Now())
	if err != nil {
		return nil, err
	}
	return &AdminItemView{Item: v, SellerStudentNumber: seller.StudentNumber}, nil
}

// transactionViews は取引に商品と当事者の学籍番号を付ける。
func (s *AdminDealService) transactionViews(ctx context.Context, trs []*domain.Transaction) ([]*AdminTransactionView, error) {
	itemIDs := make([]string, 0, len(trs))
	userIDs := make([]string, 0, 2*len(trs))
	for _, tr := range trs {
		itemIDs = append(itemIDs, tr.ItemID)
		userIDs = append(userIDs, tr.SellerID, tr.BuyerID)
	}
	items, err := s.d.Items.FindByIDs(ctx, itemIDs)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	users, err := s.d.Users.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]*AdminTransactionView, len(trs))
	for i, tr := range trs {
		it, okItem := items[tr.ItemID]
		seller, okSeller := users[tr.SellerID]
		buyer, okBuyer := users[tr.BuyerID]
		if !okItem || !okSeller || !okBuyer {
			return nil, apperr.Internal(errors.New("service: 取引の商品か当事者が見つかりません"))
		}
		out[i] = &AdminTransactionView{Transaction: tr, Item: it, SellerStudentNumber: seller.StudentNumber, BuyerStudentNumber: buyer.StudentNumber}
	}
	return out, nil
}
