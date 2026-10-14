// Package repository は GORM による DB アクセスを置く（DB）。
// スキーマは migrations/ の SQL だけで管理し、ここではテーブルと一致する行の構造体と、条件付きUPDATE を含むクエリを持つ。
// go-libsql は1回の Exec で1文しか実行しないため、メソッドは1回に1文ずつ実行する。
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrNotFound は対象の行がないときに返す。
	ErrNotFound = errors.New("repository: 見つかりません")
	// ErrConflict は条件付きUPDATE の更新件数が0（状態が想定と違う・使用済み・期限切れ）のときに返す（DB）。
	ErrConflict = errors.New("repository: 状態が変わったため更新できません")
	// ErrDuplicate は一意制約に違反したときに返す。
	ErrDuplicate = errors.New("repository: すでに存在します")
)

// timeLayout は日時の列（TEXT）の形式。UTC・マイクロ秒までの固定幅にして、文字列の大小で期限を比べられるようにする。
// RFC 3339 として読める。
const timeLayout = "2006-01-02T15:04:05.000000Z"

// dateLayout は日付の列（有効期限など）の形式。
const dateLayout = "2006-01-02"

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

// parseTime は日時の列を読む。go-libsql は日時らしい TEXT を time.Time にして返すため、読み戻した文字列は
// 保存した形（timeLayout）ではなく RFC 3339 になることがある。どちらも RFC 3339 として読める。
func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }

// parseDate は日付の列を読む。保存した形（YYYY-MM-DD）のほか、ドライバが時刻にして返した RFC 3339 も受け付ける。
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse(dateLayout, s); err == nil {
		return t, nil
	}
	t, err := parseTime(s)
	if err != nil {
		return time.Time{}, err
	}
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatTime(*t)
	return &s
}

func parseTimePtr(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := parseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// newID は外部に出しても推測しにくいID（UUID v4）を作る。
func newID() string { return uuid.NewString() }

// isUniqueViolation は一意制約違反かを返す。go-libsql のエラーは gorm.ErrDuplicatedKey に変換されないため文字列で判定する。
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

type txKey struct{}

// Transactor は複数のテーブルにまたがる処理を1つのトランザクションで行う。
// Do に渡した関数の ctx を各 repository に渡すと、同じトランザクションで実行される。
type Transactor struct {
	db *gorm.DB
}

func NewTransactor(db *gorm.DB) *Transactor {
	return &Transactor{db: db}
}

// Do は fn をトランザクションの中で実行する。fn がエラーを返したらロールバックする。
func (t *Transactor) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// conn は ctx にトランザクションがあればそれを、なければ db を返す。
func conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.WithContext(ctx)
}
