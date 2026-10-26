package jobs

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"google.golang.org/api/idtoken"
)

// ErrUnauthorized は内部用エンドポイントの呼び出し元を確かめられなかったことを表す。
var ErrUnauthorized = errors.New("jobs: 呼び出し元を確かめられません")

// bearerToken は Authorization ヘッダーの Bearer トークンを返す。なければ空。
func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return header[len(prefix):]
}

// OIDCVerifier は Cloud Scheduler が付ける Google の OIDC ID トークンを確かめる（JOBS-1, ADR 0027）。
// 署名・発行者・期限・audience を検証し、トークンのメールアドレスが Cloud Scheduler のサービスアカウントであることを確かめる。
type OIDCVerifier struct {
	// Audience は Cloud Scheduler のジョブに設定した audience（呼び出す URL）
	Audience string
	// ServiceAccount は Cloud Scheduler が使うサービスアカウントのメールアドレス
	ServiceAccount string
}

func (v OIDCVerifier) Verify(ctx context.Context, authorization string) error {
	token := bearerToken(authorization)
	if token == "" {
		return ErrUnauthorized
	}
	p, err := idtoken.Validate(ctx, token, v.Audience)
	if err != nil {
		return ErrUnauthorized
	}
	email, _ := p.Claims["email"].(string)
	verified, _ := p.Claims["email_verified"].(bool)
	if !verified || email != v.ServiceAccount {
		return ErrUnauthorized
	}
	return nil
}

// SecretVerifier は Bearer トークンが共有シークレットと一致するかを定数時間で比べる。
// Cloud Scheduler のないローカル環境だけで使う代替の認証（JOBS-1, ADR 0027）。
type SecretVerifier struct {
	Secret string
}

func (v SecretVerifier) Verify(_ context.Context, authorization string) error {
	token := bearerToken(authorization)
	if v.Secret == "" || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(v.Secret)) != 1 {
		return ErrUnauthorized
	}
	return nil
}
