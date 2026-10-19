package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

// itemRow は items の1行（migrations/000010_create_items）。真偽値の列は 0/1 の INTEGER。
type itemRow struct {
	ID               string
	SellerID         string
	ShareID          string
	Status           string
	Title            string
	Category         *string
	Condition        *string
	Price            *int64
	Description      *string
	Campus           *string
	CourseName       *string
	Instructor       *string
	Isbn             *string
	HasWriting       *int64
	Size             *string
	OperationChecked *int64
	PublishedAt      *string
	WithdrawnAt      *string
	CreatedAt        string
	UpdatedAt        string
}

func (itemRow) TableName() string { return "items" }

// itemImageRow は item_images の1行（migrations/000011_create_item_images）。
type itemImageRow struct {
	ID          string
	ItemID      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Position    int
	CreatedAt   string
	UpdatedAt   string
}

func (itemImageRow) TableName() string { return "item_images" }

// contentColumns は編集で書き換える列（ItemContent）と値を返す。
func contentColumns(c domain.ItemContent) map[string]any {
	cols := map[string]any{
		"title":             c.Title,
		"category":          enumPtr(c.Category),
		"condition":         enumPtr(c.Condition),
		"price":             c.Price,
		"description":       c.Description,
		"campus":            enumPtr(c.Campus),
		"course_name":       nil,
		"instructor":        nil,
		"isbn":              nil,
		"has_writing":       nil,
		"size":              nil,
		"operation_checked": nil,
	}
	if t := c.Textbook; t != nil {
		cols["course_name"] = t.CourseName
		cols["instructor"] = t.Instructor
		cols["isbn"] = t.ISBN
		cols["has_writing"] = boolToInt(t.HasWriting)
	}
	if f := c.FurnitureAppliance; f != nil {
		cols["size"] = f.Size
		cols["operation_checked"] = boolToInt(f.OperationChecked)
	}
	return cols
}

func (r itemRow) toDomain() (*domain.Item, error) {
	it := &domain.Item{
		ID:       r.ID,
		SellerID: r.SellerID,
		ShareID:  r.ShareID,
		Status:   domain.ItemStatus(r.Status),
		ItemContent: domain.ItemContent{
			Title:       r.Title,
			Category:    (*domain.ItemCategory)(r.Category),
			Condition:   (*domain.ItemCondition)(r.Condition),
			Price:       r.Price,
			Description: r.Description,
			Campus:      (*domain.Campus)(r.Campus),
		},
	}
	if r.CourseName != nil || r.Instructor != nil || r.Isbn != nil || r.HasWriting != nil {
		it.Textbook = &domain.TextbookDetails{
			CourseName: r.CourseName, Instructor: r.Instructor, ISBN: r.Isbn, HasWriting: intToBool(r.HasWriting),
		}
	}
	if r.Size != nil || r.OperationChecked != nil {
		it.FurnitureAppliance = &domain.FurnitureApplianceDetails{
			Size: r.Size, OperationChecked: intToBool(r.OperationChecked),
		}
	}
	var err error
	if it.CreatedAt, err = parseTime(r.CreatedAt); err != nil {
		return nil, fmt.Errorf("repository: items.created_at: %w", err)
	}
	if it.UpdatedAt, err = parseTime(r.UpdatedAt); err != nil {
		return nil, fmt.Errorf("repository: items.updated_at: %w", err)
	}
	if it.PublishedAt, err = parseTimePtr(r.PublishedAt); err != nil {
		return nil, fmt.Errorf("repository: items.published_at: %w", err)
	}
	if it.WithdrawnAt, err = parseTimePtr(r.WithdrawnAt); err != nil {
		return nil, fmt.Errorf("repository: items.withdrawn_at: %w", err)
	}
	return it, nil
}

func (r itemImageRow) toDomain() (domain.ItemImage, error) {
	img := domain.ItemImage{
		ID: r.ID, ItemID: r.ItemID, ObjectKey: r.ObjectKey, ContentType: r.ContentType, SizeBytes: r.SizeBytes, Position: r.Position,
	}
	var err error
	if img.CreatedAt, err = parseTime(r.CreatedAt); err != nil {
		return img, fmt.Errorf("repository: item_images.created_at: %w", err)
	}
	if img.UpdatedAt, err = parseTime(r.UpdatedAt); err != nil {
		return img, fmt.Errorf("repository: item_images.updated_at: %w", err)
	}
	return img, nil
}

// ItemRepository は items と item_images を読み書きする（ITEM）。
type ItemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// Create は商品と写真を保存する。ID が空なら採番する。写真も含めて1つのトランザクションで呼ぶ。
// 共有用IDや写真のオブジェクト名がすでにあるなら ErrDuplicate。
func (r *ItemRepository) Create(ctx context.Context, it *domain.Item) error {
	if it.ID == "" {
		it.ID = newID()
	}
	// 内容の列は編集と同じ contentColumns で組み立て、列と値の対応を一か所にまとめる
	cols := contentColumns(it.ItemContent)
	cols["id"] = it.ID
	cols["seller_id"] = it.SellerID
	cols["share_id"] = it.ShareID
	cols["status"] = string(it.Status)
	cols["published_at"] = formatTimePtr(it.PublishedAt)
	cols["withdrawn_at"] = formatTimePtr(it.WithdrawnAt)
	cols["created_at"] = formatTime(it.CreatedAt)
	cols["updated_at"] = formatTime(it.UpdatedAt)
	err := conn(ctx, r.db).Model(&itemRow{}).Create(cols).Error
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	if err != nil {
		return err
	}
	return r.ReplaceImages(ctx, it.ID, it.Images, it.CreatedAt)
}

// FindByID は ID で商品を写真つきで探す。なければ ErrNotFound。
func (r *ItemRepository) FindByID(ctx context.Context, id string) (*domain.Item, error) {
	var rows []itemRow
	if err := conn(ctx, r.db).Where("id = ?", id).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	it, err := rows[0].toDomain()
	if err != nil {
		return nil, err
	}
	var imgRows []itemImageRow
	if err := conn(ctx, r.db).Where("item_id = ?", id).Order("position ASC").Find(&imgRows).Error; err != nil {
		return nil, err
	}
	it.Images = make([]domain.ItemImage, 0, len(imgRows))
	for _, ir := range imgRows {
		img, err := ir.toDomain()
		if err != nil {
			return nil, err
		}
		it.Images = append(it.Images, img)
	}
	return it, nil
}

// UpdateContent は出品者の商品が、読んだときのステータス status と更新日時 updatedAt のままのときだけ内容を書き換える
// （条件付きUPDATE。DB）。読んだ後にステータスや内容が変わっていれば ErrConflict。
// 確かめた入力の決まり（下書きか公開中か）と、実際に書き換える商品のステータスを食い違わせないため。
// 写真は ReplaceImages で同じトランザクションの中で入れ替える。
func (r *ItemRepository) UpdateContent(ctx context.Context, id, sellerID string, status domain.ItemStatus, updatedAt time.Time, c domain.ItemContent, now time.Time) error {
	cols := contentColumns(c)
	cols["updated_at"] = formatTime(now)
	return r.conditionalUpdate(ctx, id, sellerID, []domain.ItemStatus{status}, &updatedAt, cols)
}

// UpdateStatus は出品者の商品のステータスが from のどれかのときだけ to にする（条件付きUPDATE。DB）。
// unchangedSince を渡すと、更新日時がその値のまま（読んだ後に編集されていない）のときだけ更新する（公開時の入力の確認に使う）。
// 公開なら published_at、取り下げなら withdrawn_at に now を入れる。当てはまらなければ ErrConflict。
func (r *ItemRepository) UpdateStatus(ctx context.Context, id, sellerID string, from []domain.ItemStatus, unchangedSince *time.Time, to domain.ItemStatus, now time.Time) error {
	ts := formatTime(now)
	cols := map[string]any{"status": string(to), "updated_at": ts}
	switch to {
	case domain.ItemStatusPublished:
		cols["published_at"] = ts
	case domain.ItemStatusWithdrawn:
		cols["withdrawn_at"] = ts
	}
	return r.conditionalUpdate(ctx, id, sellerID, from, unchangedSince, cols)
}

func (r *ItemRepository) conditionalUpdate(ctx context.Context, id, sellerID string, from []domain.ItemStatus, unchangedSince *time.Time, cols map[string]any) error {
	q := conn(ctx, r.db).Model(&itemRow{}).
		Where("id = ? AND seller_id = ? AND status IN ?", id, sellerID, enumStrings(from))
	if unchangedSince != nil {
		q = q.Where("updated_at = ?", formatTime(*unchangedSince))
	}
	res := q.Updates(cols)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

// ReplaceImages は商品の写真を images（表示順）に入れ替える。1つのトランザクションの中で呼ぶ。
// 写真の ID が空なら採番し、Position は並び順で振り直す。ほかの商品が使っているオブジェクト名なら ErrDuplicate。
func (r *ItemRepository) ReplaceImages(ctx context.Context, itemID string, images []domain.ItemImage, now time.Time) error {
	if err := conn(ctx, r.db).Where("item_id = ?", itemID).Delete(&itemImageRow{}).Error; err != nil {
		return err
	}
	ts := formatTime(now)
	for i := range images {
		img := &images[i]
		img.ID = newID()
		img.ItemID = itemID
		img.Position = i
		row := itemImageRow{
			ID: img.ID, ItemID: itemID, ObjectKey: img.ObjectKey, ContentType: img.ContentType,
			SizeBytes: img.SizeBytes, Position: i, CreatedAt: ts, UpdatedAt: ts,
		}
		err := conn(ctx, r.db).Create(&row).Error
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func enumPtr[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func enumStrings[T ~string](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

func boolToInt(b *bool) *int64 {
	if b == nil {
		return nil
	}
	var n int64
	if *b {
		n = 1
	}
	return &n
}

func intToBool(n *int64) *bool {
	if n == nil {
		return nil
	}
	b := *n != 0
	return &b
}
