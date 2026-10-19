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

func ptr[T any](v T) *T { return &v }

func newItemImage(objectKey string) domain.ItemImage {
	return domain.ItemImage{ObjectKey: objectKey, ContentType: "image/webp", SizeBytes: 2048}
}

func TestItemRepository(t *testing.T) {
	ctx := context.Background()
	gdb := testdb.New(t)
	users := repository.NewUserRepository(gdb)
	items := repository.NewItemRepository(gdb)
	tx := repository.NewTransactor(gdb)

	u := newUser("", email, "26238001")
	if err := users.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	// 下書き：タイトルだけ
	draft := &domain.Item{
		SellerID: u.ID, ShareID: "share0000001", Status: domain.ItemStatusDraft,
		ItemContent: domain.ItemContent{Title: "下書き"}, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := items.Create(ctx, draft); err != nil {
		t.Fatal(err)
	}
	got, err := items.FindByID(ctx, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "下書き" || got.Status != domain.ItemStatusDraft || got.Category != nil || got.Price != nil ||
		got.Textbook != nil || got.FurnitureAppliance != nil || got.PublishedAt != nil || len(got.Images) != 0 || !got.CreatedAt.Equal(t0) {
		t.Errorf("下書き: got = %+v", got)
	}

	// 公開中：すべての項目と写真
	pub := &domain.Item{
		SellerID: u.ID, ShareID: "share0000002", Status: domain.ItemStatusPublished,
		ItemContent: domain.ItemContent{
			Title: "線形代数", Category: ptr(domain.ItemCategoryTextbook), Condition: ptr(domain.ItemConditionFair),
			Price: ptr[int64](1200), Description: ptr("説明"), Campus: ptr(domain.CampusNabeshima),
			Textbook: &domain.TextbookDetails{CourseName: ptr("線形代数"), ISBN: ptr("9784000000000"), HasWriting: ptr(false)},
		},
		Images:      []domain.ItemImage{newItemImage(u.ID + "/b"), newItemImage(u.ID + "/a")},
		PublishedAt: &t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := tx.Do(ctx, func(ctx context.Context) error { return items.Create(ctx, pub) }); err != nil {
		t.Fatal(err)
	}
	got, err = items.FindByID(ctx, pub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Category != domain.ItemCategoryTextbook || *got.Condition != domain.ItemConditionFair || *got.Price != 1200 ||
		*got.Campus != domain.CampusNabeshima || got.PublishedAt == nil || !got.PublishedAt.Equal(t0) {
		t.Errorf("公開中: got = %+v", got)
	}
	if tb := got.Textbook; tb == nil || *tb.CourseName != "線形代数" || *tb.ISBN != "9784000000000" || *tb.HasWriting || tb.Instructor != nil {
		t.Errorf("教科書の情報: got = %+v", got.Textbook)
	}
	if len(got.Images) != 2 || got.Images[0].ObjectKey != u.ID+"/b" || got.Images[1].ObjectKey != u.ID+"/a" ||
		got.Images[0].Position != 0 || got.Images[1].Position != 1 || got.Images[0].ContentType != "image/webp" {
		t.Errorf("写真の表示順: got = %+v", got.Images)
	}

	// ほかの商品と同じ写真は使えない
	dup := &domain.Item{
		SellerID: u.ID, ShareID: "share0000003", Status: domain.ItemStatusDraft, ItemContent: domain.ItemContent{Title: "x"},
		Images: []domain.ItemImage{newItemImage(u.ID + "/a")}, CreatedAt: t0, UpdatedAt: t0,
	}
	err = tx.Do(ctx, func(ctx context.Context) error { return items.Create(ctx, dup) })
	if !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("同じ写真: err = %v, want ErrDuplicate", err)
	}
	if _, err := items.FindByID(ctx, dup.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("ロールバックされていない: err = %v", err)
	}
	// 共有用IDの重複
	if err := items.Create(ctx, &domain.Item{SellerID: u.ID, ShareID: "share0000001", Status: domain.ItemStatusDraft,
		ItemContent: domain.ItemContent{Title: "x"}, CreatedAt: t0, UpdatedAt: t0}); !errors.Is(err, repository.ErrDuplicate) {
		t.Errorf("共有用IDの重複: err = %v, want ErrDuplicate", err)
	}

	// 編集：家具・家電に変えると教科書の列は空になる。写真は入れ替わる
	t1 := t0.Add(time.Hour)
	c := domain.ItemContent{
		Title: "机", Category: ptr(domain.ItemCategoryFurnitureAppliance), Condition: ptr(domain.ItemConditionGood),
		Price: ptr[int64](3000), Description: ptr("机です"), Campus: ptr(domain.CampusHonjo),
		FurnitureAppliance: &domain.FurnitureApplianceDetails{Size: ptr("W60"), OperationChecked: ptr(true)},
	}
	err = tx.Do(ctx, func(ctx context.Context) error {
		if err := items.UpdateContent(ctx, pub.ID, u.ID, domain.ItemStatusPublished, t0, c, t1); err != nil {
			return err
		}
		return items.ReplaceImages(ctx, pub.ID, []domain.ItemImage{newItemImage(u.ID + "/a"), newItemImage(u.ID + "/c")}, t1)
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err = items.FindByID(ctx, pub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "机" || got.Textbook != nil || got.FurnitureAppliance == nil || *got.FurnitureAppliance.Size != "W60" ||
		!*got.FurnitureAppliance.OperationChecked || !got.UpdatedAt.Equal(t1) {
		t.Errorf("編集後: got = %+v", got)
	}
	if len(got.Images) != 2 || got.Images[0].ObjectKey != u.ID+"/a" || got.Images[1].ObjectKey != u.ID+"/c" {
		t.Errorf("編集後の写真: got = %+v", got.Images)
	}

	// 出品者でない・想定外のステータスは ErrConflict
	if err := items.UpdateContent(ctx, pub.ID, "other", domain.ItemStatusPublished, t1, c, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("出品者でない: err = %v, want ErrConflict", err)
	}
	if err := items.UpdateContent(ctx, pub.ID, u.ID, domain.ItemStatusDraft, t1, c, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("読んだ後にステータスが変わった: err = %v, want ErrConflict", err)
	}
	if err := items.UpdateContent(ctx, pub.ID, u.ID, domain.ItemStatusPublished, t0, c, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("読んだ後に編集された: err = %v, want ErrConflict", err)
	}
	if err := items.UpdateStatus(ctx, pub.ID, u.ID, domain.ItemPublishableStatuses, nil, domain.ItemStatusPublished, t1); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("公開中を公開: err = %v, want ErrConflict", err)
	}

	// 取り下げ → 再出品
	t2 := t1.Add(time.Hour)
	if err := items.UpdateStatus(ctx, pub.ID, u.ID, domain.ItemWithdrawableStatuses, nil, domain.ItemStatusWithdrawn, t2); err != nil {
		t.Fatal(err)
	}
	got, _ = items.FindByID(ctx, pub.ID)
	if got.Status != domain.ItemStatusWithdrawn || got.WithdrawnAt == nil || !got.WithdrawnAt.Equal(t2) || !got.PublishedAt.Equal(t0) {
		t.Errorf("取り下げ後: got = %+v", got)
	}
	t3 := t2.Add(time.Hour)
	if err := items.UpdateStatus(ctx, pub.ID, u.ID, domain.ItemPublishableStatuses, &t1, domain.ItemStatusPublished, t3); !errors.Is(err, repository.ErrConflict) {
		t.Errorf("公開の確認の後に編集された: err = %v, want ErrConflict", err)
	}
	if err := items.UpdateStatus(ctx, pub.ID, u.ID, domain.ItemPublishableStatuses, &t2, domain.ItemStatusPublished, t3); err != nil {
		t.Fatal(err)
	}
	got, _ = items.FindByID(ctx, pub.ID)
	if got.Status != domain.ItemStatusPublished || !got.PublishedAt.Equal(t3) {
		t.Errorf("再出品後: got = %+v", got)
	}

	if _, err := items.FindByID(ctx, "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("ない商品: err = %v, want ErrNotFound", err)
	}
}
