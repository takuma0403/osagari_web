package service

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// 出品の機能固有のエラーコード（docs/tech/api.md の API-1）。
const (
	CodeSellerNotApproved      apperr.Code = "seller_not_approved"
	CodeInvalidItemImageUpload apperr.Code = "invalid_item_image_upload"
)

var (
	errSellerNotApproved = apperr.New(http.StatusForbidden, CodeSellerNotApproved,
		"出品するには学生証の承認が必要です")
	errInvalidItemImage = apperr.New(http.StatusBadRequest, CodeInvalidItemImageUpload,
		"写真を確認できませんでした。もう一度アップロードしてください")
	errItemImageTooLarge = apperr.New(http.StatusBadRequest, CodeInvalidItemImageUpload,
		"写真が大きすぎます。5MB以下にしてもう一度アップロードしてください")
	errItemNotFound = apperr.NotFound("商品が見つかりません")
	errItemConflict = apperr.InvalidStatusTransition("")
)

// shareIDLen は共有用IDの長さ。base62 で12文字（約71ビット）にして推測できないようにする。
const shareIDLen = 12

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ItemDeps は ItemService の依存。
type ItemDeps struct {
	Tx    Transactor
	Items ItemStore
	// Users は商品ページに出品者の情報を出すのに使う（BROWSE）
	Users   UserStore
	Storage storage.Storage
	// Bucket は商品写真のバケットの名前
	Bucket string
	Clock  clock.Clock
}

// ItemService は出品・編集・取り下げ・再出品と、商品写真のアップロードURLの発行（ITEM, STORAGE）、
// ほかの利用者向けの一覧・検索・商品ページ（BROWSE。item_browse.go）を扱う。
type ItemService struct {
	d ItemDeps
}

func NewItemService(d ItemDeps) *ItemService {
	return &ItemService{d: d}
}

// ItemView は出品者本人に返す商品。写真には閲覧用の署名付きURLを付ける。
type ItemView struct {
	*domain.Item
	// ImageURLs は Images と同じ順の閲覧用の署名付きURL
	ImageURLs []storage.SignedURL
}

// CreateImageUpload は商品写真を直接 PUT するための署名付きURLを発行する。オブジェクト名は `{ユーザーID}/{UUID}`。
func (s *ItemService) CreateImageUpload(ctx context.Context, u *domain.User, contentType string) (ImageUpload, error) {
	if !validImageContentType(contentType) {
		return ImageUpload{}, errImageContentType
	}
	if err := requireSeller(u); err != nil {
		return ImageUpload{}, err
	}
	up, err := signImageUpload(ctx, s.d.Storage, s.d.Bucket, u.ID, contentType, s.d.Clock.Now())
	if err != nil {
		return ImageUpload{}, apperr.Internal(err)
	}
	return up, nil
}

// Create は商品を下書き（publish=false）か公開中（publish=true）として作る。下書きはタイトルだけ必須（ADR 0016）。
func (s *ItemService) Create(ctx context.Context, u *domain.User, in domain.ItemContent, imageKeys []string, publish bool) (*ItemView, error) {
	if err := requireSeller(u); err != nil {
		return nil, err
	}
	in = in.Normalize()
	if err := validateItemContent(in, len(imageKeys), publish); err != nil {
		return nil, err
	}
	images, err := s.resolveImages(ctx, u, imageKeys, nil)
	if err != nil {
		return nil, err
	}
	shareID, err := newShareID()
	if err != nil {
		return nil, apperr.Internal(err)
	}

	now := s.d.Clock.Now()
	it := &domain.Item{
		SellerID:    u.ID,
		ShareID:     shareID,
		Status:      domain.ItemStatusDraft,
		ItemContent: in,
		Images:      images,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if publish {
		it.Status = domain.ItemStatusPublished
		it.PublishedAt = &now
	}
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error { return s.d.Items.Create(ctx, it) })
	// 写真がほかの商品で使われている（同時に同じ写真で出品した場合も含む）
	if errors.Is(err, repository.ErrDuplicate) {
		return nil, errInvalidItemImage
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.view(ctx, it)
}

// Get は出品者本人の商品を返す。ほかの人の商品は、あるかどうかを区別せずに 404 にする。
func (s *ItemService) Get(ctx context.Context, u *domain.User, id string) (*ItemView, error) {
	it, err := s.findOwn(ctx, u, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, it)
}

// Update は商品の内容と写真をすべて置き換える。編集できるのは下書き・公開中・取り下げだけ（ADR 0016）。
// 公開中の商品はすべての共通項目と写真1枚以上が必要で、下書き・取り下げはタイトルだけ必須。
func (s *ItemService) Update(ctx context.Context, u *domain.User, id string, in domain.ItemContent, imageKeys []string) (*ItemView, error) {
	if err := requireSeller(u); err != nil {
		return nil, err
	}
	it, err := s.findOwn(ctx, u, id)
	if err != nil {
		return nil, err
	}
	if !it.CanEdit() {
		return nil, errItemConflict
	}
	in = in.Normalize()
	if err := validateItemContent(in, len(imageKeys), it.Status == domain.ItemStatusPublished); err != nil {
		return nil, err
	}
	images, err := s.resolveImages(ctx, u, imageKeys, it.Images)
	if err != nil {
		return nil, err
	}

	now := s.d.Clock.Now()
	err = s.d.Tx.Do(ctx, func(ctx context.Context) error {
		// 読んだときのステータス・内容のままのときだけ書き換える（確かめた入力の決まりと食い違わせない）
		if err := s.d.Items.UpdateContent(ctx, it.ID, u.ID, it.Status, it.UpdatedAt, in, now); err != nil {
			return err
		}
		return s.d.Items.ReplaceImages(ctx, it.ID, images, now)
	})
	switch {
	case errors.Is(err, repository.ErrConflict):
		return nil, errItemConflict
	case errors.Is(err, repository.ErrDuplicate):
		return nil, errInvalidItemImage
	case err != nil:
		return nil, apperr.Internal(err)
	}
	return s.reload(ctx, it.ID)
}

// Publish は下書きか取り下げの商品を公開中にする（再出品を含む）。すべての共通項目と写真1枚以上が必要。
func (s *ItemService) Publish(ctx context.Context, u *domain.User, id string) (*ItemView, error) {
	if err := requireSeller(u); err != nil {
		return nil, err
	}
	it, err := s.findOwn(ctx, u, id)
	if err != nil {
		return nil, err
	}
	if !it.CanPublish() {
		return nil, errItemConflict
	}
	if err := validateItemContent(it.ItemContent, len(it.Images), true); err != nil {
		return nil, err
	}
	// 確かめた後に編集されていたら公開しない（足りない内容のまま公開させない）
	err = s.d.Items.UpdateStatus(ctx, it.ID, u.ID, []domain.ItemStatus{it.Status}, &it.UpdatedAt, domain.ItemStatusPublished, s.d.Clock.Now())
	if errors.Is(err, repository.ErrConflict) {
		return nil, errItemConflict
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.reload(ctx, it.ID)
}

// Withdraw は下書き・公開中・購入希望ありの商品を取り下げる。取引中は DEAL-4 のキャンセルでしか止められない（ADR 0016）。
// 学生証の承認は問わない（出品者が出品をやめる操作なので）。
func (s *ItemService) Withdraw(ctx context.Context, u *domain.User, id string) (*ItemView, error) {
	it, err := s.findOwn(ctx, u, id)
	if err != nil {
		return nil, err
	}
	if !it.CanWithdraw() {
		return nil, errItemConflict
	}
	// TODO(#25): 購入希望ありの商品を取り下げたときは、その購入希望を却下して枠を解放する（同じトランザクションで行う）
	err = s.d.Items.UpdateStatus(ctx, it.ID, u.ID, domain.ItemWithdrawableStatuses, nil, domain.ItemStatusWithdrawn, s.d.Clock.Now())
	if errors.Is(err, repository.ErrConflict) {
		return nil, errItemConflict
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.reload(ctx, it.ID)
}

// requireSeller は学生証が承認された出品者かを確かめる（ITEM, CARD）。
func requireSeller(u *domain.User) error {
	if u.SellerStatus != domain.SellerStatusApproved {
		return errSellerNotApproved
	}
	return nil
}

// validateItemContent は公開するなら公開の決まりで、そうでなければ下書きの決まりで入力を確かめる。
func validateItemContent(c domain.ItemContent, imageCount int, publish bool) error {
	validate := c.ValidateDraft
	if publish {
		validate = c.ValidateForPublish
	}
	var ie *domain.ItemInputError
	if err := validate(imageCount); errors.As(err, &ie) {
		return apperr.BadRequest(ie.Message)
	} else if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// findOwn は出品者本人の商品を返す。ないか、ほかの人の商品なら 404。
func (s *ItemService) findOwn(ctx context.Context, u *domain.User, id string) (*domain.Item, error) {
	it, err := s.d.Items.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errItemNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if it.SellerID != u.ID {
		return nil, errItemNotFound
	}
	return it, nil
}

// resolveImages は写真のオブジェクト名を、並び順どおりの写真にする。current（編集前の写真）にあるものはそのまま使い、
// 新しいものはアップロード済みかを確かめる。同じ写真を2回並べることはできない。
func (s *ItemService) resolveImages(ctx context.Context, u *domain.User, keys []string, current []domain.ItemImage) ([]domain.ItemImage, error) {
	known := make(map[string]domain.ItemImage, len(current))
	for _, img := range current {
		known[img.ObjectKey] = img
	}
	seen := make(map[string]bool, len(keys))
	images := make([]domain.ItemImage, 0, len(keys))
	for _, key := range keys {
		if seen[key] {
			return nil, errInvalidItemImage
		}
		seen[key] = true
		if img, ok := known[key]; ok {
			images = append(images, domain.ItemImage{ObjectKey: key, ContentType: img.ContentType, SizeBytes: img.SizeBytes})
			continue
		}
		info, err := statUploadedImage(ctx, s.d.Storage, s.d.Bucket, u.ID, key)
		switch {
		case errors.Is(err, errUploadedImageInvalid):
			return nil, errInvalidItemImage
		case errors.Is(err, errUploadedImageTooLarge):
			return nil, errItemImageTooLarge
		case err != nil:
			return nil, apperr.Internal(err)
		}
		images = append(images, domain.ItemImage{ObjectKey: key, ContentType: info.ContentType, SizeBytes: info.Size})
	}
	return images, nil
}

func (s *ItemService) reload(ctx context.Context, id string) (*ItemView, error) {
	it, err := s.d.Items.FindByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return s.view(ctx, it)
}

// view は写真に閲覧用の署名付きURL（15分有効。STORAGE）を付ける。
func (s *ItemService) view(ctx context.Context, it *domain.Item) (*ItemView, error) {
	expiresAt := s.d.Clock.Now().Add(imageUploadTTL)
	urls := make([]storage.SignedURL, len(it.Images))
	for i, img := range it.Images {
		signed, err := s.d.Storage.SignedGetURL(ctx, s.d.Bucket, img.ObjectKey, expiresAt)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		urls[i] = signed
	}
	return &ItemView{Item: it, ImageURLs: urls}, nil
}

// newShareID は共有用ID（base62 で shareIDLen 文字）を crypto/rand で作る。
func newShareID() (string, error) {
	b := make([]byte, shareIDLen)
	// 248 未満だけを使い、62 で割った余りが偏らないようにする
	out := make([]byte, 0, shareIDLen)
	for len(out) < shareIDLen {
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		for _, c := range b {
			if c < 248 && len(out) < shareIDLen {
				out = append(out, base62[c%62])
			}
		}
	}
	return string(out), nil
}
