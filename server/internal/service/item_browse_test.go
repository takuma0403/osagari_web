package service_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/service"
)

func TestItemList(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)

	// 3件公開し、1件は下書きのまま
	var published []*service.ItemView
	for range 3 {
		f.tick()
		published = append(published, f.publish(t))
	}
	if _, err := f.svc.Create(ctx, f.seller, domain.ItemContent{Title: "下書き"}, nil, false); err != nil {
		t.Fatal(err)
	}

	page, err := f.svc.List(ctx, service.ItemListQuery{Limit: ptr(2)})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("1ページ目: total = %d, items = %d, next = %v", page.Total, len(page.Items), page.NextCursor)
	}
	// 新着順。写真は1枚目だけで、閲覧用の署名付きURLを付ける
	if page.Items[0].ID != published[2].ID || page.Items[1].ID != published[1].ID {
		t.Errorf("新着順: got = %s, %s", page.Items[0].ID, page.Items[1].ID)
	}
	if v := page.Items[0]; len(v.Images) != 1 || len(v.ImageURLs) != 1 || !strings.Contains(v.ImageURLs[0].URL, v.Images[0].ObjectKey) ||
		!v.ImageURLs[0].ExpiresAt.Equal(f.clock.Now().Add(15*time.Minute)) {
		t.Errorf("写真: got = %+v", v)
	}

	page, err = f.svc.List(ctx, service.ItemListQuery{Limit: ptr(2), Cursor: page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].ID != published[0].ID || page.NextCursor != nil {
		t.Errorf("2ページ目: total = %d, items = %+v, next = %v", page.Total, page.Items, page.NextCursor)
	}

	// 既定の件数と、空白だけの条件は条件にしない
	page, err = f.svc.List(ctx, service.ItemListQuery{Keyword: ptr("　 "), CourseName: ptr(" ")})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Items) != 3 || page.NextCursor != nil {
		t.Errorf("条件なし: total = %d, items = %d", page.Total, len(page.Items))
	}

	page, err = f.svc.List(ctx, service.ItemListQuery{Keyword: ptr("線形 入門"), PriceMin: ptr[int64](1500), PriceMax: ptr[int64](1500)})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 {
		t.Errorf("条件に合う: total = %d", page.Total)
	}
	page, err = f.svc.List(ctx, service.ItemListQuery{Category: ptr(domain.ItemCategoryGameOther)})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 || len(page.Items) != 0 || page.NextCursor != nil {
		t.Errorf("条件に合わない: %+v", page)
	}
}

func TestItemListInvalidQuery(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)

	tests := []struct {
		name string
		q    service.ItemListQuery
	}{
		{"件数が0", service.ItemListQuery{Limit: ptr(0)}},
		{"件数が上限を超える", service.ItemListQuery{Limit: ptr(51)}},
		{"価格の下限が上限より大きい", service.ItemListQuery{PriceMin: ptr[int64](2000), PriceMax: ptr[int64](1000)}},
		{"価格が負", service.ItemListQuery{PriceMin: ptr[int64](-1)}},
		{"カテゴリが不正", service.ItemListQuery{Category: ptr(domain.ItemCategory("book"))}},
		{"キャンパスが不正", service.ItemListQuery{Campus: ptr(domain.Campus("tokyo"))}},
		{"状態が不正", service.ItemListQuery{Condition: ptr(domain.ItemCondition("bad"))}},
		{"キーワードが長すぎる", service.ItemListQuery{Keyword: ptr(strings.Repeat("あ", 101))}},
		{"授業名が長すぎる", service.ItemListQuery{CourseName: ptr(strings.Repeat("あ", 101))}},
		{"続きの位置が base64 でない", service.ItemListQuery{Cursor: ptr("!!!")}},
		{"続きの位置の形式が違う", service.ItemListQuery{Cursor: ptr(base64.RawURLEncoding.EncodeToString([]byte("2026-10-14")))}},
		{"続きの位置の日時が読めない", service.ItemListQuery{Cursor: ptr(base64.RawURLEncoding.EncodeToString([]byte("x|id")))}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.svc.List(ctx, tt.q)
			wantCode(t, err, apperr.CodeBadRequest)
		})
	}
}

func TestItemGetPublic(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)
	viewer := f.createUser(t, "26238002@edu.cc.saga-u.ac.jp", "26238002", domain.SellerStatusNone)

	// 一覧と同じ範囲に加え、取引中・取引完了は売り切れとして見せる。それ以外は出品者本人にも 404
	tests := []struct {
		status domain.ItemStatus
		ok     bool
	}{
		{domain.ItemStatusPublished, true},
		{domain.ItemStatusRequested, true},
		{domain.ItemStatusInDeal, true},
		{domain.ItemStatusCompleted, true},
		{domain.ItemStatusDraft, false},
		{domain.ItemStatusWithdrawn, false},
		{domain.ItemStatusSuspended, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			v := f.publish(t)
			f.setStatus(t, v.ID, tt.status)
			for _, u := range []*domain.User{viewer, f.seller} {
				got, err := f.svc.GetPublic(ctx, u, v.ShareID)
				if !tt.ok {
					wantCode(t, err, apperr.CodeNotFound)
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if got.ID != v.ID || got.Status != tt.status || len(got.ImageURLs) != 1 || got.IsOwn != (u.ID == f.seller.ID) {
					t.Errorf("got = %+v", got)
				}
				// 商品IDは本人の出品のときだけ返す
				if got.IsOwn != (got.OwnItemID != nil) || (got.OwnItemID != nil && *got.OwnItemID != v.ID) {
					t.Errorf("OwnItemID = %v, IsOwn = %v", got.OwnItemID, got.IsOwn)
				}
			}
		})
	}

	_, err := f.svc.GetPublic(ctx, viewer, "missing")
	wantCode(t, err, apperr.CodeNotFound)
}

func TestItemGetPublicSeller(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)
	viewer := f.createUser(t, "26238002@edu.cc.saga-u.ac.jp", "26238002", domain.SellerStatusNone)

	// 出品者の学部・学科と入学年度だけを出す（学籍番号は出さない。DEAL-1）
	dept := "238"
	seller := &domain.User{
		Email: "25238003@edu.cc.saga-u.ac.jp", StudentNumber: "25238003", AdmissionYear: 2025, DepartmentCode: "238", Department: &dept,
		ExpiresOn: time.Date(2029, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.UserStatusActive, SellerStatus: domain.SellerStatusApproved,
		PasswordHash: "hash", CodeVerifiedAt: t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := f.users.Create(ctx, seller); err != nil {
		t.Fatal(err)
	}
	key := f.upload(t, seller, "image/jpeg", 1024)
	v, err := f.svc.Create(ctx, seller, fullItem(), []string{key}, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.GetPublic(ctx, viewer, v.ShareID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Seller.DepartmentName == nil || *got.Seller.DepartmentName != "理工学部 理工学科" || got.Seller.AdmissionYear != 2025 || got.IsOwn || got.OwnItemID != nil {
		t.Errorf("出品者: got = %+v", got.Seller)
	}

	// 出品者の有効期限（2029-03-31）を日本時間で過ぎたら、日次ジョブで失効になる前でも見せない
	f.clock.Set(time.Date(2029, 3, 31, 14, 59, 0, 0, time.UTC))
	if _, err := f.svc.GetPublic(ctx, viewer, v.ShareID); err != nil {
		t.Errorf("有効期限の日の終わりまでは見せる: err = %v", err)
	}
	if page, err := f.svc.List(ctx, service.ItemListQuery{}); err != nil || page.Total != 1 {
		t.Errorf("有効期限の日の終わりまでは一覧に出す: page = %+v, err = %v", page, err)
	}
	f.clock.Set(time.Date(2029, 3, 31, 15, 0, 0, 0, time.UTC))
	_, err = f.svc.GetPublic(ctx, viewer, v.ShareID)
	wantCode(t, err, apperr.CodeNotFound)
	if page, err := f.svc.List(ctx, service.ItemListQuery{}); err != nil || page.Total != 0 {
		t.Errorf("有効期限を過ぎたら一覧に出さない: page = %+v, err = %v", page, err)
	}
}
