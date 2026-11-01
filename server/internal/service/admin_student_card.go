package service

import (
	"context"
	"errors"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// studentCardViewTTL は学生証画像の閲覧用の署名付きURLの有効期限（STORAGE, ADR 0015）。
const studentCardViewTTL = 15 * time.Minute

var (
	errStudentCardNotFound     = apperr.NotFound("学生証が見つかりません")
	errStudentCardImageDeleted = apperr.NotFound("学生証の画像はすでに削除されています")
	errStudentCardNotPending   = apperr.InvalidStatusTransition("この学生証は審査待ちではありません")
)

// AdminStudentCardDeps は AdminStudentCardService の依存。
type AdminStudentCardDeps struct {
	Tx      Transactor
	Users   UserStore
	Cards   StudentCardStore
	Views   StudentCardViewStore
	Storage storage.Storage
	// Bucket は学生証の非公開バケットの名前
	Bucket string
	Admins domain.AdminList
	Clock  clock.Clock
}

// AdminStudentCardService は管理者による学生証の審査（一覧・画像の閲覧・承認・差し戻し）を行う（ADMIN, CARD, STORAGE）。
// どの操作も管理者でなければ 404 にする（管理APIの存在を隠す。ミドルウェアと二重のチェック。ADR 0015）。
type AdminStudentCardService struct {
	d AdminStudentCardDeps
}

func NewAdminStudentCardService(d AdminStudentCardDeps) *AdminStudentCardService {
	return &AdminStudentCardService{d: d}
}

// AdminStudentCard は管理者に見せる学生証と本人。
type AdminStudentCard struct {
	Card *domain.StudentCard
	User *domain.User
}

// StudentCardImage は発行した閲覧URLと、記録した閲覧の日時。
type StudentCardImage struct {
	storage.SignedURL
	ViewedAt time.Time
}

// StudentCardViewEntry は閲覧記録の1件。管理者は学籍番号で示す（AdminCard の「管理者A」の部分）。
type StudentCardViewEntry struct {
	ViewedAt           time.Time
	AdminStudentNumber string
}

// RequireAdmin は管理者でなければ、存在しないパスと同じ 404 を返す。
func (s *AdminStudentCardService) RequireAdmin(u *domain.User) error {
	return requireAdmin(s.d.Admins, u)
}

// ListPending は審査待ちの学生証を提出の古い順に返す。
func (s *AdminStudentCardService) ListPending(ctx context.Context, admin *domain.User) ([]AdminStudentCard, error) {
	if err := s.RequireAdmin(admin); err != nil {
		return nil, err
	}
	cards, err := s.d.Cards.ListPending(ctx)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	ids := make([]string, len(cards))
	for i, c := range cards {
		ids[i] = c.UserID
	}
	users, err := s.d.Users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	list := make([]AdminStudentCard, 0, len(cards))
	for _, c := range cards {
		u, ok := users[c.UserID]
		if !ok {
			return nil, apperr.Internal(errors.New("service: 学生証の本人が見つかりません"))
		}
		list = append(list, AdminStudentCard{Card: c, User: u})
	}
	return list, nil
}

// CreateView は学生証画像の閲覧URL（15分有効）を発行し、閲覧した管理者と日時を記録する。
// 記録を先に残し、記録のない閲覧URLを作らない。
func (s *AdminStudentCardService) CreateView(ctx context.Context, admin *domain.User, cardID string) (StudentCardImage, error) {
	if err := s.RequireAdmin(admin); err != nil {
		return StudentCardImage{}, err
	}
	card, err := s.findCard(ctx, cardID)
	if err != nil {
		return StudentCardImage{}, err
	}
	if card.ImageDeletedAt != nil {
		return StudentCardImage{}, errStudentCardImageDeleted
	}
	now := s.d.Clock.Now()
	view := &domain.StudentCardView{StudentCardID: card.ID, AdminUserID: admin.ID, ViewedAt: now, CreatedAt: now, UpdatedAt: now}
	if err := s.d.Views.Create(ctx, view); err != nil {
		return StudentCardImage{}, apperr.Internal(err)
	}
	signed, err := s.d.Storage.SignedGetURL(ctx, s.d.Bucket, card.ObjectKey, now.Add(studentCardViewTTL))
	if err != nil {
		return StudentCardImage{}, apperr.Internal(err)
	}
	return StudentCardImage{SignedURL: signed, ViewedAt: now}, nil
}

// ListViews は学生証画像の閲覧記録を新しい順に返す。
func (s *AdminStudentCardService) ListViews(ctx context.Context, admin *domain.User, cardID string) ([]StudentCardViewEntry, error) {
	if err := s.RequireAdmin(admin); err != nil {
		return nil, err
	}
	if _, err := s.findCard(ctx, cardID); err != nil {
		return nil, err
	}
	views, err := s.d.Views.ListByCard(ctx, cardID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	ids := make([]string, len(views))
	for i, v := range views {
		ids[i] = v.AdminUserID
	}
	admins, err := s.d.Users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	entries := make([]StudentCardViewEntry, len(views))
	for i, v := range views {
		entries[i] = StudentCardViewEntry{ViewedAt: v.ViewedAt}
		if a, ok := admins[v.AdminUserID]; ok {
			entries[i].AdminStudentNumber = a.StudentNumber
		}
	}
	return entries, nil
}

// Approve は学生証を承認し、本人を出品できるようにする（CARD）。
func (s *AdminStudentCardService) Approve(ctx context.Context, admin *domain.User, cardID string) (AdminStudentCard, error) {
	return s.review(ctx, admin, cardID, domain.StudentCardStatusApproved, domain.SellerStatusApproved)
}

// Reject は学生証を差し戻す。理由は持たない（ADR 0015）。本人は再提出できる。
func (s *AdminStudentCardService) Reject(ctx context.Context, admin *domain.User, cardID string) (AdminStudentCard, error) {
	return s.review(ctx, admin, cardID, domain.StudentCardStatusRejected, domain.SellerStatusRejected)
}

// review は審査待ちで置き換えられていない学生証と、審査中の本人の承認状態を、1つのトランザクションで条件付きに更新する（DB）。
// 同時に審査した・その間に再提出されたなどで条件に合わなければ 409。
func (s *AdminStudentCardService) review(ctx context.Context, admin *domain.User, cardID string, to domain.StudentCardStatus, seller domain.SellerStatus) (AdminStudentCard, error) {
	if err := s.RequireAdmin(admin); err != nil {
		return AdminStudentCard{}, err
	}
	card, err := s.findCard(ctx, cardID)
	if err != nil {
		return AdminStudentCard{}, err
	}
	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if err := s.d.Cards.Review(ctx, card.ID, to, admin.ID, now); err != nil {
			return err
		}
		return s.d.Users.UpdateSellerStatus(ctx, card.UserID, []domain.SellerStatus{domain.SellerStatusPending}, seller, now)
	})
	if errors.Is(err, repository.ErrConflict) {
		return AdminStudentCard{}, errStudentCardNotPending
	}
	if err != nil {
		return AdminStudentCard{}, apperr.Internal(err)
	}
	if card, err = s.d.Cards.FindByID(ctx, card.ID); err != nil {
		return AdminStudentCard{}, apperr.Internal(err)
	}
	u, err := s.d.Users.FindByID(ctx, card.UserID)
	if err != nil {
		return AdminStudentCard{}, apperr.Internal(err)
	}
	return AdminStudentCard{Card: card, User: u}, nil
}

func (s *AdminStudentCardService) findCard(ctx context.Context, id string) (*domain.StudentCard, error) {
	card, err := s.d.Cards.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errStudentCardNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return card, nil
}
