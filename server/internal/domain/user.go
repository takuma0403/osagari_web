package domain

import "time"

// User はユーザー（users）。DATA, AUTH-1〜4。
type User struct {
	ID             string
	Email          string
	StudentNumber  string
	AdmissionYear  int
	DepartmentCode string
	// Department は確定した学部・学科のコード。nil は不明（管理者が後から直す。ADR 0006）
	Department   *string
	ExpiresOn    time.Time
	Status       UserStatus
	SellerStatus SellerStatus
	PasswordHash string
	// CodeVerifiedAt は最後にメールのコードで確認した日時（AUTHN-3）
	CodeVerifiedAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DepartmentName は学部・学科の表示名を返す。不明なら nil。
func (u User) DepartmentName() *string {
	if u.Department == nil {
		return nil
	}
	d, ok := LookupDepartment(*u.Department)
	if !ok {
		return nil
	}
	return &d.Name
}

// jst は有効期限の日付の境目に使う日本時間。distroless にタイムゾーンのデータがないので固定の時差で持つ。
var jst = time.FixedZone("JST", 9*60*60)

// IsExpired は now の時点でアカウントが失効しているかを返す（AUTH-3）。
// 状態が失効か、日本時間の日付が有効期限（その日の終わりまで使える）を過ぎていれば失効とみなす。
// 日次ジョブが状態を expired にする前でも、期限を過ぎたアカウントは使えないようにする。
func (u User) IsExpired(now time.Time) bool {
	if u.Status == UserStatusExpired {
		return true
	}
	ey, em, ed := u.ExpiresOn.Date()
	return JSTDate(now).After(time.Date(ey, em, ed, 0, 0, 0, 0, time.UTC))
}

// IsActive は now の時点でアカウントが使える（有効で、失効していない）かを返す。
func (u User) IsActive(now time.Time) bool {
	return u.Status == UserStatusActive && !u.IsExpired(now)
}

// JSTDate は now の日本時間での日付を、UTC の0時として返す。有効期限（日付）と比べるのに使う。
func JSTDate(now time.Time) time.Time {
	y, m, d := now.In(jst).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Session はログインセッション（sessions）。Cookie の値は持たず、SHA-256 のハッシュだけを持つ（AUTHN-5）。
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// VerificationCode はメールで送る6桁の認証コード（verification_codes）。コードは argon2id のハッシュだけを持つ（AUTHN-2, AUTHN-6）。
type VerificationCode struct {
	ID             string
	Email          string
	Purpose        VerificationPurpose
	CodeHash       string
	FailedAttempts int
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PasswordSetupToken はパスワード設定用の一時トークン（password_setup_tokens）。SHA-256 のハッシュだけを持つ（AUTHN-2, ADR 0007）。
type PasswordSetupToken struct {
	ID         string
	Email      string
	Purpose    PasswordSetupPurpose
	TokenHash  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TrustedDevice は再確認を省略できる登録端末（trusted_devices）。端末識別子の SHA-256 だけを持つ（AUTHN-3）。
type TrustedDevice struct {
	ID         string
	UserID     string
	DeviceHash string
	LastUsedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// LoginChallenge はログインチャレンジ（login_challenges）。パスワードが一致したが再確認が必要なときに発行し、
// コードの確認・再送に使う。トークンの SHA-256 だけを持つ（AUTHN-3, ADR 0009）。
type LoginChallenge struct {
	ID        string
	UserID    string
	TokenHash string
	Reason    LoginRecheckReason
	// DeviceHash はログイン時に送られた端末識別子の SHA-256。ヘッダーがなければ空
	DeviceHash string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
