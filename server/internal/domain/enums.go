package domain

// UserStatus はユーザーの状態。
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"    // 有効
	UserStatusSuspended UserStatus = "suspended" // 停止
	UserStatusExpired   UserStatus = "expired"   // 失効
	UserStatusDeleted   UserStatus = "deleted"   // 削除
)

// UserStatuses は UserStatus のすべての値。
var UserStatuses = []UserStatus{UserStatusActive, UserStatusSuspended, UserStatusExpired, UserStatusDeleted}

func (v UserStatus) Valid() bool { return enumValid(v, UserStatuses) }

// SellerStatus は出品者の承認状態。
type SellerStatus string

const (
	SellerStatusNone     SellerStatus = "none"     // 未登録
	SellerStatusPending  SellerStatus = "pending"  // 審査中
	SellerStatusApproved SellerStatus = "approved" // 承認
	SellerStatusRejected SellerStatus = "rejected" // 差し戻し
)

// SellerStatuses は SellerStatus のすべての値。
var SellerStatuses = []SellerStatus{SellerStatusNone, SellerStatusPending, SellerStatusApproved, SellerStatusRejected}

func (v SellerStatus) Valid() bool { return enumValid(v, SellerStatuses) }

// StudentCardStatus は学生証の審査状態。
type StudentCardStatus string

const (
	StudentCardStatusPending  StudentCardStatus = "pending"
	StudentCardStatusApproved StudentCardStatus = "approved"
	StudentCardStatusRejected StudentCardStatus = "rejected"
)

// StudentCardStatuses は StudentCardStatus のすべての値。
var StudentCardStatuses = []StudentCardStatus{StudentCardStatusPending, StudentCardStatusApproved, StudentCardStatusRejected}

func (v StudentCardStatus) Valid() bool { return enumValid(v, StudentCardStatuses) }

// ItemStatus は商品のステータス。
type ItemStatus string

const (
	ItemStatusDraft     ItemStatus = "draft"     // 下書き
	ItemStatusPublished ItemStatus = "published" // 公開中
	ItemStatusRequested ItemStatus = "requested" // 購入希望あり
	ItemStatusInDeal    ItemStatus = "in_deal"   // 取引中
	ItemStatusCompleted ItemStatus = "completed" // 取引完了
	ItemStatusWithdrawn ItemStatus = "withdrawn" // 取り下げ
	ItemStatusSuspended ItemStatus = "suspended" // 停止（管理者）
)

// ItemStatuses は ItemStatus のすべての値。
var ItemStatuses = []ItemStatus{
	ItemStatusDraft, ItemStatusPublished, ItemStatusRequested, ItemStatusInDeal,
	ItemStatusCompleted, ItemStatusWithdrawn, ItemStatusSuspended,
}

func (v ItemStatus) Valid() bool { return enumValid(v, ItemStatuses) }

// TransactionStatus は取引のステータス。購入希望の段階も Transaction（requested）として扱う。
type TransactionStatus string

const (
	TransactionStatusRequested TransactionStatus = "requested" // 購入希望
	TransactionStatusRejected  TransactionStatus = "rejected"  // 却下
	TransactionStatusExpired   TransactionStatus = "expired"   // 期限切れ
	TransactionStatusInDeal    TransactionStatus = "in_deal"   // 取引中
	TransactionStatusCancelled TransactionStatus = "cancelled" // キャンセル
	TransactionStatusCompleted TransactionStatus = "completed" // 完了
)

// TransactionStatuses は TransactionStatus のすべての値。
var TransactionStatuses = []TransactionStatus{
	TransactionStatusRequested, TransactionStatusRejected, TransactionStatusExpired,
	TransactionStatusInDeal, TransactionStatusCancelled, TransactionStatusCompleted,
}

func (v TransactionStatus) Valid() bool { return enumValid(v, TransactionStatuses) }

// HandoverMethod は受け渡しの完了方法。
type HandoverMethod string

const (
	HandoverMethodQR   HandoverMethod = "qr"
	HandoverMethodLink HandoverMethod = "link"
	HandoverMethodCode HandoverMethod = "code"
	HandoverMethodAuto HandoverMethod = "auto"
)

// HandoverMethods は HandoverMethod のすべての値。
var HandoverMethods = []HandoverMethod{HandoverMethodQR, HandoverMethodLink, HandoverMethodCode, HandoverMethodAuto}

func (v HandoverMethod) Valid() bool { return enumValid(v, HandoverMethods) }

// HandoverTokenKind は受け渡しトークンの種類。
type HandoverTokenKind string

const (
	HandoverTokenKindQR   HandoverTokenKind = "qr"
	HandoverTokenKindLink HandoverTokenKind = "link"
	HandoverTokenKindCode HandoverTokenKind = "code"
)

// HandoverTokenKinds は HandoverTokenKind のすべての値。
var HandoverTokenKinds = []HandoverTokenKind{HandoverTokenKindQR, HandoverTokenKindLink, HandoverTokenKindCode}

func (v HandoverTokenKind) Valid() bool { return enumValid(v, HandoverTokenKinds) }

// PaymentKind は決済の種別。
type PaymentKind string

const (
	PaymentKindAuthorize PaymentKind = "authorize" // 枠の確保
	PaymentKindCapture   PaymentKind = "capture"   // 預かり
	PaymentKindRelease   PaymentKind = "release"   // 枠の解放
	PaymentKindRefund    PaymentKind = "refund"    // 返金
	PaymentKindSettle    PaymentKind = "settle"    // 売上の確定
	PaymentKindPayout    PaymentKind = "payout"    // 出金
)

// PaymentKinds は PaymentKind のすべての値。
var PaymentKinds = []PaymentKind{
	PaymentKindAuthorize, PaymentKindCapture, PaymentKindRelease,
	PaymentKindRefund, PaymentKindSettle, PaymentKindPayout,
}

func (v PaymentKind) Valid() bool { return enumValid(v, PaymentKinds) }

// PaymentMethod は支払い方法。
type PaymentMethod string

const (
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodPayPay PaymentMethod = "paypay"
)

// PaymentMethods は PaymentMethod のすべての値。
var PaymentMethods = []PaymentMethod{PaymentMethodCard, PaymentMethodPayPay}

func (v PaymentMethod) Valid() bool { return enumValid(v, PaymentMethods) }

// ItemCategory は商品のカテゴリ（ITEM）。
type ItemCategory string

const (
	ItemCategoryTextbook           ItemCategory = "textbook"            // 教科書・参考書
	ItemCategoryFurnitureAppliance ItemCategory = "furniture_appliance" // 家具・家電
	ItemCategoryGameOther          ItemCategory = "game_other"          // ゲーム・その他
)

// ItemCategories は ItemCategory のすべての値。
var ItemCategories = []ItemCategory{ItemCategoryTextbook, ItemCategoryFurnitureAppliance, ItemCategoryGameOther}

func (v ItemCategory) Valid() bool { return enumValid(v, ItemCategories) }

// ItemCondition は商品の状態（ITEM）。ステータスと紛らわしいので condition と呼ぶ。
type ItemCondition string

const (
	ItemConditionNewLike ItemCondition = "new_like" // 新品同様
	ItemConditionGood    ItemCondition = "good"     // 目立った傷なし
	ItemConditionFair    ItemCondition = "fair"     // やや傷あり
)

// ItemConditions は ItemCondition のすべての値。
var ItemConditions = []ItemCondition{ItemConditionNewLike, ItemConditionGood, ItemConditionFair}

func (v ItemCondition) Valid() bool { return enumValid(v, ItemConditions) }

// Campus はキャンパス。
type Campus string

const (
	CampusHonjo     Campus = "honjo"     // 本庄
	CampusNabeshima Campus = "nabeshima" // 鍋島
)

// Campuses は Campus のすべての値。
var Campuses = []Campus{CampusHonjo, CampusNabeshima}

func (v Campus) Valid() bool { return enumValid(v, Campuses) }

// VerificationPurpose は認証コードの用途。
type VerificationPurpose string

const (
	VerificationPurposeSignup        VerificationPurpose = "signup"
	VerificationPurposePasswordReset VerificationPurpose = "password_reset"
	VerificationPurposeLoginRecheck  VerificationPurpose = "login_recheck"
)

// VerificationPurposes は VerificationPurpose のすべての値。
var VerificationPurposes = []VerificationPurpose{
	VerificationPurposeSignup, VerificationPurposePasswordReset, VerificationPurposeLoginRecheck,
}

func (v VerificationPurpose) Valid() bool { return enumValid(v, VerificationPurposes) }

// PasswordSetupPurpose はパスワード設定用の一時トークンの用途。
type PasswordSetupPurpose string

const (
	PasswordSetupPurposeSignup        PasswordSetupPurpose = "signup"         // 新規登録
	PasswordSetupPurposePasswordReset PasswordSetupPurpose = "password_reset" // パスワードの再設定
)

// PasswordSetupPurposes は PasswordSetupPurpose のすべての値。
var PasswordSetupPurposes = []PasswordSetupPurpose{PasswordSetupPurposeSignup, PasswordSetupPurposePasswordReset}

func (v PasswordSetupPurpose) Valid() bool { return enumValid(v, PasswordSetupPurposes) }

// LoginRecheckReason はログイン時に再確認が必要な理由（AUTHN-3, ADR 0009）。両方に当てはまるときは code_expired にする。
type LoginRecheckReason string

const (
	LoginRecheckReasonCodeExpired LoginRecheckReason = "code_expired" // 最後にコードで確認してから30日以上たった
	LoginRecheckReasonNewDevice   LoginRecheckReason = "new_device"   // 初めて使う端末
)

// LoginRecheckReasons は LoginRecheckReason のすべての値。
var LoginRecheckReasons = []LoginRecheckReason{LoginRecheckReasonCodeExpired, LoginRecheckReasonNewDevice}

func (v LoginRecheckReason) Valid() bool { return enumValid(v, LoginRecheckReasons) }
