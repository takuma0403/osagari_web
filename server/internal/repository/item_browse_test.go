package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

// TestItemRepositoryBrowse は一覧・検索（ListPublic）と共有用IDでの取得（FindByShareID）を確かめる（BROWSE, ADR 0018）。
func TestItemRepositoryBrowse(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	items := repository.NewItemRepository(gdb)
	tx := repository.NewTransactor(gdb)
	now := t0.Add(24 * time.Hour)

	// 出品者：有効・停止・有効期限切れ（日次ジョブで expired になる前）
	active := newUser("", "26238001@edu.cc.saga-u.ac.jp", "26238001")
	suspended := newUser("", "26238002@edu.cc.saga-u.ac.jp", "26238002")
	suspended.Status = domain.UserStatusSuspended
	expired := newUser("", "22238003@edu.cc.saga-u.ac.jp", "22238003")
	expired.ExpiresOn = time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	for _, u := range []*domain.User{active, suspended, expired} {
		if err := users.Create(ctx, u); err != nil {
			t.Fatal(err)
		}
	}

	n := 0
	create := func(seller *domain.User, status domain.ItemStatus, publishedAt time.Time, c domain.ItemContent, imageKeys ...string) *domain.Item {
		t.Helper()
		n++
		it := &domain.Item{
			SellerID: seller.ID, ShareID: "share" + string(rune('A'+n)), Status: status, ItemContent: c,
			PublishedAt: &publishedAt, CreatedAt: t0, UpdatedAt: t0,
		}
		for _, k := range imageKeys {
			it.Images = append(it.Images, newItemImage(seller.ID+"/"+k))
		}
		if err := tx.Do(ctx, func(ctx context.Context) error { return items.Create(ctx, it) }); err != nil {
			t.Fatal(err)
		}
		return it
	}
	content := func(title string, cat domain.ItemCategory, cond domain.ItemCondition, price int64, campus domain.Campus) domain.ItemContent {
		return domain.ItemContent{
			Title: title, Category: &cat, Condition: &cond, Price: &price, Description: ptr(title + "の説明"), Campus: &campus,
		}
	}

	textbook := content("線形代数入門", domain.ItemCategoryTextbook, domain.ItemConditionGood, 1500, domain.CampusHonjo)
	textbook.Textbook = &domain.TextbookDetails{CourseName: ptr("線形代数学I"), Instructor: ptr("山田"), ISBN: ptr("9784000000001")}
	pubTextbook := create(active, domain.ItemStatusPublished, t0.Add(time.Hour), textbook, "a", "b")
	pubGame := create(active, domain.ItemStatusPublished, t0.Add(time.Hour),
		content("ゲーム_機", domain.ItemCategoryGameOther, domain.ItemConditionNewLike, 800, domain.CampusHonjo))
	requested := create(active, domain.ItemStatusRequested, t0.Add(2*time.Hour),
		content("冷蔵庫 100%", domain.ItemCategoryFurnitureAppliance, domain.ItemConditionFair, 6000, domain.CampusNabeshima))
	// 一覧に出ないステータスと、使えない出品者の商品
	other := content("線形代数 別冊", domain.ItemCategoryTextbook, domain.ItemConditionGood, 1000, domain.CampusHonjo)
	for _, s := range []domain.ItemStatus{
		domain.ItemStatusDraft, domain.ItemStatusInDeal, domain.ItemStatusCompleted, domain.ItemStatusWithdrawn, domain.ItemStatusSuspended,
	} {
		create(active, s, t0.Add(3*time.Hour), other)
	}
	create(suspended, domain.ItemStatusPublished, t0.Add(3*time.Hour), other)
	create(expired, domain.ItemStatusPublished, t0.Add(3*time.Hour), other)

	// 新着順。公開日時が同じなら ID の大きい順
	first, second := pubTextbook, pubGame
	if first.ID < second.ID {
		first, second = second, first
	}
	wantAll := []string{requested.ID, first.ID, second.ID}

	ids := func(its []*domain.Item) []string {
		out := make([]string, len(its))
		for i, it := range its {
			out[i] = it.ID
		}
		return out
	}
	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	got, total, err := items.ListPublic(ctx, domain.ItemFilter{}, nil, 10, now)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || !equal(ids(got), wantAll) {
		t.Fatalf("条件なし: total = %d, ids = %v, want 3, %v", total, ids(got), wantAll)
	}
	for _, it := range got {
		switch it.ID {
		case pubTextbook.ID:
			if len(it.Images) != 1 || it.Images[0].ObjectKey != active.ID+"/a" {
				t.Errorf("1枚目の写真だけ: got = %+v", it.Images)
			}
			if it.Textbook == nil || *it.Textbook.CourseName != "線形代数学I" || *it.Price != 1500 {
				t.Errorf("項目: got = %+v", it)
			}
		case requested.ID:
			if len(it.Images) != 0 {
				t.Errorf("写真なし: got = %+v", it.Images)
			}
		}
	}

	// キーセットのページング（公開日時が同じ2件をまたぐ）
	page1, total, err := items.ListPublic(ctx, domain.ItemFilter{}, nil, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	last := page1[len(page1)-1]
	page2, total2, err := items.ListPublic(ctx, domain.ItemFilter{}, &domain.ItemCursor{PublishedAt: *last.PublishedAt, ID: last.ID}, 2, now)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || total2 != 3 || !equal(append(ids(page1), ids(page2)...), wantAll) {
		t.Errorf("ページング: %v + %v (total %d, %d), want %v", ids(page1), ids(page2), total, total2, wantAll)
	}

	cat := func(v domain.ItemCategory) *domain.ItemCategory { return &v }
	tests := []struct {
		name   string
		filter domain.ItemFilter
		want   []string
	}{
		{"キーワード（複数語は AND）", domain.ItemFilter{Keywords: []string{"線形", "山田"}}, []string{pubTextbook.ID}},
		{"キーワード（説明文）", domain.ItemFilter{Keywords: []string{"機の説明"}}, []string{pubGame.ID}},
		{"キーワード（ISBN はハイフンを除いて探す）", domain.ItemFilter{Keywords: []string{"978-4000000001"}}, []string{pubTextbook.ID}},
		{"キーワード（ハイフンだけなら ISBN を探さない）", domain.ItemFilter{Keywords: []string{"-"}}, []string{}},
		{"キーワード（% はそのままの文字）", domain.ItemFilter{Keywords: []string{"%"}}, []string{requested.ID}},
		{"キーワード（_ はそのままの文字）", domain.ItemFilter{Keywords: []string{"_"}}, []string{pubGame.ID}},
		{"キーワード（当てはまらない）", domain.ItemFilter{Keywords: []string{"線形", "冷蔵庫"}}, []string{}},
		{"カテゴリ", domain.ItemFilter{Category: cat(domain.ItemCategoryTextbook)}, []string{pubTextbook.ID}},
		{"授業名", domain.ItemFilter{CourseName: ptr("代数学")}, []string{pubTextbook.ID}},
		{"価格帯（両端を含む）", domain.ItemFilter{PriceMin: ptr[int64](800), PriceMax: ptr[int64](1500)}, []string{first.ID, second.ID}},
		{"価格の下限", domain.ItemFilter{PriceMin: ptr[int64](1501)}, []string{requested.ID}},
		{"キャンパス", domain.ItemFilter{Campus: ptr(domain.CampusNabeshima)}, []string{requested.ID}},
		{"状態", domain.ItemFilter{Condition: ptr(domain.ItemConditionNewLike)}, []string{pubGame.ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := items.ListPublic(ctx, tt.filter, nil, 10, now)
			if err != nil {
				t.Fatal(err)
			}
			if total != len(tt.want) || !equal(ids(got), tt.want) {
				t.Errorf("total = %d, ids = %v, want %v", total, ids(got), tt.want)
			}
		})
	}

	// 共有用IDでの取得はステータスを問わず、写真をすべて返す
	it, err := items.FindByShareID(ctx, pubTextbook.ShareID)
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != pubTextbook.ID || len(it.Images) != 2 || it.Images[1].ObjectKey != active.ID+"/b" {
		t.Errorf("FindByShareID: got = %+v", it)
	}
	if _, err := items.FindByShareID(ctx, "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("ない共有用ID: err = %v, want ErrNotFound", err)
	}
}
