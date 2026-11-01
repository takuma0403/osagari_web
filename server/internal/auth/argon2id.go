package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// パスワードの長さ（文字数。AUTHN-5）。上限は、巨大な入力をそのまま argon2id に渡さないため（ADR 0008）。
const (
	MinPasswordLength = 10
	MaxPasswordLength = 128
)

// ErrPasswordLength はパスワードが MinPasswordLength 文字より短いか、MaxPasswordLength 文字より長いときに返す。
var ErrPasswordLength = errors.New("auth: パスワードの長さが範囲外です")

// errMalformedHash は保存されたハッシュが PHC 形式として読めないときに返す。
var errMalformedHash = errors.New("auth: argon2id のハッシュの形式が不正です")

// argon2id のパラメータ（OWASP Password Storage Cheat Sheet の推奨値：m=19MiB, t=2, p=1）。
// 照合は保存されたハッシュに書かれたパラメータで行うので、後から変えても既存のハッシュは照合できる。
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonSaltLen = 16
	argonKeyLen  = 32
)

// ValidatePassword はパスワードの長さを確かめる。
func ValidatePassword(password string) error {
	if n := utf8.RuneCountInString(password); n < MinPasswordLength || n > MaxPasswordLength {
		return ErrPasswordLength
	}
	return nil
}

// HashPassword はパスワードを argon2id でハッシュ化し、PHC 形式の文字列で返す（AUTHN-5）。長さの検証も行う。
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	return hashArgon2id(password)
}

// VerifyPassword はパスワードが PHC 形式のハッシュと一致するかを定数時間で確かめる。
func VerifyPassword(encoded, password string) (bool, error) {
	return verifyArgon2id(encoded, password)
}

// dummyPasswordHash は VerifyDummyPassword が照合に使うハッシュ。最初に使うときに1回だけ作る。
var dummyPasswordHash = sync.OnceValues(func() (string, error) {
	return hashArgon2id("dummy password for constant time")
})

// VerifyDummyPassword は、照合するユーザーがいないときにも VerifyPassword と同じ計算をする。
// ログインの所要時間から登録の有無を判別させないために使う（AUTHN-2, ADR 0009）。結果は常に捨てる。
func VerifyDummyPassword(password string) {
	if h, err := dummyPasswordHash(); err == nil {
		_, _ = verifyArgon2id(h, password)
	}
}

func hashArgon2id(secret string) (string, error) {
	salt, err := randomBytes(argonSaltLen)
	if err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(secret), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func verifyArgon2id(encoded, secret string) (bool, error) {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=…,t=…,p=…", salt, key
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errMalformedHash
	}
	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil || memory == 0 || iterations == 0 || threads == 0 {
		return false, errMalformedHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errMalformedHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, errMalformedHash
	}
	got := argon2.IDKey([]byte(secret), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
