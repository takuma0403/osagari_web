package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
)

// tokenBytes はトークン（セッションIDなど）の乱数の長さ。
const tokenBytes = 32

// GenerateCode はメールで送る6桁のコード（000000〜999999、一様分布）を crypto/rand で作る（AUTHN-1）。
func GenerateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("auth: 乱数を作れません: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashCode は6桁のコードを argon2id でハッシュ化する。
// 候補が100万通りしかなく SHA-256 では総当たりで戻せるため、ソルト付きの低速なハッシュにする（AUTHN-2）。
func HashCode(code string) (string, error) {
	return hashArgon2id(code)
}

// VerifyCode はコードがハッシュと一致するかを定数時間で確かめる。
func VerifyCode(encoded, code string) (bool, error) {
	return verifyArgon2id(encoded, code)
}

// GenerateToken は推測できないトークン（256ビット、URL で使える base64）を crypto/rand で作る。
// セッションID・パスワード設定用の一時トークンに使う。
func GenerateToken() (string, error) {
	b, err := randomBytes(tokenBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken はトークンや端末識別子を SHA-256 の16進文字列にする。DB にはこの値だけを保存し、検索にも使う。
// GenerateToken の値は十分に長い乱数なので、ソルトなしの SHA-256 で総当たりできない。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// EqualHash は2つのハッシュ文字列を定数時間で比べる。
func EqualHash(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("auth: 乱数を作れません: %w", err)
	}
	return b, nil
}
