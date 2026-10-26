package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
	"github.com/takuma0403/osagari_web/server/internal/storage"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

const itemBucket = "item-images"

type itemFixture struct {
	svc *service.ItemService
	// deals は購入希望・承認・却下・受け渡し完了（DEAL-1, DEAL-3）。商品のテストと同じ DB・時計を使う
	deals *service.DealService
	// sales は売上管理と出金申請（PAY, ADR 0029）
	sales    *service.SalesService
	users    *repository.UserRepository
	items    *repository.ItemRepository
	txs      *repository.TransactionRepository
	payments *repository.PaymentRepository
	storage  *storage.Memory
	clock    *clock.Fake
	seller   *domain.User
}

func newItemFixture(t *testing.T) *itemFixture {
	t.Helper()
	gdb := testdb.New(t)
	f := &itemFixture{
		users:    repository.NewUserRepository(gdb),
		items:    repository.NewItemRepository(gdb),
		txs:      repository.NewTransactionRepository(gdb),
		payments: repository.NewPaymentRepository(gdb),
		storage:  storage.NewMemory(),
		clock:    clock.NewFake(t0),
	}
	tx := repository.NewTransactor(gdb)
	gw := payment.NewMock(f.payments, f.clock)
	f.svc = service.NewItemService(service.ItemDeps{
		Tx:           tx,
		Items:        f.items,
		Users:        f.users,
		Transactions: f.txs,
		Payments:     gw,
		Storage:      f.storage,
		Bucket:       itemBucket,
		Clock:        f.clock,
	})
	f.deals = service.NewDealService(service.DealDeps{
		Tx:             tx,
		Items:          f.items,
		Users:          f.users,
		Transactions:   f.txs,
		HandoverTokens: repository.NewHandoverTokenRepository(gdb),
		Settings:       repository.NewSettingRepository(gdb),
		Payments:       gw,
		Storage:        f.storage,
		Bucket:         itemBucket,
		Clock:          f.clock,
	})
	f.sales = service.NewSalesService(service.SalesDeps{
		Tx:           tx,
		Items:        f.items,
		Transactions: f.txs,
		Payouts:      f.payments,
		Payments:     gw,
		Storage:      f.storage,
		Bucket:       itemBucket,
		Clock:        f.clock,
	})
	f.seller = f.createUser(t, email, "26238001", domain.SellerStatusApproved)
	return f
}

func (f *itemFixture) createUser(t *testing.T, email, number string, seller domain.SellerStatus) *domain.User {
	t.Helper()
	u := &domain.User{
		Email: email, StudentNumber: number, AdmissionYear: 2026, DepartmentCode: "238",
		ExpiresOn: time.Date(2030, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:    domain.UserStatusActive, SellerStatus: seller,
		PasswordHash: "hash", CodeVerifiedAt: t0, CreatedAt: t0, UpdatedAt: t0,
	}
	if err := f.users.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

// upload は署名付きURLを発行し、ブラウザがその URL に PUT したことにする。
func (f *itemFixture) upload(t *testing.T, u *domain.User, contentType string, size int64) string {
	t.Helper()
	up, err := f.svc.CreateImageUpload(context.Background(), u, contentType)
	if err != nil {
		t.Fatal(err)
	}
	f.storage.Put(itemBucket, up.ObjectKey, storage.ObjectInfo{Size: size, ContentType: contentType})
	return up.ObjectKey
}

// tick は時計を進める（編集の前後で更新日時を変える）。
func (f *itemFixture) tick() { f.clock.Advance(time.Minute) }

func ptr[T any](v T) *T { return &v }

func fullItem() domain.ItemContent {
	return domain.ItemContent{
		Title:       "線形代数入門",
		Category:    ptr(domain.ItemCategoryTextbook),
		Condition:   ptr(domain.ItemConditionGood),
		Price:       ptr[int64](1500),
		Description: ptr("1年前期で使いました"),
		Campus:      ptr(domain.CampusHonjo),
		Textbook:    &domain.TextbookDetails{ISBN: ptr("978-4-00-000000-0"), HasWriting: ptr(false)},
	}
}

func (f *itemFixture) publish(t *testing.T) *service.ItemView {
	t.Helper()
	key := f.upload(t, f.seller, "image/jpeg", 1024)
	v, err := f.svc.Create(context.Background(), f.seller, fullItem(), []string{key}, true)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *itemFixture) setStatus(t *testing.T, id string, to domain.ItemStatus) {
	t.Helper()
	f.tick()
	if err := f.items.UpdateStatus(context.Background(), id, f.seller.ID, domain.ItemStatuses, nil, to, f.clock.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestItemCreateImageUpload(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)

	up, err := f.svc.CreateImageUpload(ctx, f.seller, "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(up.ObjectKey, f.seller.ID+"/") || up.Method != "PUT" ||
		up.Headers["Content-Type"] != "image/webp" || up.Headers["x-goog-content-length-range"] != "0,5242880" ||
		up.Headers["x-goog-if-generation-match"] != "0" || !up.ExpiresAt.Equal(t0.Add(15*time.Minute)) ||
		!strings.Contains(up.URL, "/"+itemBucket+"/"+up.ObjectKey) {
		t.Errorf("upload = %+v", up)
	}

	_, err = f.svc.CreateImageUpload(ctx, f.seller, "image/png")
	wantCode(t, err, apperr.CodeBadRequest)

	for _, st := range []domain.SellerStatus{domain.SellerStatusNone, domain.SellerStatusPending, domain.SellerStatusRejected} {
		u := &domain.User{ID: "u-" + string(st), SellerStatus: st}
		_, err := f.svc.CreateImageUpload(ctx, u, "image/jpeg")
		wantCode(t, err, service.CodeSellerNotApproved)
	}

	f.storage.Err = errors.New("storage down")
	_, err = f.svc.CreateImageUpload(ctx, f.seller, "image/jpeg")
	wantCode(t, err, apperr.CodeInternal)
}

func TestItemCreate(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)

	// 公開：入力は正規化して保存し、写真は並び順どおり
	k1 := f.upload(t, f.seller, "image/jpeg", 1024)
	k2 := f.upload(t, f.seller, "image/webp", 2048)
	in := fullItem()
	in.Title = "  線形代数入門  "
	v, err := f.svc.Create(ctx, f.seller, in, []string{k2, k1}, true)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != domain.ItemStatusPublished || v.Title != "線形代数入門" || v.SellerID != f.seller.ID ||
		v.PublishedAt == nil || !v.PublishedAt.Equal(t0) || *v.Textbook.ISBN != "9784000000000" || len(v.ShareID) != 12 {
		t.Errorf("公開: got = %+v", v.Item)
	}
	if len(v.Images) != 2 || v.Images[0].ObjectKey != k2 || v.Images[0].ContentType != "image/webp" || v.Images[0].SizeBytes != 2048 ||
		v.Images[1].ObjectKey != k1 || len(v.ImageURLs) != 2 || v.ImageURLs[0].Method != "GET" ||
		!strings.Contains(v.ImageURLs[0].URL, "/"+itemBucket+"/"+k2) || !v.ImageURLs[0].ExpiresAt.Equal(t0.Add(15*time.Minute)) {
		t.Errorf("写真: images=%+v urls=%+v", v.Images, v.ImageURLs)
	}

	// 下書き：タイトルだけでよい
	d, err := f.svc.Create(ctx, f.seller, domain.ItemContent{Title: "下書き"}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != domain.ItemStatusDraft || d.PublishedAt != nil || len(d.Images) != 0 || d.ShareID == v.ShareID {
		t.Errorf("下書き: got = %+v", d.Item)
	}

	other := f.createUser(t, "26238002@edu.cc.saga-u.ac.jp", "26238002", domain.SellerStatusApproved)
	otherKey := f.upload(t, other, "image/jpeg", 1024)
	tooLarge := f.upload(t, f.seller, "image/jpeg", 5<<20+1)
	pngKey := f.seller.ID + "/" + uuid.NewString()
	f.storage.Put(itemBucket, pngKey, storage.ObjectInfo{Size: 10, ContentType: "image/png"})
	notUploaded := f.seller.ID + "/" + uuid.NewString()
	eleven := make([]string, 11)
	for i := range eleven {
		eleven[i] = f.upload(t, f.seller, "image/jpeg", 10)
	}

	tests := []struct {
		name    string
		user    *domain.User
		content domain.ItemContent
		keys    func() []string
		publish bool
		code    apperr.Code
	}{
		{"学生証が未承認", &domain.User{ID: "x", SellerStatus: domain.SellerStatusPending}, fullItem(), nil, true, service.CodeSellerNotApproved},
		{"写真なしで公開", f.seller, fullItem(), func() []string { return nil }, true, apperr.CodeBadRequest},
		{"価格が範囲外", f.seller, func() domain.ItemContent { c := fullItem(); c.Price = ptr[int64](99); return c }(), nil, true, apperr.CodeBadRequest},
		{"下書きでも価格が範囲外", f.seller, domain.ItemContent{Title: "x", Price: ptr[int64](100001)}, func() []string { return nil }, false, apperr.CodeBadRequest},
		{"カテゴリと追加項目が合わない", f.seller, func() domain.ItemContent {
			c := fullItem()
			c.Category = ptr(domain.ItemCategoryGameOther)
			return c
		}(), nil, true, apperr.CodeBadRequest},
		{"写真11枚", f.seller, fullItem(), func() []string { return eleven }, true, apperr.CodeBadRequest},
		{"他人の写真", f.seller, fullItem(), func() []string { return []string{otherKey} }, true, service.CodeInvalidItemImageUpload},
		{"アップロードしていない", f.seller, fullItem(), func() []string { return []string{notUploaded} }, true, service.CodeInvalidItemImageUpload},
		{"形式が違う", f.seller, fullItem(), func() []string { return []string{pngKey} }, true, service.CodeInvalidItemImageUpload},
		{"5MBを超える", f.seller, fullItem(), func() []string { return []string{tooLarge} }, true, service.CodeInvalidItemImageUpload},
		{"同じ写真を2回", f.seller, fullItem(), func() []string { return []string{k1, k1} }, true, service.CodeInvalidItemImageUpload},
		{"ほかの商品の写真", f.seller, fullItem(), func() []string { return []string{k1} }, true, service.CodeInvalidItemImageUpload},
		{"発行していない形の名前", f.seller, fullItem(), func() []string { return []string{f.seller.ID + "/../x"} }, true, service.CodeInvalidItemImageUpload},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := []string{f.upload(t, f.seller, "image/jpeg", 10)}
			if tt.keys != nil {
				keys = tt.keys()
			}
			_, err := f.svc.Create(ctx, tt.user, tt.content, keys, tt.publish)
			wantCode(t, err, tt.code)
		})
	}
}

func TestItemGet(t *testing.T) {
	ctx := context.Background()
	f := newItemFixture(t)
	v := f.publish(t)

	got, err := f.svc.Get(ctx, f.seller, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != v.ID || len(got.ImageURLs) != 1 {
		t.Errorf("got = %+v", got)
	}

	other := f.createUser(t, "26238002@edu.cc.saga-u.ac.jp", "26238002", domain.SellerStatusApproved)
	_, err = f.svc.Get(ctx, other, v.ID)
	wantCode(t, err, apperr.CodeNotFound)
	_, err = f.svc.Get(ctx, f.seller, "missing")
	wantCode(t, err, apperr.CodeNotFound)
}

func TestItemUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("公開中の編集で写真を入れ替える", func(t *testing.T) {
		f := newItemFixture(t)
		v := f.publish(t)
		f.tick()
		kept := v.Images[0].ObjectKey
		added := f.upload(t, f.seller, "image/webp", 300)
		c := fullItem()
		c.Category = ptr(domain.ItemCategoryFurnitureAppliance)
		c.Textbook = nil
		c.FurnitureAppliance = &domain.FurnitureApplianceDetails{Size: ptr("W60"), OperationChecked: ptr(true)}
		got, err := f.svc.Update(ctx, f.seller, v.ID, c, []string{added, kept})
		if err != nil {
			t.Fatal(err)
		}
		if *got.Category != domain.ItemCategoryFurnitureAppliance || got.Textbook != nil || *got.FurnitureAppliance.Size != "W60" ||
			got.Status != domain.ItemStatusPublished || !got.UpdatedAt.Equal(f.clock.Now()) {
			t.Errorf("got = %+v", got.Item)
		}
		if len(got.Images) != 2 || got.Images[0].ObjectKey != added || got.Images[1].ObjectKey != kept ||
			got.Images[1].SizeBytes != 1024 || got.Images[0].SizeBytes != 300 {
			t.Errorf("写真 = %+v", got.Images)
		}
	})

	t.Run("公開中は全項目が必要・下書きと取り下げはタイトルだけ", func(t *testing.T) {
		f := newItemFixture(t)
		v := f.publish(t)
		_, err := f.svc.Update(ctx, f.seller, v.ID, domain.ItemContent{Title: "x"}, nil)
		wantCode(t, err, apperr.CodeBadRequest)

		f.setStatus(t, v.ID, domain.ItemStatusWithdrawn)
		got, err := f.svc.Update(ctx, f.seller, v.ID, domain.ItemContent{Title: "x"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.ItemStatusWithdrawn || got.Price != nil || len(got.Images) != 0 {
			t.Errorf("取り下げ中の編集: got = %+v", got.Item)
		}
	})

	t.Run("編集できないステータス", func(t *testing.T) {
		for _, st := range []domain.ItemStatus{domain.ItemStatusRequested, domain.ItemStatusInDeal, domain.ItemStatusCompleted, domain.ItemStatusSuspended} {
			t.Run(string(st), func(t *testing.T) {
				f := newItemFixture(t)
				v := f.publish(t)
				f.setStatus(t, v.ID, st)
				_, err := f.svc.Update(ctx, f.seller, v.ID, fullItem(), []string{v.Images[0].ObjectKey})
				wantCode(t, err, apperr.CodeInvalidStatusTransition)
			})
		}
	})

	t.Run("権限", func(t *testing.T) {
		f := newItemFixture(t)
		v := f.publish(t)
		other := f.createUser(t, "26238002@edu.cc.saga-u.ac.jp", "26238002", domain.SellerStatusApproved)
		_, err := f.svc.Update(ctx, other, v.ID, fullItem(), []string{v.Images[0].ObjectKey})
		wantCode(t, err, apperr.CodeNotFound)
		_, err = f.svc.Update(ctx, &domain.User{ID: f.seller.ID, SellerStatus: domain.SellerStatusNone}, v.ID, fullItem(), nil)
		wantCode(t, err, service.CodeSellerNotApproved)
	})
}

func TestItemPublishAndWithdraw(t *testing.T) {
	ctx := context.Background()

	t.Run("下書き → 公開 → 取り下げ → 再出品", func(t *testing.T) {
		f := newItemFixture(t)
		d, err := f.svc.Create(ctx, f.seller, domain.ItemContent{Title: "下書き"}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		// 足りない項目があると公開できない
		_, err = f.svc.Publish(ctx, f.seller, d.ID)
		wantCode(t, err, apperr.CodeBadRequest)

		f.tick()
		if _, err := f.svc.Update(ctx, f.seller, d.ID, fullItem(), []string{f.upload(t, f.seller, "image/jpeg", 10)}); err != nil {
			t.Fatal(err)
		}
		f.tick()
		p, err := f.svc.Publish(ctx, f.seller, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if p.Status != domain.ItemStatusPublished || !p.PublishedAt.Equal(f.clock.Now()) {
			t.Errorf("公開: got = %+v", p.Item)
		}
		_, err = f.svc.Publish(ctx, f.seller, d.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)

		f.tick()
		w, err := f.svc.Withdraw(ctx, f.seller, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if w.Status != domain.ItemStatusWithdrawn || !w.WithdrawnAt.Equal(f.clock.Now()) {
			t.Errorf("取り下げ: got = %+v", w.Item)
		}
		_, err = f.svc.Withdraw(ctx, f.seller, d.ID)
		wantCode(t, err, apperr.CodeInvalidStatusTransition)

		f.tick()
		r, err := f.svc.Publish(ctx, f.seller, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if r.Status != domain.ItemStatusPublished || !r.PublishedAt.Equal(f.clock.Now()) {
			t.Errorf("再出品: got = %+v", r.Item)
		}
	})

	t.Run("ステータスごとの取り下げ", func(t *testing.T) {
		tests := []struct {
			status domain.ItemStatus
			code   apperr.Code
		}{
			{domain.ItemStatusDraft, ""},
			{domain.ItemStatusRequested, ""},
			{domain.ItemStatusInDeal, apperr.CodeInvalidStatusTransition},
			{domain.ItemStatusCompleted, apperr.CodeInvalidStatusTransition},
			{domain.ItemStatusSuspended, apperr.CodeInvalidStatusTransition},
		}
		for _, tt := range tests {
			t.Run(string(tt.status), func(t *testing.T) {
				f := newItemFixture(t)
				v := f.publish(t)
				f.setStatus(t, v.ID, tt.status)
				_, err := f.svc.Withdraw(ctx, f.seller, v.ID)
				wantCode(t, err, tt.code)
			})
		}
	})

	t.Run("権限", func(t *testing.T) {
		f := newItemFixture(t)
		v := f.publish(t)
		other := f.createUser(t, "26238002@edu.cc.saga-u.ac.jp", "26238002", domain.SellerStatusApproved)
		_, err := f.svc.Withdraw(ctx, other, v.ID)
		wantCode(t, err, apperr.CodeNotFound)
		f.setStatus(t, v.ID, domain.ItemStatusWithdrawn)
		_, err = f.svc.Publish(ctx, other, v.ID)
		wantCode(t, err, apperr.CodeNotFound)
		_, err = f.svc.Publish(ctx, &domain.User{ID: f.seller.ID, SellerStatus: domain.SellerStatusRejected}, v.ID)
		wantCode(t, err, service.CodeSellerNotApproved)
	})

	t.Run("同時に公開しても1回だけ通る", func(t *testing.T) {
		f := newItemFixture(t)
		v := f.publish(t)
		f.setStatus(t, v.ID, domain.ItemStatusWithdrawn)
		const n = 8
		var wg sync.WaitGroup
		errs := make([]error, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.svc.Publish(ctx, f.seller, v.ID)
			}()
		}
		wg.Wait()
		ok := 0
		for _, err := range errs {
			switch apperr.CodeOf(err) {
			case "":
				ok++
			case apperr.CodeInvalidStatusTransition:
			default:
				t.Errorf("err = %v", err)
			}
		}
		if ok != 1 {
			t.Errorf("成功 = %d, want 1", ok)
		}
	})
}
