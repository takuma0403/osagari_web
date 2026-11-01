package repository

import (
	"context"
	"fmt"
	"strings"
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
	return r.withImages(ctx, rows[0])
}

// FindByShareID は共有用IDで商品を写真つきで探す（BROWSE）。ステータスは問わない。なければ ErrNotFound。
func (r *ItemRepository) FindByShareID(ctx context.Context, shareID string) (*domain.Item, error) {
	var rows []itemRow
	if err := conn(ctx, r.db).Where("share_id = ?", shareID).Limit(1).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return r.withImages(ctx, rows[0])
}

// withImages は商品の行に、すべての写真を表示順で付けて返す。
func (r *ItemRepository) withImages(ctx context.Context, row itemRow) (*domain.Item, error) {
	it, err := row.toDomain()
	if err != nil {
		return nil, err
	}
	var imgRows []itemImageRow
	if err := conn(ctx, r.db).Where("item_id = ?", it.ID).Order("position ASC").Find(&imgRows).Error; err != nil {
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

// ListPublic は一覧・検索に出す商品（domain.ItemListableStatuses）のうち、条件 f に合うものを新着順に limit 件まで返す（BROWSE, ADR 0018）。
// 出品者のアカウントが now の時点で使えない（有効でない・有効期限を過ぎた。domain.User.IsActive）商品は含めない。
// after を渡すと、その位置より後ろを返す。写真は1枚目だけを Images に入れる。total は after に関係なく条件に合う件数。
func (r *ItemRepository) ListPublic(ctx context.Context, f domain.ItemFilter, after *domain.ItemCursor, limit int, now time.Time) ([]*domain.Item, int, error) {
	var total int64
	if err := r.publicQuery(ctx, f, now).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.publicQuery(ctx, f, now)
	if after != nil {
		at := formatTime(after.PublishedAt)
		q = q.Where("(items.published_at < ? OR (items.published_at = ? AND items.id < ?))", at, at, after.ID)
	}
	var rows []itemRow
	if err := q.Select("items.*").Order("items.published_at DESC, items.id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := r.withThumbnails(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

// FindByIDs は ID で商品をまとめて探し、ID ごとの対応で返す（取引一覧。ADR 0028）。ステータスは問わない。
// 写真は1枚目だけを Images に入れる。見つからない ID は結果に含めない。
func (r *ItemRepository) FindByIDs(ctx context.Context, ids []string) (map[string]*domain.Item, error) {
	out := make(map[string]*domain.Item, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []itemRow
	if err := conn(ctx, r.db).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	items, err := r.withThumbnails(ctx, rows)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		out[it.ID] = it
	}
	return out, nil
}

// ListBySeller は出品者 sellerID のすべての商品（ステータスを問わない）を、最後に更新した順（updated_at の新しい順、同じなら ID の大きい順）で返す。
// 写真は1枚目だけを Images に入れる（マイページ。AUTH-3, ADR 0029）。
func (r *ItemRepository) ListBySeller(ctx context.Context, sellerID string) ([]*domain.Item, error) {
	var rows []itemRow
	if err := conn(ctx, r.db).Where("seller_id = ?", sellerID).Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.withThumbnails(ctx, rows)
}

// ListForAdmin はすべての商品（下書き・取り下げ・停止を含む）を、作った日時の新しい順（同じなら ID の大きい順）に
// after より後ろから limit 件返す。写真は1枚目だけ（ADMIN）。
func (r *ItemRepository) ListForAdmin(ctx context.Context, after *domain.AdminCursor, limit int) ([]*domain.Item, error) {
	q := conn(ctx, r.db)
	if after != nil {
		at := formatTime(after.At)
		q = q.Where("(created_at < ? OR (created_at = ? AND id < ?))", at, at, after.ID)
	}
	var rows []itemRow
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.withThumbnails(ctx, rows)
}

// withThumbnails は商品の行の並びに、1枚目の写真だけを付けて返す（一覧のサムネイル）。写真がなければ Images は空。
func (r *ItemRepository) withThumbnails(ctx context.Context, rows []itemRow) ([]*domain.Item, error) {
	items := make([]*domain.Item, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		it, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		it.Images = []domain.ItemImage{}
		items = append(items, it)
		ids = append(ids, it.ID)
	}
	if len(ids) == 0 {
		return items, nil
	}
	var imgRows []itemImageRow
	if err := conn(ctx, r.db).Where("item_id IN ? AND position = 0", ids).Find(&imgRows).Error; err != nil {
		return nil, err
	}
	byItem := make(map[string]domain.ItemImage, len(imgRows))
	for _, ir := range imgRows {
		img, err := ir.toDomain()
		if err != nil {
			return nil, err
		}
		byItem[img.ItemID] = img
	}
	for _, it := range items {
		if img, ok := byItem[it.ID]; ok {
			it.Images = []domain.ItemImage{img}
		}
	}
	return items, nil
}

// publicQuery は一覧・検索の条件（並びと位置を除く）を組み立てる。users と結合するので列には items. を付ける。
func (r *ItemRepository) publicQuery(ctx context.Context, f domain.ItemFilter, now time.Time) *gorm.DB {
	q := conn(ctx, r.db).Model(&itemRow{}).
		Joins("JOIN users ON users.id = items.seller_id").
		Where("items.status IN ?", enumStrings(domain.ItemListableStatuses)).
		// User.IsActive と同じ判定：有効で、日本時間の今日が有効期限を過ぎていない
		Where("users.status = ? AND users.expires_on >= ?", string(domain.UserStatusActive), domain.JSTDate(now).Format(dateLayout))
	for _, w := range f.Keywords {
		p := likePattern(w)
		cond := `items.title LIKE ? ESCAPE '\' OR items.description LIKE ? ESCAPE '\' OR items.course_name LIKE ? ESCAPE '\'` +
			` OR items.instructor LIKE ? ESCAPE '\'`
		args := []any{p, p, p, p}
		// ISBN はハイフンを除いて保存しているので、語からもハイフンを除いて探す。ハイフンだけの語では ISBN を探さない
		if isbn := strings.ReplaceAll(w, "-", ""); isbn != "" {
			cond += ` OR items.isbn LIKE ? ESCAPE '\'`
			args = append(args, likePattern(isbn))
		}
		q = q.Where("("+cond+")", args...)
	}
	if f.Category != nil {
		q = q.Where("items.category = ?", string(*f.Category))
	}
	if f.CourseName != nil {
		q = q.Where(`items.course_name LIKE ? ESCAPE '\'`, likePattern(*f.CourseName))
	}
	if f.PriceMin != nil {
		q = q.Where("items.price >= ?", *f.PriceMin)
	}
	if f.PriceMax != nil {
		q = q.Where("items.price <= ?", *f.PriceMax)
	}
	if f.Campus != nil {
		q = q.Where("items.campus = ?", string(*f.Campus))
	}
	if f.Condition != nil {
		q = q.Where("items.condition = ?", string(*f.Condition))
	}
	return q
}

// likeEscaper は LIKE の特殊文字（% と _）とエスケープ文字（\）をそのままの文字として探せるようにする。
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// likePattern は s を部分一致で探す LIKE のパターンにする（ESCAPE '\' と一緒に使う）。
func likePattern(s string) string { return "%" + likeEscaper.Replace(s) + "%" }

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

// UpdateDealStatus は取引の進み具合に合わせて商品のステータスを変える（購入希望あり・取引中・公開中に戻す。DEAL-1）。
// ステータスが from のどれかのときだけ変え、違えば ErrConflict。UpdateStatus と違い公開日時は変えない（新着順を保つ。ADR 0022）。
func (r *ItemRepository) UpdateDealStatus(ctx context.Context, id, sellerID string, from []domain.ItemStatus, to domain.ItemStatus, now time.Time) error {
	return r.conditionalUpdate(ctx, id, sellerID, from, nil, map[string]any{"status": string(to), "updated_at": formatTime(now)})
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
