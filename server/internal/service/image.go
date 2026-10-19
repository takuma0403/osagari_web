package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// 学生証と商品写真に共通する画像のアップロードの決まり（STORAGE）。
const (
	// imageUploadTTL はアップロード用・閲覧用の署名付きURLの有効期限
	imageUploadTTL = 15 * time.Minute
	// imageMaxBytes は画像1枚の上限。ブラウザで長辺1600pxに縮小した画像なら十分に収まる
	imageMaxBytes = 5 << 20
)

// imageContentTypes は受け付ける画像の形式。
var imageContentTypes = []string{"image/jpeg", "image/webp"}

var (
	// errImageContentType はアップロードしようとした画像の形式が違う。
	errImageContentType = apperr.BadRequest("JPEG か WebP の画像を選んでください")
	// errUploadedImageInvalid はオブジェクトが本人のものでない・アップロードされていない・形式が違う。
	errUploadedImageInvalid = errors.New("service: アップロードした画像を確認できません")
	// errUploadedImageTooLarge は画像が imageMaxBytes を超えている。
	errUploadedImageTooLarge = errors.New("service: アップロードした画像が大きすぎます")
)

// ImageUpload は発行したアップロード先。ObjectKey を登録の API に送る。
type ImageUpload struct {
	ObjectKey string
	storage.SignedURL
}

// validImageContentType は受け付ける画像の形式かを返す。
func validImageContentType(contentType string) bool {
	return slices.Contains(imageContentTypes, contentType)
}

// signImageUpload は bucket に画像を直接 PUT するための署名付きURLを発行する。オブジェクト名は `{ユーザーID}/{UUID}` にして、
// ほかの人の名前で登録できないようにする。contentType は呼び出し側で validImageContentType を確かめておく。
func signImageUpload(ctx context.Context, st storage.Storage, bucket, userID, contentType string, now time.Time) (ImageUpload, error) {
	objectKey := userID + "/" + uuid.NewString()
	signed, err := st.SignedPutURL(ctx, bucket, objectKey, contentType, imageMaxBytes, now.Add(imageUploadTTL))
	if err != nil {
		return ImageUpload{}, err
	}
	return ImageUpload{ObjectKey: objectKey, SignedURL: signed}, nil
}

// ownImageKey は objectKey が signImageUpload で userID に発行した形（`{ユーザーID}/{UUID}`）かを返す。
func ownImageKey(userID, objectKey string) bool {
	id, ok := strings.CutPrefix(objectKey, userID+"/")
	parsed, err := uuid.Parse(id)
	return ok && err == nil && parsed.String() == id
}

// statUploadedImage は本人がアップロードした画像を確かめ、その情報を返す。
// 本人のものでない・ない・形式が違うなら errUploadedImageInvalid、大きすぎるなら errUploadedImageTooLarge、
// ストレージの障害ならそのエラーを返す。
func statUploadedImage(ctx context.Context, st storage.Storage, bucket, userID, objectKey string) (storage.ObjectInfo, error) {
	if !ownImageKey(userID, objectKey) {
		return storage.ObjectInfo{}, errUploadedImageInvalid
	}
	info, err := st.Stat(ctx, bucket, objectKey)
	if errors.Is(err, storage.ErrNotFound) {
		return storage.ObjectInfo{}, errUploadedImageInvalid
	}
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	if !validImageContentType(info.ContentType) || info.Size <= 0 {
		return storage.ObjectInfo{}, errUploadedImageInvalid
	}
	if info.Size > imageMaxBytes {
		return storage.ObjectInfo{}, errUploadedImageTooLarge
	}
	return info, nil
}
