package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
)

var (
	errItemListQuery = apperr.BadRequest("検索の条件を確認してください")
	errItemListPrice = apperr.BadRequest("価格の範囲を確認してください")
	// errItemListCursor は続きの位置（cursor）が読めない。改ざんされたか、古い形式のもの
	errItemListCursor = apperr.BadRequest("続きを読み込めませんでした。最初から読み込み直してください")
)

// ItemListQuery は一覧・検索の入力（BROWSE）。nil の項目は条件にしない。
type ItemListQuery struct {
	Keyword    *string
	Category   *domain.ItemCategory
	CourseName *string
	PriceMin   *int64
	PriceMax   *int64
	Campus     *domain.Campus
	Condition  *domain.ItemCondition
	// Cursor は前のページの NextCursor
	Cursor *string
	// Limit は1回に返す件数。nil なら domain.ItemListDefaultLimit
	Limit *int
}

// ItemListPage は一覧・検索の1ページ。Items の写真は1枚目だけ（閲覧用の署名付きURLつき）。
type ItemListPage struct {
	Items []*ItemView
	// Total は条件に合う商品の総数（Cursor に関係なく数える）
	Total int
	// NextCursor は続きを取るときに渡す値。続きがなければ nil
	NextCursor *string
}

// PublicItemView は商品ページに出す商品（BROWSE）。出品者の情報は学籍番号を含めず、表示する項目だけを持つ（DEAL-1）。
type PublicItemView struct {
	*ItemView
	Seller SellerProfile
	// IsOwn は見ている本人が出品者かどうか
	IsOwn bool
}

// SellerProfile は商品ページに出す出品者の情報。
type SellerProfile struct {
	// DepartmentName は学部・学科の名前。不明なら nil
	DepartmentName *string
	AdmissionYear  int
}

// List は一覧・検索に出す商品（公開中・購入希望あり）を新着順に返す（BROWSE, ADR 0018）。
// 出品者のアカウントが使えない商品は含めない。条件が不正なら 400 bad_request。
func (s *ItemService) List(ctx context.Context, q ItemListQuery) (*ItemListPage, error) {
	filter, err := itemFilter(q)
	if err != nil {
		return nil, err
	}
	limit := domain.ItemListDefaultLimit
	if q.Limit != nil {
		limit = *q.Limit
	}
	if limit < 1 || limit > domain.ItemListMaxLimit {
		return nil, errItemListQuery
	}
	var after *domain.ItemCursor
	if q.Cursor != nil {
		c, err := decodeItemCursor(*q.Cursor)
		if err != nil {
			return nil, errItemListCursor
		}
		after = &c
	}

	// 1件多く読み、続きがあるかを確かめる
	items, total, err := s.d.Items.ListPublic(ctx, filter, after, limit+1, s.d.Clock.Now())
	if err != nil {
		return nil, apperr.Internal(err)
	}
	page := &ItemListPage{Items: make([]*ItemView, 0, min(len(items), limit)), Total: total}
	if len(items) > limit {
		items = items[:limit]
		last := items[limit-1]
		next := encodeItemCursor(domain.ItemCursor{PublishedAt: *last.PublishedAt, ID: last.ID})
		page.NextCursor = &next
	}
	for _, it := range items {
		v, err := s.view(ctx, it)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, v)
	}
	return page, nil
}

// GetPublic は共有用IDで商品ページに出す商品を返す（BROWSE, ADR 0018）。公開中・購入希望ありに加え、
// 取引中・取引完了は売り切れとして返す。それ以外のステータスと、出品者のアカウントが使えない商品は 404（出品者本人も同じ）。
func (s *ItemService) GetPublic(ctx context.Context, u *domain.User, shareID string) (*PublicItemView, error) {
	it, err := s.d.Items.FindByShareID(ctx, shareID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errItemNotFound
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if !it.IsViewable() {
		return nil, errItemNotFound
	}
	seller, err := s.d.Users.FindByID(ctx, it.SellerID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if !seller.IsActive(s.d.Clock.Now()) {
		return nil, errItemNotFound
	}
	v, err := s.view(ctx, it)
	if err != nil {
		return nil, err
	}
	return &PublicItemView{
		ItemView: v,
		Seller:   SellerProfile{DepartmentName: seller.DepartmentName(), AdmissionYear: seller.AdmissionYear},
		IsOwn:    it.SellerID == u.ID,
	}, nil
}

// itemFilter は入力を確かめて検索の条件にする。キーワードは空白で区切り、前後の空白を除いて空なら条件にしない。
func itemFilter(q ItemListQuery) (domain.ItemFilter, error) {
	var f domain.ItemFilter
	if q.Keyword != nil {
		if utf8.RuneCountInString(*q.Keyword) > domain.ItemSearchTextMaxLen {
			return f, errItemListQuery
		}
		f.Keywords = strings.Fields(*q.Keyword)
	}
	if q.CourseName != nil {
		if utf8.RuneCountInString(*q.CourseName) > domain.ItemSearchTextMaxLen {
			return f, errItemListQuery
		}
		if c := strings.TrimSpace(*q.CourseName); c != "" {
			f.CourseName = &c
		}
	}
	if (q.Category != nil && !q.Category.Valid()) || (q.Campus != nil && !q.Campus.Valid()) ||
		(q.Condition != nil && !q.Condition.Valid()) {
		return f, errItemListQuery
	}
	if (q.PriceMin != nil && *q.PriceMin < 0) || (q.PriceMax != nil && *q.PriceMax < 0) ||
		(q.PriceMin != nil && q.PriceMax != nil && *q.PriceMin > *q.PriceMax) {
		return f, errItemListPrice
	}
	f.Category, f.Campus, f.Condition = q.Category, q.Campus, q.Condition
	f.PriceMin, f.PriceMax = q.PriceMin, q.PriceMax
	return f, nil
}

// cursorSep は続きの位置の中の、公開日時と ID の区切り。どちらにも含まれない文字にする。
const cursorSep = "|"

// encodeItemCursor は続きの位置を、URL のクエリにそのまま入れられる文字列にする（公開日時と ID を base64url にしたもの）。
func encodeItemCursor(c domain.ItemCursor) string {
	raw := c.PublishedAt.UTC().Format(time.RFC3339Nano) + cursorSep + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeItemCursor(s string) (domain.ItemCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return domain.ItemCursor{}, err
	}
	at, id, ok := strings.Cut(string(raw), cursorSep)
	if !ok || id == "" {
		return domain.ItemCursor{}, errors.New("service: 続きの位置の形式が違います")
	}
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return domain.ItemCursor{}, err
	}
	return domain.ItemCursor{PublishedAt: t, ID: id}, nil
}
