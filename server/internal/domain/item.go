package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// 出品の入力の制限（ITEM）。
const (
	ItemTitleMaxLen       = 100
	ItemDescriptionMaxLen = 2000
	// ItemDetailMaxLen は授業名・担当教員・サイズの上限
	ItemDetailMaxLen = 100
	ItemPriceMin     = 100
	ItemPriceMax     = 100000
	// ItemImagesMax は写真の上限。公開するときは1枚以上が必要
	ItemImagesMax = 10
)

// 操作ごとに、できる商品のステータス（ITEM, ADR 0016）。repository の条件付きUPDATE にもこの一覧を渡す。
var (
	// ItemEditableStatuses は編集できるステータス。購入希望あり以降は枠を確保した内容と食い違わないよう編集させない
	ItemEditableStatuses = []ItemStatus{ItemStatusDraft, ItemStatusPublished, ItemStatusWithdrawn}
	// ItemWithdrawableStatuses は取り下げられるステータス。取引中は DEAL-4 の双方同意のキャンセルだけで止める
	ItemWithdrawableStatuses = []ItemStatus{ItemStatusDraft, ItemStatusPublished, ItemStatusRequested}
	// ItemPublishableStatuses は公開（再出品を含む）できるステータス
	ItemPublishableStatuses = []ItemStatus{ItemStatusDraft, ItemStatusWithdrawn}
)

// Item は商品（items）。下書きはタイトルだけ必須なので、ほかの共通項目は nil を許す（ITEM, ADR 0016）。
type Item struct {
	ID       string
	SellerID string
	// ShareID は商品ページの固定URL /items/:shareId に使う（BROWSE）
	ShareID string
	Status  ItemStatus
	ItemContent
	// Images は表示順に並べた写真
	Images []ItemImage
	// PublishedAt は最後に公開した日時。一度も公開していなければ nil
	PublishedAt *time.Time
	// WithdrawnAt は最後に取り下げた日時
	WithdrawnAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ItemContent は出品者が入力する商品の内容（写真を除く）。
type ItemContent struct {
	Title       string
	Category    *ItemCategory
	Condition   *ItemCondition
	Price       *int64
	Description *string
	Campus      *Campus
	// Textbook は Category が textbook のときだけ持つ
	Textbook *TextbookDetails
	// FurnitureAppliance は Category が furniture_appliance のときだけ持つ
	FurnitureAppliance *FurnitureApplianceDetails
}

// TextbookDetails は教科書・参考書の追加項目（どれも任意）。
type TextbookDetails struct {
	CourseName *string
	Instructor *string
	// ISBN はハイフンを除いた10桁か13桁
	ISBN       *string
	HasWriting *bool
}

// FurnitureApplianceDetails は家具・家電の追加項目（どれも任意）。
type FurnitureApplianceDetails struct {
	Size             *string
	OperationChecked *bool
}

// ItemImage は商品写真（item_images）。画像そのものは商品写真のバケットにある（STORAGE）。
type ItemImage struct {
	ID          string
	ItemID      string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	// Position は表示順（0 から）
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ItemInputError は出品の入力が不正なときに返す。Message は利用者に表示できる日本語。
type ItemInputError struct {
	Message string
}

func (e *ItemInputError) Error() string { return "domain: " + e.Message }

func itemInputError(msg string) *ItemInputError { return &ItemInputError{Message: msg} }

// CanEdit は編集できるステータスかを返す。
func (i Item) CanEdit() bool { return slices.Contains(ItemEditableStatuses, i.Status) }

// CanWithdraw は取り下げられるステータスかを返す。
func (i Item) CanWithdraw() bool { return slices.Contains(ItemWithdrawableStatuses, i.Status) }

// CanPublish は公開できるステータスかを返す。
func (i Item) CanPublish() bool { return slices.Contains(ItemPublishableStatuses, i.Status) }

// Normalize は前後の空白を除き、空の任意項目を nil にし、ISBN からハイフンと空白を除いて大文字にする（末尾の X）。
func (c ItemContent) Normalize() ItemContent {
	c.Title = strings.TrimSpace(c.Title)
	c.Description = trimOrNil(c.Description)
	if t := c.Textbook; t != nil {
		isbn := t.ISBN
		if isbn != nil {
			s := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(*isbn))
			isbn = &s
		}
		c.Textbook = &TextbookDetails{
			CourseName: trimOrNil(t.CourseName),
			Instructor: trimOrNil(t.Instructor),
			ISBN:       trimOrNil(isbn),
			HasWriting: t.HasWriting,
		}
	}
	if f := c.FurnitureAppliance; f != nil {
		c.FurnitureAppliance = &FurnitureApplianceDetails{
			Size:             trimOrNil(f.Size),
			OperationChecked: f.OperationChecked,
		}
	}
	return c
}

// ValidateDraft は下書きとして保存できるかを確かめる（タイトルだけ必須。入力した項目は形式と範囲を確かめる）。
// Normalize した後の値に使う。
func (c ItemContent) ValidateDraft(imageCount int) error {
	if c.Title == "" {
		return itemInputError("タイトルを入力してください")
	}
	if utf8.RuneCountInString(c.Title) > ItemTitleMaxLen {
		return itemInputError("タイトルは100文字以内で入力してください")
	}
	if c.Category != nil && !c.Category.Valid() {
		return itemInputError("カテゴリを選び直してください")
	}
	if c.Condition != nil && !c.Condition.Valid() {
		return itemInputError("状態を選び直してください")
	}
	if c.Campus != nil && !c.Campus.Valid() {
		return itemInputError("受け渡し希望キャンパスを選び直してください")
	}
	if c.Price != nil && (*c.Price < ItemPriceMin || *c.Price > ItemPriceMax) {
		return itemInputError("価格は100円から100,000円の範囲で入力してください")
	}
	if c.Description != nil && utf8.RuneCountInString(*c.Description) > ItemDescriptionMaxLen {
		return itemInputError("説明は2,000文字以内で入力してください")
	}
	if imageCount > ItemImagesMax {
		return itemInputError("写真は10枚までです")
	}
	if c.Textbook != nil {
		if c.Category == nil || *c.Category != ItemCategoryTextbook {
			return itemInputError("教科書の情報は、カテゴリが教科書・参考書のときだけ入力できます")
		}
		if tooLong(c.Textbook.CourseName) || tooLong(c.Textbook.Instructor) {
			return itemInputError("授業名と担当教員は100文字以内で入力してください")
		}
		if c.Textbook.ISBN != nil && !ValidISBN(*c.Textbook.ISBN) {
			return itemInputError("ISBN を確認してください（10桁か13桁）")
		}
	}
	if c.FurnitureAppliance != nil {
		if c.Category == nil || *c.Category != ItemCategoryFurnitureAppliance {
			return itemInputError("サイズと動作確認は、カテゴリが家具・家電のときだけ入力できます")
		}
		if tooLong(c.FurnitureAppliance.Size) {
			return itemInputError("サイズは100文字以内で入力してください")
		}
	}
	return nil
}

// ValidateForPublish は公開できるかを確かめる。下書きの確認に加えて、すべての共通項目と写真1枚以上が必要（ADR 0016）。
func (c ItemContent) ValidateForPublish(imageCount int) error {
	if err := c.ValidateDraft(imageCount); err != nil {
		return err
	}
	switch {
	case imageCount == 0:
		return itemInputError("写真を1枚以上追加してください")
	case c.Category == nil:
		return itemInputError("カテゴリを選んでください")
	case c.Condition == nil:
		return itemInputError("状態を選んでください")
	case c.Price == nil:
		return itemInputError("価格を入力してください")
	case c.Description == nil:
		return itemInputError("説明を入力してください")
	case c.Campus == nil:
		return itemInputError("受け渡し希望キャンパスを選んでください")
	}
	return nil
}

// ValidISBN はハイフンを除いた ISBN-10 か ISBN-13 のチェックディジットが正しいかを返す。
func ValidISBN(s string) bool {
	switch len(s) {
	case 10:
		sum := 0
		for i := range 10 {
			var d int
			switch c := s[i]; {
			case c >= '0' && c <= '9':
				d = int(c - '0')
			case i == 9 && c == 'X':
				d = 10
			default:
				return false
			}
			sum += d * (10 - i)
		}
		return sum%11 == 0
	case 13:
		sum := 0
		for i := range 13 {
			c := s[i]
			if c < '0' || c > '9' {
				return false
			}
			d := int(c - '0')
			if i%2 == 1 {
				d *= 3
			}
			sum += d
		}
		return sum%10 == 0
	}
	return false
}

func trimOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func tooLong(s *string) bool { return s != nil && utf8.RuneCountInString(*s) > ItemDetailMaxLen }
