package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// transactionRow は transactions の1行（migrations/000013_create_transactions）。
type transactionRow struct {
	ID                   string
	ItemID               string
	BuyerID              string
	SellerID             string
	Price                int64
	PaymentMethod        string
	Status               string
	ApprovalDeadline     string
	FeeRateBp            *int
	Fee                  *int64
	SellerProceeds       *int64
	IdentityDisclosedAt  *string
	SellerAcknowledgedAt *string
	BuyerCancelAgreedAt  *string
	SellerCancelAgreedAt *string
	CompletedAt          *string
	HandoverMethod       *string
	CreatedAt            string
	UpdatedAt            string
}

func (transactionRow) TableName() string { return "transactions" }

func (r transactionRow) toDomain() (*domain.Transaction, error) {
	tr := &domain.Transaction{
		ID:             r.ID,
		ItemID:         r.ItemID,
		BuyerID:        r.BuyerID,
		SellerID:       r.SellerID,
		Price:          r.Price,
		PaymentMethod:  domain.PaymentMethod(r.PaymentMethod),
		Status:         domain.TransactionStatus(r.Status),
		FeeRateBP:      r.FeeRateBp,
		Fee:            r.Fee,
		SellerProceeds: r.SellerProceeds,
		HandoverMethod: (*domain.HandoverMethod)(r.HandoverMethod),
	}
	var err error
	for _, f := range []struct {
		name string
		dst  *time.Time
		src  string
	}{
		{"approval_deadline", &tr.ApprovalDeadline, r.ApprovalDeadline},
		{"created_at", &tr.CreatedAt, r.CreatedAt},
		{"updated_at", &tr.UpdatedAt, r.UpdatedAt},
	} {
		if *f.dst, err = parseTime(f.src); err != nil {
			return nil, fmt.Errorf("repository: transactions.%s: %w", f.name, err)
		}
	}
	for _, f := range []struct {
		name string
		dst  **time.Time
		src  *string
	}{
		{"identity_disclosed_at", &tr.IdentityDisclosedAt, r.IdentityDisclosedAt},
		{"seller_acknowledged_at", &tr.SellerAcknowledgedAt, r.SellerAcknowledgedAt},
		{"buyer_cancel_agreed_at", &tr.BuyerCancelAgreedAt, r.BuyerCancelAgreedAt},
		{"seller_cancel_agreed_at", &tr.SellerCancelAgreedAt, r.SellerCancelAgreedAt},
		{"completed_at", &tr.CompletedAt, r.CompletedAt},
	} {
		if *f.dst, err = parseTimePtr(f.src); err != nil {
			return nil, fmt.Errorf("repository: transactions.%s: %w", f.name, err)
		}
	}
	return tr, nil
}

// TransactionRepository は取引（transactions）を読み書きする（DEAL, DATA）。
// 状態遷移（承認・完了・キャンセルなど）は、取引の各機能で条件付きUPDATE のメソッドとして足す（DB）。
type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create は取引を保存する。ID が空なら採番する。
// 同じ購入者の同じ商品への購入希望がすでにある、または商品の取引がすでに成立しているときは ErrDuplicate。
func (r *TransactionRepository) Create(ctx context.Context, tr *domain.Transaction) error {
	if tr.ID == "" {
		tr.ID = newID()
	}
	err := conn(ctx, r.db).Create(&transactionRow{
		ID:                   tr.ID,
		ItemID:               tr.ItemID,
		BuyerID:              tr.BuyerID,
		SellerID:             tr.SellerID,
		Price:                tr.Price,
		PaymentMethod:        string(tr.PaymentMethod),
		Status:               string(tr.Status),
		ApprovalDeadline:     formatTime(tr.ApprovalDeadline),
		FeeRateBp:            tr.FeeRateBP,
		Fee:                  tr.Fee,
		SellerProceeds:       tr.SellerProceeds,
		IdentityDisclosedAt:  formatTimePtr(tr.IdentityDisclosedAt),
		SellerAcknowledgedAt: formatTimePtr(tr.SellerAcknowledgedAt),
		BuyerCancelAgreedAt:  formatTimePtr(tr.BuyerCancelAgreedAt),
		SellerCancelAgreedAt: formatTimePtr(tr.SellerCancelAgreedAt),
		CompletedAt:          formatTimePtr(tr.CompletedAt),
		HandoverMethod:       enumPtr(tr.HandoverMethod),
		CreatedAt:            formatTime(tr.CreatedAt),
		UpdatedAt:            formatTime(tr.UpdatedAt),
	}).Error
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

// FindByID は ID で取引を探す。なければ ErrNotFound。
func (r *TransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	var rows []transactionRow
	if err := conn(ctx, r.db).Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ListRequestedByItem は商品の承認待ちの購入希望（requested）を、希望の古い順に返す（DEAL-1）。
func (r *TransactionRepository) ListRequestedByItem(ctx context.Context, itemID string) ([]*domain.Transaction, error) {
	var rows []transactionRow
	err := conn(ctx, r.db).Where("item_id = ? AND status = ?", itemID, string(domain.TransactionStatusRequested)).
		Order("created_at ASC, id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return toTransactions(rows)
}

// ListByParty は role として当事者 userID の取引のうち、ステータスが statuses のものを、
// 最後に動いた順（updated_at の新しい順、同じなら ID の大きい順）で返す（取引一覧。ADR 0028）。
// after を渡すとその位置より後ろを返す。limit が 0 なら全件。
func (r *TransactionRepository) ListByParty(ctx context.Context, role domain.TransactionRole, userID string, statuses []domain.TransactionStatus, after *domain.TransactionCursor, limit int) ([]*domain.Transaction, error) {
	q := conn(ctx, r.db).Where(partyColumn(role)+" = ? AND status IN ?", userID, enumStrings(statuses))
	if after != nil {
		at := formatTime(after.UpdatedAt)
		q = q.Where("(updated_at < ? OR (updated_at = ? AND id < ?))", at, at, after.ID)
	}
	q = q.Order("updated_at DESC, id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []transactionRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toTransactions(rows)
}

// ListRequestGroupsBySeller は出品者 sellerID の商品ごとに、承認待ちの購入希望（requested）の件数と最新の購入希望の日時を、
// 最新の購入希望の新しい順で返す（取引一覧の出品側。ADR 0028）。
func (r *TransactionRepository) ListRequestGroupsBySeller(ctx context.Context, sellerID string) ([]domain.TransactionRequestGroup, error) {
	var rows []struct {
		ItemID string
		N      int
		Latest string
	}
	err := conn(ctx, r.db).Model(&transactionRow{}).Select("item_id, COUNT(*) AS n, MAX(created_at) AS latest").
		Where("seller_id = ? AND status = ?", sellerID, string(domain.TransactionStatusRequested)).
		Group("item_id").Order("latest DESC, item_id DESC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.TransactionRequestGroup, 0, len(rows))
	for _, row := range rows {
		at, err := parseTime(row.Latest)
		if err != nil {
			return nil, fmt.Errorf("repository: transactions.created_at: %w", err)
		}
		out = append(out, domain.TransactionRequestGroup{ItemID: row.ItemID, Count: row.N, LatestRequestedAt: at})
	}
	return out, nil
}

// toTransactions は行の並びを取引の並びにする。
func toTransactions(rows []transactionRow) ([]*domain.Transaction, error) {
	out := make([]*domain.Transaction, 0, len(rows))
	for _, row := range rows {
		tr, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, nil
}

// ListExpiredRequests は承認期限 now を過ぎた承認待ちの購入希望（requested）を、期限の古い順に返す（DEAL-1, JOBS-1）。
func (r *TransactionRepository) ListExpiredRequests(ctx context.Context, now time.Time) ([]*domain.Transaction, error) {
	var rows []transactionRow
	err := conn(ctx, r.db).Where("status = ? AND approval_deadline <= ?", string(domain.TransactionStatusRequested), formatTime(now)).
		Order("approval_deadline ASC, id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return toTransactions(rows)
}

// FindRequestedByItemAndBuyer は、購入者が商品に出している承認待ちの購入希望（requested）を返す（DEAL-1）。
// 一意インデックスにより高々1件。なければ ErrNotFound。
func (r *TransactionRepository) FindRequestedByItemAndBuyer(ctx context.Context, itemID, buyerID string) (*domain.Transaction, error) {
	var rows []transactionRow
	err := conn(ctx, r.db).Where("item_id = ? AND buyer_id = ? AND status = ?", itemID, buyerID, string(domain.TransactionStatusRequested)).
		Limit(1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0].toDomain()
}

// ListByPartyAndStatus は購入者または出品者として userID が当事者の、ステータスが status の取引を、古い順に返す（アカウントの失効。ADR 0033）。
func (r *TransactionRepository) ListByPartyAndStatus(ctx context.Context, userID string, status domain.TransactionStatus) ([]*domain.Transaction, error) {
	var rows []transactionRow
	err := conn(ctx, r.db).Where("(buyer_id = ? OR seller_id = ?) AND status = ?", userID, userID, string(status)).
		Order("created_at ASC, id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return toTransactions(rows)
}

// CountRequestedByItem は商品の承認待ちの購入希望の件数を返す。
func (r *TransactionRepository) CountRequestedByItem(ctx context.Context, itemID string) (int, error) {
	var n int64
	err := conn(ctx, r.db).Model(&transactionRow{}).
		Where("item_id = ? AND status = ?", itemID, string(domain.TransactionStatusRequested)).Count(&n).Error
	return int(n), err
}

// CountCompletedByUsers はユーザーごとに、購入者・出品者として完了した取引の件数を返す（DEAL-1 の購入希望一覧）。
// 完了した取引がないユーザーは結果に含めない。
func (r *TransactionRepository) CountCompletedByUsers(ctx context.Context, userIDs []string) (map[string]int, error) {
	out := make(map[string]int, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	for _, col := range []string{"buyer_id", "seller_id"} {
		var rows []struct {
			UserID string
			N      int
		}
		err := conn(ctx, r.db).Model(&transactionRow{}).Select(col+" AS user_id, COUNT(*) AS n").
			Where(col+" IN ? AND status = ?", userIDs, string(domain.TransactionStatusCompleted)).
			Group(col).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			out[row.UserID] += row.N
		}
	}
	return out, nil
}

// Approve は出品者 sellerID の承認待ちの購入希望を、承認期限 now より前のときだけ取引中にする（DEAL-1, DB）。
// 適用した手数料率・手数料・出品者の売上と、学籍番号を開示した日時を同時に記録する（PAY-2, ADR 0003）。
// 状態が違う・期限切れなら ErrConflict、同じ商品の取引がすでに成立していれば ErrDuplicate。
func (r *TransactionRepository) Approve(ctx context.Context, id, sellerID string, feeRateBP int, fee, proceeds int64, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND seller_id = ? AND status = ? AND approval_deadline > ?", id, sellerID, string(domain.TransactionStatusRequested), ts).
		Updates(map[string]any{
			"status":                string(domain.TransactionStatusInDeal),
			"fee_rate_bp":           feeRateBP,
			"fee":                   fee,
			"seller_proceeds":       proceeds,
			"identity_disclosed_at": ts,
			"updated_at":            ts,
		})
	if isUniqueViolation(res.Error) {
		return ErrDuplicate
	}
	return rowsAffected(res)
}

// UpdateStatus は取引のステータスが from のときだけ to にする（却下・期限切れなど。DB）。違えば ErrConflict。
func (r *TransactionRepository) UpdateStatus(ctx context.Context, id string, from, to domain.TransactionStatus, now time.Time) error {
	res := conn(ctx, r.db).Model(&transactionRow{}).Where("id = ? AND status = ?", id, string(from)).
		Updates(map[string]any{"status": string(to), "updated_at": formatTime(now)})
	return rowsAffected(res)
}

// Acknowledge は出品者 sellerID の取引中の取引に、確認モーダルに同意した日時を記録する（DEAL-2）。
// 取引中でない・記録済みなら ErrConflict。
func (r *TransactionRepository) Acknowledge(ctx context.Context, id, sellerID string, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND seller_id = ? AND status = ? AND seller_acknowledged_at IS NULL", id, sellerID, string(domain.TransactionStatusInDeal)).
		Updates(map[string]any{"seller_acknowledged_at": ts, "updated_at": ts})
	return rowsAffected(res)
}

// Complete は購入者 buyerID の取引中の取引を受け渡し完了にし、完了した日時と完了方法を記録する（DEAL-3, DB）。
// 取引中でない（すでに完了・キャンセル済みを含む）なら ErrConflict。
func (r *TransactionRepository) Complete(ctx context.Context, id, buyerID string, method domain.HandoverMethod, now time.Time) error {
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND buyer_id = ? AND status = ?", id, buyerID, string(domain.TransactionStatusInDeal)).
		Updates(map[string]any{
			"status":          string(domain.TransactionStatusCompleted),
			"completed_at":    ts,
			"handover_method": string(method),
			"updated_at":      ts,
		})
	return rowsAffected(res)
}

// AgreeCancel は role として当事者 userID の取引中の取引に、キャンセルへの同意（申請）日時を記録する（DEAL-4, DB）。
// 取引中でない・当事者でない・同意済みなら ErrConflict。
func (r *TransactionRepository) AgreeCancel(ctx context.Context, id, userID string, role domain.TransactionRole, now time.Time) error {
	party, own, _ := cancelColumns(role)
	ts := formatTime(now)
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND "+party+" = ? AND status = ? AND "+own+" IS NULL", id, userID, string(domain.TransactionStatusInDeal)).
		Updates(map[string]any{own: ts, "updated_at": ts})
	return rowsAffected(res)
}

// Cancel は取引中の取引を、双方がキャンセルに同意しているときだけキャンセルにする（DEAL-4, DB）。
// 取引中でない・どちらかが同意していないなら ErrConflict。
func (r *TransactionRepository) Cancel(ctx context.Context, id string, now time.Time) error {
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND status = ? AND buyer_cancel_agreed_at IS NOT NULL AND seller_cancel_agreed_at IS NOT NULL", id, string(domain.TransactionStatusInDeal)).
		Updates(map[string]any{"status": string(domain.TransactionStatusCancelled), "updated_at": formatTime(now)})
	return rowsAffected(res)
}

// CancelBySystem は取引中の取引を、当事者の同意なしにキャンセルにする（当事者のアカウントの失効。DEAL-4 の例外, ADR 0033）。
// 同意日時は入れないので、利用者のキャンセル回数には数えない。取引中でなければ ErrConflict。
func (r *TransactionRepository) CancelBySystem(ctx context.Context, id string, now time.Time) error {
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND status = ?", id, string(domain.TransactionStatusInDeal)).
		Updates(map[string]any{"status": string(domain.TransactionStatusCancelled), "updated_at": formatTime(now)})
	return rowsAffected(res)
}

// WithdrawCancel は role として当事者 userID が出したキャンセルの申請を取り下げる（同意日時を消す。DEAL-4, DB）。
// 取引中でない・当事者でない・申請していない・相手が同意済みなら ErrConflict。
func (r *TransactionRepository) WithdrawCancel(ctx context.Context, id, userID string, role domain.TransactionRole, now time.Time) error {
	party, own, other := cancelColumns(role)
	res := conn(ctx, r.db).Model(&transactionRow{}).
		Where("id = ? AND "+party+" = ? AND status = ? AND "+own+" IS NOT NULL AND "+other+" IS NULL", id, userID, string(domain.TransactionStatusInDeal)).
		Updates(map[string]any{own: nil, "updated_at": formatTime(now)})
	return rowsAffected(res)
}

// cancelColumns は role から見た当事者の列、自分と相手のキャンセルへの同意日時の列を返す（DEAL-4）。
func cancelColumns(role domain.TransactionRole) (party, own, other string) {
	if role == domain.TransactionRoleSeller {
		return partyColumn(role), "seller_cancel_agreed_at", "buyer_cancel_agreed_at"
	}
	return partyColumn(role), "buyer_cancel_agreed_at", "seller_cancel_agreed_at"
}

// partyColumn は role の当事者のユーザーIDの列を返す。
func partyColumn(role domain.TransactionRole) string {
	if role == domain.TransactionRoleSeller {
		return "seller_id"
	}
	return "buyer_id"
}

// rowsAffected は条件付きUPDATE の結果を返す。更新件数が0なら ErrConflict。
func rowsAffected(res *gorm.DB) error {
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}
