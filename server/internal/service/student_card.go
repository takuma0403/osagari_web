package service

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// 学生証の機能固有のエラーコード（docs/tech/api.md の API-1）。
const CodeInvalidStudentCardUpload apperr.Code = "invalid_student_card_upload"

const (
	// studentCardUploadTTL はアップロード用の署名付きURLの有効期限（STORAGE）。
	studentCardUploadTTL = 15 * time.Minute
	// studentCardMaxBytes は学生証画像の上限（STORAGE）。ブラウザで長辺1600pxに縮小した画像なら十分に収まる。
	studentCardMaxBytes = 5 << 20
)

// studentCardContentTypes は受け付ける画像の形式（STORAGE）。
var studentCardContentTypes = []string{"image/jpeg", "image/webp"}

var (
	errInvalidStudentCardUpload = apperr.New(http.StatusBadRequest, CodeInvalidStudentCardUpload,
		"学生証の画像を確認できませんでした。もう一度アップロードしてください")
	errStudentCardTooLarge = apperr.New(http.StatusBadRequest, CodeInvalidStudentCardUpload,
		"学生証の画像が大きすぎます。5MB以下にしてもう一度アップロードしてください")
	errStudentCardApproved = apperr.InvalidStatusTransition("学生証はすでに承認されています")
	errStudentCardConflict = apperr.InvalidStatusTransition("")
)

// StudentCardDeps は StudentCardService の依存。
type StudentCardDeps struct {
	Tx      Transactor
	Users   UserStore
	Cards   StudentCardStore
	Storage storage.Storage
	// Bucket は学生証の非公開バケットの名前
	Bucket string
	Clock  clock.Clock
}

// StudentCardService は学生証画像のアップロードURLの発行・登録・審査状況の取得を行う（CARD, STORAGE）。
// 画像の閲覧URLはここでは発行しない（管理者APIだけが発行する）。
type StudentCardService struct {
	d StudentCardDeps
}

func NewStudentCardService(d StudentCardDeps) *StudentCardService {
	return &StudentCardService{d: d}
}

// StudentCardUpload は発行したアップロード先。ObjectKey を Submit に送る。
type StudentCardUpload struct {
	ObjectKey string
	storage.SignedURL
}

// StudentCardState は本人の学生証の審査状況。
type StudentCardState struct {
	Status      domain.SellerStatus
	SubmittedAt *time.Time
	ReviewedAt  *time.Time
}

// CreateUpload は学生証画像を直接 PUT するための署名付きURLを発行する。オブジェクト名は `{ユーザーID}/{UUID}`。
func (s *StudentCardService) CreateUpload(ctx context.Context, u *domain.User, contentType string) (StudentCardUpload, error) {
	if !slices.Contains(studentCardContentTypes, contentType) {
		return StudentCardUpload{}, apperr.BadRequest("JPEG か WebP の画像を選んでください")
	}
	if u.SellerStatus == domain.SellerStatusApproved {
		return StudentCardUpload{}, errStudentCardApproved
	}
	objectKey := u.ID + "/" + uuid.NewString()
	signed, err := s.d.Storage.SignedPutURL(ctx, s.d.Bucket, objectKey, contentType, studentCardMaxBytes, s.d.Clock.Now().Add(studentCardUploadTTL))
	if err != nil {
		return StudentCardUpload{}, apperr.Internal(err)
	}
	return StudentCardUpload{ObjectKey: objectKey, SignedURL: signed}, nil
}

// Submit はアップロードした画像を学生証として登録し、出品者の承認状態を審査中にする。
// 審査中・差し戻しのときは前の学生証を置き換える（前の行と画像は残す。ADR 0014）。
func (s *StudentCardService) Submit(ctx context.Context, u *domain.User, objectKey string) (StudentCardState, error) {
	if u.SellerStatus == domain.SellerStatusApproved {
		return StudentCardState{}, errStudentCardApproved
	}
	// ほかの人のオブジェクトや、発行していない形の名前は受け付けない
	id, ok := strings.CutPrefix(objectKey, u.ID+"/")
	if parsed, err := uuid.Parse(id); !ok || err != nil || parsed.String() != id {
		return StudentCardState{}, errInvalidStudentCardUpload
	}
	exists, err := s.d.Cards.ExistsByObjectKey(ctx, objectKey)
	if err != nil {
		return StudentCardState{}, apperr.Internal(err)
	}
	if exists {
		return StudentCardState{}, errInvalidStudentCardUpload
	}
	info, err := s.d.Storage.Stat(ctx, s.d.Bucket, objectKey)
	if errors.Is(err, storage.ErrNotFound) {
		return StudentCardState{}, errInvalidStudentCardUpload
	}
	if err != nil {
		return StudentCardState{}, apperr.Internal(err)
	}
	if !slices.Contains(studentCardContentTypes, info.ContentType) || info.Size <= 0 {
		return StudentCardState{}, errInvalidStudentCardUpload
	}
	if info.Size > studentCardMaxBytes {
		return StudentCardState{}, errStudentCardTooLarge
	}

	now := s.d.Clock.Now()
	card := &domain.StudentCard{
		UserID:      u.ID,
		ObjectKey:   objectKey,
		ContentType: info.ContentType,
		SizeBytes:   info.Size,
		Status:      domain.StudentCardStatusPending,
		SubmittedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		if _, err := s.d.Cards.SupersedeCurrent(ctx, u.ID, now); err != nil {
			return err
		}
		if err := s.d.Cards.Create(ctx, card); err != nil {
			return err
		}
		return s.d.Users.UpdateSellerStatus(ctx, u.ID,
			[]domain.SellerStatus{domain.SellerStatusNone, domain.SellerStatusPending, domain.SellerStatusRejected},
			domain.SellerStatusPending, now)
	})
	// 同時に登録した・その間に承認された（現在の学生証の一意制約、承認状態の条件付き更新）
	if errors.Is(err, repository.ErrDuplicate) || errors.Is(err, repository.ErrConflict) {
		return StudentCardState{}, errStudentCardConflict
	}
	if err != nil {
		return StudentCardState{}, apperr.Internal(err)
	}
	return StudentCardState{Status: domain.SellerStatusPending, SubmittedAt: &card.SubmittedAt}, nil
}

// Current は本人の現在の学生証の審査状況を返す。未登録なら状態は none。
func (s *StudentCardService) Current(ctx context.Context, u *domain.User) (StudentCardState, error) {
	card, err := s.d.Cards.FindCurrentByUser(ctx, u.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return StudentCardState{Status: u.SellerStatus}, nil
	}
	if err != nil {
		return StudentCardState{}, apperr.Internal(err)
	}
	return StudentCardState{Status: u.SellerStatus, SubmittedAt: &card.SubmittedAt, ReviewedAt: card.ReviewedAt}, nil
}
