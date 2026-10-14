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
