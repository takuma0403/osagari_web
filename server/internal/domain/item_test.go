package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/takuma0403/osagari_web/server/internal/domain"
)

func ptr[T any](v T) *T { return &v }

// ADR 0016 の表の写し。
func TestItemStatusRules(t *testing.T) {
	tests := []struct {
		status                       domain.ItemStatus
		canEdit, canWithdraw, canPub bool
	}{
		{domain.ItemStatusDraft, true, true, true},
		{domain.ItemStatusPublished, true, true, false},
		{domain.ItemStatusRequested, false, true, false},
		{domain.ItemStatusInDeal, false, false, false},
		{domain.ItemStatusCompleted, false, false, false},
		{domain.ItemStatusWithdrawn, true, false, true},
		{domain.ItemStatusSuspended, false, false, false},
	}
	if len(tests) != len(domain.ItemStatuses) {
		t.Fatalf("すべてのステータスを確かめていません")
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			it := domain.Item{Status: tt.status}
			if got := it.CanEdit(); got != tt.canEdit {
				t.Errorf("CanEdit=%v want=%v", got, tt.canEdit)
			}
			if got := it.CanWithdraw(); got != tt.canWithdraw {
				t.Errorf("CanWithdraw=%v want=%v", got, tt.canWithdraw)
			}
			if got := it.CanPublish(); got != tt.canPub {
				t.Errorf("CanPublish=%v want=%v", got, tt.canPub)
			}
		})
	}
}

func fullContent() domain.ItemContent {
	return domain.ItemContent{
		Title:       "線形代数入門",
		Category:    ptr(domain.ItemCategoryTextbook),
		Condition:   ptr(domain.ItemConditionGood),
		Price:       ptr[int64](1500),
		Description: ptr("1年前期で使いました"),
		Campus:      ptr(domain.CampusHonjo),
		Textbook:    &domain.TextbookDetails{CourseName: ptr("線形代数"), ISBN: ptr("9784000000000"), HasWriting: ptr(false)},
	}
}

func TestItemContentValidate(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(c *domain.ItemContent)
		images     int
		draftErr   bool
		publishErr bool
	}{
		{"すべてそろっている", func(*domain.ItemContent) {}, 1, false, false},
		{"写真10枚", func(*domain.ItemContent) {}, 10, false, false},
		{"写真11枚", func(*domain.ItemContent) {}, 11, true, true},
		{"写真なし", func(*domain.ItemContent) {}, 0, false, true},
		{"タイトルだけ", func(c *domain.ItemContent) { *c = domain.ItemContent{Title: "下書き"} }, 0, false, true},
		{"タイトルなし", func(c *domain.ItemContent) { c.Title = "" }, 1, true, true},
		{"タイトル100文字", func(c *domain.ItemContent) { c.Title = strings.Repeat("あ", 100) }, 1, false, false},
		{"タイトル101文字", func(c *domain.ItemContent) { c.Title = strings.Repeat("あ", 101) }, 1, true, true},
		{"価格99円", func(c *domain.ItemContent) { c.Price = ptr[int64](99) }, 1, true, true},
		{"価格100円", func(c *domain.ItemContent) { c.Price = ptr[int64](100) }, 1, false, false},
		{"価格100000円", func(c *domain.ItemContent) { c.Price = ptr[int64](100000) }, 1, false, false},
		{"価格100001円", func(c *domain.ItemContent) { c.Price = ptr[int64](100001) }, 1, true, true},
		{"価格なし", func(c *domain.ItemContent) { c.Price = nil }, 1, false, true},
		{"カテゴリなし", func(c *domain.ItemContent) { c.Category = nil; c.Textbook = nil }, 1, false, true},
		{"状態なし", func(c *domain.ItemContent) { c.Condition = nil }, 1, false, true},
		{"説明なし", func(c *domain.ItemContent) { c.Description = nil }, 1, false, true},
		{"キャンパスなし", func(c *domain.ItemContent) { c.Campus = nil }, 1, false, true},
		{"説明2001文字", func(c *domain.ItemContent) { c.Description = ptr(strings.Repeat("a", 2001)) }, 1, true, true},
		{"不明なカテゴリ", func(c *domain.ItemContent) { c.Category = ptr(domain.ItemCategory("game")) }, 1, true, true},
		{"不明な状態", func(c *domain.ItemContent) { c.Condition = ptr(domain.ItemCondition("bad")) }, 1, true, true},
		{"不明なキャンパス", func(c *domain.ItemContent) { c.Campus = ptr(domain.Campus("x")) }, 1, true, true},
		{"家具に教科書の情報", func(c *domain.ItemContent) { c.Category = ptr(domain.ItemCategoryFurnitureAppliance) }, 1, true, true},
		{"教科書に家具の情報", func(c *domain.ItemContent) {
			c.FurnitureAppliance = &domain.FurnitureApplianceDetails{Size: ptr("W60")}
		}, 1, true, true},
		{"カテゴリなしで教科書の情報", func(c *domain.ItemContent) { c.Category = nil }, 1, true, true},
		{"授業名101文字", func(c *domain.ItemContent) { c.Textbook.CourseName = ptr(strings.Repeat("a", 101)) }, 1, true, true},
		{"ISBNが違う", func(c *domain.ItemContent) { c.Textbook.ISBN = ptr("9784000000001") }, 1, true, true},
		{"家具・家電", func(c *domain.ItemContent) {
			c.Category = ptr(domain.ItemCategoryFurnitureAppliance)
			c.Textbook = nil
			c.FurnitureAppliance = &domain.FurnitureApplianceDetails{Size: ptr("W60×D40"), OperationChecked: ptr(true)}
		}, 1, false, false},
		{"サイズ101文字", func(c *domain.ItemContent) {
			c.Category = ptr(domain.ItemCategoryFurnitureAppliance)
			c.Textbook = nil
			c.FurnitureAppliance = &domain.FurnitureApplianceDetails{Size: ptr(strings.Repeat("a", 101))}
		}, 1, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fullContent()
			tt.edit(&c)
			assertInputErr(t, "ValidateDraft", c.ValidateDraft(tt.images), tt.draftErr)
			assertInputErr(t, "ValidateForPublish", c.ValidateForPublish(tt.images), tt.publishErr)
		})
	}
}

func assertInputErr(t *testing.T, name string, err error, want bool) {
	t.Helper()
	if (err != nil) != want {
		t.Errorf("%s err=%v want error=%v", name, err, want)
	}
	var ie *domain.ItemInputError
	if err != nil && !errors.As(err, &ie) {
		t.Errorf("%s err=%T want *ItemInputError", name, err)
	}
}

func TestItemContentNormalize(t *testing.T) {
	c := domain.ItemContent{
		Title:              "  本  ",
		Description:        ptr("   "),
		Textbook:           &domain.TextbookDetails{CourseName: ptr(""), Instructor: ptr(" 佐賀 "), ISBN: ptr("4-00-000000-x")},
		FurnitureAppliance: &domain.FurnitureApplianceDetails{Size: ptr(" ")},
	}.Normalize()
	if c.Title != "本" || c.Description != nil {
		t.Errorf("Title=%q Description=%v", c.Title, c.Description)
	}
	if c.Textbook.CourseName != nil || *c.Textbook.Instructor != "佐賀" || *c.Textbook.ISBN != "400000000X" {
		t.Errorf("Textbook=%+v", c.Textbook)
	}
	if c.FurnitureAppliance.Size != nil {
		t.Errorf("Size=%v", c.FurnitureAppliance.Size)
	}
}

func TestValidISBN(t *testing.T) {
	tests := map[string]bool{
		"9784000000000": true,
		"9784000000001": false,
		"4000000004":    true,
		"080442957X":    true,
		"0306406152":    true,
		"0306406153":    false,
		"978400000000":  false,
		"97840000000a2": false,
		"X306406152":    false,
		"":              false,
	}
	for s, want := range tests {
		if got := domain.ValidISBN(s); got != want {
			t.Errorf("ValidISBN(%q)=%v want=%v", s, got, want)
		}
	}
}
