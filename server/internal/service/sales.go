package service

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

var errNoPayableSales = apperr.InvalidStatusTransition("出金できる売上がありません")

// SalesDeps は SalesService の依存。
type SalesDeps struct {
	Tx           Transactor
	Items        ItemStore
	Transactions TransactionStore
	// Payouts は出金の記録の読み出し。記録は Payments（payment.Gateway）で行う
	Payouts  PaymentStore
	Payments payment.Gateway
	// Storage・Bucket は明細のサムネイルの閲覧用の署名付きURLに使う
	Storage storage.Storage
	// Bucket は商品写真のバケットの名前
	Bucket string
	Clock  clock.Clock
}

// SalesService は売上管理（Sales）の明細・集計と出金申請を扱う（PAY, PAY-2, ADR 0029）。
// 手数料は取引成立時に記録した料率・金額をそのまま使い、計算し直さない（ADR 0003）。
type SalesService struct {
	d SalesDeps
}

func NewSalesService(d SalesDeps) *SalesService {
	return &SalesService{d: d}
}

// SalesEntryView は売上の明細の1行。
type SalesEntryView struct {
	domain.SalesEntry
	// Item は商品。写真は1枚目だけ（閲覧用の署名付きURLつき）
	Item *ItemView
}

// SalesView は売上の集計と、最後に動いた順の明細。
type SalesView struct {
	domain.SalesSummary
	Entries []*SalesEntryView
}

// Get は本人が出品者として成立した取引（取引中・完了）の売上を返す。学生証の承認がなくても、取引がなければ空を返す。
func (s *SalesService) Get(ctx context.Context, u *domain.User) (*SalesView, error) {
	entries, err := s.entries(ctx, u)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.view(ctx, entries)
}

// RequestPayout は出金できる売上（settled）をすべて出金し、取引ごとに出金を記録する（モック。PAY, JOBS-2, ADR 0029）。
// 出金できる売上がないか、同時に申請されて先に出金されていたら 409 invalid_status_transition。
func (s *SalesService) RequestPayout(ctx context.Context, u *domain.User) (*SalesView, error) {
	err := s.d.Tx.Do(ctx, func(ctx context.Context) error {
		// トランザクションの中で読み直し、出金済みの取引を出金し直さない
		entries, err := s.entries(ctx, u)
		if err != nil {
			return err
		}
		n := 0
		for _, e := range entries {
			if e.State() != domain.SalesEntryStateSettled {
				continue
			}
			// 取引ごとに1行だけ。同時の申請は一意制約で ErrDuplicate になる
			if err := s.d.Payments.Payout(ctx, e.Transaction); err != nil {
				return err
			}
			n++
		}
		if n == 0 {
			return errNoPayableSales
		}
		return nil
	})
	if err := dealError(err); err != nil {
		return nil, err
	}
	return s.Get(ctx, u)
}

// entries は本人の売上の明細を、出金の有無つきで返す（並びは問わない）。
func (s *SalesService) entries(ctx context.Context, u *domain.User) ([]domain.SalesEntry, error) {
	trs, err := s.d.Transactions.ListByParty(ctx, domain.TransactionRoleSeller, u.ID, domain.SalesTransactionStatuses, nil, 0)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(trs))
	for i, tr := range trs {
		ids[i] = tr.ID
	}
	payouts, err := s.d.Payouts.ListPayoutsByTransactions(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.SalesEntry, len(trs))
	for i, tr := range trs {
		// 取引中・完了の取引には成立時に手数料が記録されている（PAY-2）
		if tr.SellerProceeds == nil || tr.Fee == nil || tr.FeeRateBP == nil || tr.IdentityDisclosedAt == nil {
			return nil, errors.New("service: 成立した取引に手数料が記録されていません")
		}
		out[i] = domain.SalesEntry{Transaction: tr}
		if p, ok := payouts[tr.ID]; ok {
			out[i].PaidOutAt = &p.CreatedAt
		}
	}
	return out, nil
}

// view は明細に商品を付け、集計して、最後に動いた順に並べる。
func (s *SalesService) view(ctx context.Context, entries []domain.SalesEntry) (*SalesView, error) {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.Transaction.ItemID
	}
	items, err := s.d.Items.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	now := s.d.Clock.Now()
	out := &SalesView{SalesSummary: domain.SummarizeSales(entries), Entries: make([]*SalesEntryView, 0, len(entries))}
	for _, e := range entries {
		it, ok := items[e.Transaction.ItemID]
		if !ok {
			return nil, apperr.Internal(errors.New("service: 売上の取引の商品が見つかりません"))
		}
		v, err := newItemView(ctx, s.d.Storage, s.d.Bucket, it, now)
		if err != nil {
			return nil, err
		}
		out.Entries = append(out.Entries, &SalesEntryView{SalesEntry: e, Item: v})
	}
	slices.SortStableFunc(out.Entries, func(a, b *SalesEntryView) int {
		if c := b.LastActivityAt().Compare(a.LastActivityAt()); c != 0 {
			return c
		}
		return strings.Compare(b.Transaction.ID, a.Transaction.ID)
	})
	return out, nil
}
