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

var (
	errTransactionNotFound = apperr.NotFound("取引が見つかりません")
	errTransactionConflict = apperr.InvalidStatusTransition("")
	errOwnItemRequest      = apperr.Forbidden("自分の商品には購入希望を出せません")
	errPaymentMethod       = apperr.BadRequest("支払い方法を選んでください")
)

// DealDeps は DealService の依存。
type DealDeps struct {
	Tx           Transactor
	Items        ItemStore
	Users        UserStore
	Transactions TransactionStore
	// HandoverTokens は受け渡しトークン（DEAL-3。handover.go）
	HandoverTokens HandoverTokenStore
	Settings       SettingStore
	Payments       payment.Gateway
	// Storage・Bucket は取引一覧のサムネイルの閲覧用の署名付きURLに使う（ADR 0028）
	Storage storage.Storage
	// Bucket は商品写真のバケットの名前
	Bucket string
	Clock  clock.Clock
}

// DealService は購入希望・承認・却下・受け渡し完了・キャンセルと取引の詳細を扱う（DEAL-1, DEAL-2, DEAL-3, DEAL-4）。
// 状態遷移は条件付きUPDATE で行い、決済は payment.Gateway に同じトランザクションで記録する（DB, JOBS-2）。
type DealService struct {
	d DealDeps
}

func NewDealService(d DealDeps) *DealService {
	return &DealService{d: d}
}

// TransactionView は当事者に返す取引。
type TransactionView struct {
	*domain.Transaction
	// Role は見ている本人が購入者か出品者か
	Role domain.TransactionRole
	Item *domain.Item
	// CounterpartStudentNumber は相手の学籍番号。承認（学籍番号の開示）の後だけ入る（DEAL-1）
	CounterpartStudentNumber *string
}

// PurchaseRequestView は出品者に見せる承認待ちの購入希望。購入者の学籍番号は含めない（DEAL-1）。
type PurchaseRequestView struct {
	*domain.Transaction
	Buyer BuyerProfile
}

// BuyerProfile は購入希望の一覧に出す購入者の情報。
type BuyerProfile struct {
	// DepartmentName は学部・学科の名前。不明なら nil
	DepartmentName *string
	AdmissionYear  int
	// CompletedDealCount は購入者・出品者として取引を完了した回数
	CompletedDealCount int
}

// CreateRequest は商品ページで見られる商品に購入希望を出し、枠を確保する（DEAL-1, JOBS-2）。
// 自分の商品は 403、取引中・取引完了の商品と、同じ商品に承認待ちの購入希望を出し済みなら 409。
func (s *DealService) CreateRequest(ctx context.Context, u *domain.User, shareID string, method domain.PaymentMethod) (*TransactionView, error) {
	if !method.Valid() {
		return nil, errPaymentMethod
	}
	now := s.d.Clock.Now()
	it, _, err := findPublicItem(ctx, s.d.Items, s.d.Users, shareID, now)
	if err != nil {
		return nil, err
	}
	if it.SellerID == u.ID {
		return nil, errOwnItemRequest
	}
	if !it.CanRequest() {
		return nil, errTransactionConflict
	}
	tr := &domain.Transaction{
		ItemID:           it.ID,
		BuyerID:          u.ID,
		SellerID:         it.SellerID,
		PaymentMethod:    method,
		Status:           domain.TransactionStatusRequested,
		ApprovalDeadline: now.Add(domain.ApprovalTTL),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		// 購入希望ありの間もほかの人は購入希望を出せる（DEAL-1）
		from := []domain.ItemStatus{domain.ItemStatusPublished, domain.ItemStatusRequested}
		if err := s.d.Items.UpdateDealStatus(ctx, it.ID, it.SellerID, from, domain.ItemStatusRequested, now); err != nil {
			return err
		}
		// 価格はトランザクションの中で読み直す。読んだ後に公開中のまま編集されても、記録する金額と食い違わせない
		cur, err := s.d.Items.FindByID(ctx, it.ID)
		if err != nil {
			return err
		}
		if cur.Price == nil {
			return repository.ErrConflict
		}
		tr.Price = *cur.Price
		if err := s.d.Transactions.Create(ctx, tr); err != nil {
			return err
		}
		return s.d.Payments.Authorize(ctx, tr)
	})
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.view(ctx, u, tr.ID)
}

// ListRequests は出品者本人の商品に届いている承認待ちの購入希望を、希望の古い順に返す（DEAL-1）。ほかの人の商品は 404。
func (s *DealService) ListRequests(ctx context.Context, u *domain.User, itemID string) ([]*PurchaseRequestView, error) {
	it, err := s.d.Items.FindByID(ctx, itemID)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && it.SellerID != u.ID) {
		return nil, errItemNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	reqs, err := s.d.Transactions.ListRequestedByItem(ctx, it.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	ids := make([]string, len(reqs))
	for i, tr := range reqs {
		ids[i] = tr.BuyerID
	}
	buyers, err := s.d.Users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	counts, err := s.d.Transactions.CountCompletedByUsers(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]*PurchaseRequestView, 0, len(reqs))
	for _, tr := range reqs {
		b, ok := buyers[tr.BuyerID]
		if !ok {
			return nil, apperr.Internal(errors.New("service: 購入希望の購入者が見つかりません"))
		}
		out = append(out, &PurchaseRequestView{Transaction: tr, Buyer: BuyerProfile{
			DepartmentName: b.DepartmentName(), AdmissionYear: b.AdmissionYear, CompletedDealCount: counts[b.ID],
		}})
	}
	return out, nil
}

// Get は本人が当事者の取引を返す。当事者でなければ 404（DEAL-1）。
func (s *DealService) Get(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	return s.view(ctx, u, id)
}

// Approve は出品者が購入希望を承認して取引を成立させる（DEAL-1, PAY-2）。
// 代金を預かり、その時点の手数料率で手数料を記録し（ADR 0003）、学籍番号を開示する。同じ商品のほかの購入希望は却下して枠を解放する。
// 承認待ちでない・承認期限を過ぎた・同じ商品の取引がすでに成立しているときは 409。
func (s *DealService) Approve(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	tr, err := s.findAsSeller(ctx, u, id)
	if err != nil {
		return nil, err
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		rate, err := s.d.Settings.FeeRate(ctx)
		if err != nil {
			return err
		}
		fee, proceeds := domain.CalcFee(tr.Price, rate)
		if err := s.d.Transactions.Approve(ctx, tr.ID, u.ID, rate, fee, proceeds, now); err != nil {
			return err
		}
		if err := s.d.Items.UpdateDealStatus(ctx, tr.ItemID, u.ID, []domain.ItemStatus{domain.ItemStatusRequested}, domain.ItemStatusInDeal, now); err != nil {
			return err
		}
		if err := s.d.Payments.Capture(ctx, tr); err != nil {
			return err
		}
		return rejectRequests(ctx, s.d.Transactions, s.d.Payments, tr.ItemID, now)
	})
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.view(ctx, u, tr.ID)
}

// Reject は出品者が購入希望を却下して枠を解放する（DEAL-1, JOBS-2）。
// 承認待ちの購入希望がなくなったら商品を公開中に戻す（ADR 0022）。承認待ちでなければ 409。
func (s *DealService) Reject(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	tr, err := s.findAsSeller(ctx, u, id)
	if err != nil {
		return nil, err
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		return endRequest(ctx, s.d.Transactions, s.d.Items, s.d.Payments, tr, domain.TransactionStatusRejected, now)
	})
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.view(ctx, u, tr.ID)
}

// ExpireRequests は承認期限を過ぎた購入希望を期限切れにして枠を解放し、期限切れにした件数を返す（DEAL-1, JOBS-1）。
// 短周期ジョブから呼ぶ。1件ずつトランザクションで行い、その間に承認・却下された購入希望は飛ばす（何度呼んでも結果は同じ）。
// 1件が失敗しても残りは続け、失敗をまとめて返す。
func (s *DealService) ExpireRequests(ctx context.Context) (int, error) {
	now := s.d.Clock.Now()
	reqs, err := s.d.Transactions.ListExpiredRequests(ctx, now)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	n := 0
	var errs []error
	for _, tr := range reqs {
		err := s.d.Tx.Do(ctx, func(ctx context.Context) error {
			return endRequest(ctx, s.d.Transactions, s.d.Items, s.d.Payments, tr, domain.TransactionStatusExpired, now)
		})
		switch {
		case errors.Is(err, repository.ErrConflict):
		case err != nil:
			errs = append(errs, err)
		default:
			n++
		}
	}
	if err := errors.Join(errs...); err != nil {
		return n, apperr.Internal(err)
	}
	return n, nil
}

// DeleteExpiredHandoverTokens は期限が切れた受け渡しトークンを消し、消した件数を返す（DEAL-3, JOBS-1）。
func (s *DealService) DeleteExpiredHandoverTokens(ctx context.Context) (int, error) {
	n, err := s.d.HandoverTokens.DeleteExpired(ctx, s.d.Clock.Now())
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

// endRequest は承認待ちの購入希望を to（却下・期限切れ）にして枠を解放し、
// 承認待ちの購入希望がなくなったら商品を公開中に戻す（DEAL-1, ADR 0022）。トランザクションの中で呼ぶ。承認待ちでなければ repository.ErrConflict。
// アカウントの失効（ADR 0033）からも呼ぶ。
func endRequest(ctx context.Context, txs TransactionStore, items ItemStore, gw payment.Gateway, tr *domain.Transaction, to domain.TransactionStatus, now time.Time) error {
	if err := closeRequest(ctx, txs, gw, tr, to, now); err != nil {
		return err
	}
	n, err := txs.CountRequestedByItem(ctx, tr.ItemID)
	if err != nil || n > 0 {
		return err
	}
	err = items.UpdateDealStatus(ctx, tr.ItemID, tr.SellerID, []domain.ItemStatus{domain.ItemStatusRequested}, domain.ItemStatusPublished, now)
	// 商品が購入希望ありでなければ（停止など）そのままにする
	if errors.Is(err, repository.ErrConflict) {
		return nil
	}
	return err
}

// Acknowledge は出品者が取引成立の確認モーダルに同意した日時を記録する（DEAL-2）。
// 記録済みなら最初の日時のまま返す（同時に押された場合も含む）。取引中でなければ 409。
func (s *DealService) Acknowledge(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	tr, err := s.findAsSeller(ctx, u, id)
	if err != nil {
		return nil, err
	}
	err = s.d.Transactions.Acknowledge(ctx, tr.ID, u.ID, s.d.Clock.Now())
	if errors.Is(err, repository.ErrConflict) {
		v, err := s.view(ctx, u, tr.ID)
		if err != nil {
			return nil, err
		}
		if v.Status != domain.TransactionStatusInDeal || v.SellerAcknowledgedAt == nil {
			return nil, errTransactionConflict
		}
		return v, nil
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.view(ctx, u, tr.ID)
}

// findAsSeller は本人が出品者の取引を返す。ないか、出品者でなければ 404（購入者にも存在を区別させない）。
func (s *DealService) findAsSeller(ctx context.Context, u *domain.User, id string) (*domain.Transaction, error) {
	return s.findAs(ctx, u, id, domain.TransactionRoleSeller)
}

// findAs は本人が role として当事者の取引を返す。ないか、role でなければ 404（相手にも存在を区別させない）。
func (s *DealService) findAs(ctx context.Context, u *domain.User, id string, role domain.TransactionRole) (*domain.Transaction, error) {
	tr, r, err := s.findAsParty(ctx, u, id)
	if err != nil {
		return nil, err
	}
	if r != role {
		return nil, errTransactionNotFound
	}
	return tr, nil
}

// findAsParty は本人が当事者の取引と、本人が購入者か出品者かを返す。ないか、当事者でなければ 404。
func (s *DealService) findAsParty(ctx context.Context, u *domain.User, id string) (*domain.Transaction, domain.TransactionRole, error) {
	tr, err := s.d.Transactions.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, "", errTransactionNotFound
	}
	if err != nil {
		return nil, "", apperr.Internal(err)
	}
	r, ok := tr.RoleOf(u.ID)
	if !ok {
		return nil, "", errTransactionNotFound
	}
	return tr, r, nil
}

// view は取引を読み直し、本人から見た形にする。当事者でなければ 404。
// 相手の学籍番号は開示の後だけ、手数料は出品者にだけ入れる（DEAL-1, PAY-2）。
func (s *DealService) view(ctx context.Context, u *domain.User, id string) (*TransactionView, error) {
	tr, err := s.d.Transactions.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errTransactionNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	role, ok := tr.RoleOf(u.ID)
	if !ok {
		return nil, errTransactionNotFound
	}
	it, err := s.d.Items.FindByID(ctx, tr.ItemID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	v := &TransactionView{Transaction: tr, Role: role, Item: it}
	if role == domain.TransactionRoleBuyer {
		tr.FeeRateBP, tr.Fee, tr.SellerProceeds = nil, nil, nil
	}
	if tr.IdentityDisclosedAt != nil {
		other, err := s.d.Users.FindByID(ctx, tr.CounterpartID(u.ID))
		if err != nil {
			return nil, apperr.Internal(err)
		}
		v.CounterpartStudentNumber = &other.StudentNumber
	}
	return v, nil
}

// closeRequest は承認待ちの購入希望を to（却下・期限切れ）にして枠を解放する（DEAL-1, JOBS-2）。承認待ちでなければ repository.ErrConflict。
func closeRequest(ctx context.Context, txs TransactionStore, gw payment.Gateway, tr *domain.Transaction, to domain.TransactionStatus, now time.Time) error {
	if err := txs.UpdateStatus(ctx, tr.ID, domain.TransactionStatusRequested, to, now); err != nil {
		return err
	}
	return gw.Release(ctx, tr)
}

// rejectRequests は商品の承認待ちの購入希望をすべて却下して枠を解放する（承認・商品の取り下げ。DEAL-1, ADR 0022）。
// 1件ごとに解放を記録するため、まとめて UPDATE せずに1件ずつ行う。トランザクションの中で呼ぶ。
func rejectRequests(ctx context.Context, txs TransactionStore, gw payment.Gateway, itemID string, now time.Time) error {
	reqs, err := txs.ListRequestedByItem(ctx, itemID)
	if err != nil {
		return err
	}
	for _, tr := range reqs {
		if err := closeRequest(ctx, txs, gw, tr, domain.TransactionStatusRejected, now); err != nil {
			return err
		}
	}
	return nil
}

// dealError は取引の状態遷移のエラーを応答のエラーにする。競合（条件付きUPDATE の0件・一意制約）は 409。
func dealError(err error) error {
	var ae *apperr.Error
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrConflict), errors.Is(err, repository.ErrDuplicate):
		return errTransactionConflict
	case errors.As(err, &ae):
		return err
	default:
		return apperr.Internal(err)
	}
}
