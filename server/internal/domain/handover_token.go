package domain

import "time"

const (
	// HandoverQRTTL は受け渡しQRとその下の確認コードの有効期限（DEAL-3）
	HandoverQRTTL = 5 * time.Minute
	// HandoverIssuedTTL は QRを読み忘れたときに発行する完了リンク・完了コードの有効期限（DEAL-3）
	HandoverIssuedTTL = 24 * time.Hour
)

// HandoverToken は受け渡しトークン（handover_tokens）。取引IDと使い捨てのトークンだけを持ち、個人情報・金額は持たない（DEAL-3）。
// qr は TokenHash と CodeHash、link は TokenHash、code は CodeHash だけが入る。
type HandoverToken struct {
	ID            string
	TransactionID string
	Kind          HandoverTokenKind
	// TokenHash は QR・完了リンクのトークンの SHA-256
	TokenHash *string
	// CodeHash は6桁のコードの argon2id のハッシュ
	CodeHash   *string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TTL はトークンの種類ごとの有効期限を返す（DEAL-3）。
func (k HandoverTokenKind) TTL() time.Duration {
	if k == HandoverTokenKindQR {
		return HandoverQRTTL
	}
	return HandoverIssuedTTL
}

// HasToken は QR・完了リンクに載せるトークンを持つ種類か。
func (k HandoverTokenKind) HasToken() bool {
	return k == HandoverTokenKindQR || k == HandoverTokenKindLink
}

// HasCode は6桁のコードを持つ種類か。
func (k HandoverTokenKind) HasCode() bool {
	return k == HandoverTokenKindQR || k == HandoverTokenKindCode
}

// HandoverMethodOf は購入者が入力した種類を受け渡しの完了方法にする。QR の下の確認コードを入力したときも code になる（ADR 0024）。
func HandoverMethodOf(input HandoverTokenKind) HandoverMethod {
	return HandoverMethod(input)
}
