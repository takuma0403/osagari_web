package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"
)

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name     string
		password string
		ok       bool
	}{
		{"9文字", "abcdefghi", false},
		{"10文字", "abcdefghij", true},
		{"日本語は文字数で数える（10文字）", "あいうえおかきくけこ", true},
		{"日本語の9文字（30バイト近いが短い）", "あいうえおかきくけ", false},
		{"空", "", false},
		{"128文字", strings.Repeat("a", 128), true},
		{"129文字", strings.Repeat("a", 129), false},
		{"日本語の128文字（バイト数ではなく文字数で数える）", strings.Repeat("あ", 128), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password)
			if tc.ok != (err == nil) {
				t.Fatalf("err=%v", err)
			}
			if err != nil && !errors.Is(err, ErrPasswordLength) {
				t.Fatalf("ErrPasswordLength ではない: %v", err)
			}
		})
	}
}

func TestHashPasswordとVerifyPassword(t *testing.T) {
	const password = "correct horse battery"
	h1, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h1, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("PHC 形式ではない: %q", h1)
	}
	if strings.Contains(h1, password) {
		t.Error("ハッシュに平文が含まれる")
	}
	h2, _ := HashPassword(password)
	if h1 == h2 {
		t.Error("ソルトが毎回変わっていない")
	}

	if ok, err := VerifyPassword(h1, password); err != nil || !ok {
		t.Errorf("正しいパスワードで一致しない: ok=%v err=%v", ok, err)
	}
	if ok, err := VerifyPassword(h1, password+"x"); err != nil || ok {
		t.Errorf("違うパスワードで一致する: ok=%v err=%v", ok, err)
	}
	if _, err := HashPassword("short"); !errors.Is(err, ErrPasswordLength) {
		t.Errorf("短いパスワードをハッシュ化できてしまう: %v", err)
	}
}

func TestVerifyPasswordは保存されたパラメータで照合する(t *testing.T) {
	// パラメータを変えても既存のハッシュを照合できること（m=8MiB, t=1 で作られたハッシュ）
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("old-params-password"), salt, 1, 8192, 1, 32)
	b64 := base64.RawStdEncoding
	old := "$argon2id$v=19$m=8192,t=1,p=1$" + b64.EncodeToString(salt) + "$" + b64.EncodeToString(key)
	if ok, err := VerifyPassword(old, "old-params-password"); err != nil || !ok {
		t.Fatalf("別のパラメータのハッシュを照合できない: ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyPassword(old, "wrong-password!"); ok {
		t.Fatal("違うパスワードで一致する")
	}
}

func TestVerifyPasswordは不正な形式をエラーにする(t *testing.T) {
	for _, encoded := range []string{
		"",
		"plain",
		"$bcrypt$v=19$m=19456,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$m=0,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$a2V5",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$",
	} {
		if ok, err := VerifyPassword(encoded, "password123"); ok || err == nil {
			t.Errorf("%q: ok=%v err=%v", encoded, ok, err)
		}
	}
}

func TestGenerateCode(t *testing.T) {
	sixDigits := regexp.MustCompile(`^[0-9]{6}$`)
	seen := map[string]bool{}
	for range 200 {
		c, err := GenerateCode()
		if err != nil {
			t.Fatal(err)
		}
		if !sixDigits.MatchString(c) {
			t.Fatalf("6桁の数字ではない: %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Errorf("重複が多すぎる（乱数になっていない）: %d種類", len(seen))
	}
}

func TestHashCodeとVerifyCode(t *testing.T) {
	h, err := HashCode("012345")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h, "012345") {
		t.Error("ハッシュに平文が含まれる")
	}
	if ok, err := VerifyCode(h, "012345"); err != nil || !ok {
		t.Errorf("正しいコードで一致しない: ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyCode(h, "012346"); ok {
		t.Error("違うコードで一致する")
	}
}

func TestGenerateTokenとHashToken(t *testing.T) {
	a, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateToken()
	if a == b {
		t.Error("同じトークンが作られた")
	}
	if len(a) != 43 || strings.ContainsAny(a, "+/=") {
		t.Errorf("256ビットの URL 用 base64 ではない: %q", a)
	}

	h := HashToken(a)
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(h) {
		t.Errorf("SHA-256 の16進ではない: %q", h)
	}
	if HashToken(a) != h {
		t.Error("同じトークンのハッシュが変わる（検索に使えない）")
	}
	if !EqualHash(h, HashToken(a)) || EqualHash(h, HashToken(b)) {
		t.Error("EqualHash の結果が違う")
	}
}

func TestNewSessionCookie(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	c := NewSessionCookie("token", now, now.Add(SessionTTL))

	if c.Name != "__session" || c.Value != "token" || c.Path != "/" {
		t.Errorf("Cookie=%+v", c)
	}
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("属性が AUTHN-5 と違う: HttpOnly=%v Secure=%v SameSite=%v", c.HttpOnly, c.Secure, c.SameSite)
	}
	if c.MaxAge != 30*24*60*60 || !c.Expires.Equal(now.Add(SessionTTL)) {
		t.Errorf("有効期間が30日ではない: MaxAge=%d Expires=%v", c.MaxAge, c.Expires)
	}

	header := c.String()
	for _, want := range []string{"__session=token", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie に %q がない: %s", want, header)
		}
	}
}

func Test期限切れのセッションCookieは削除になる(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	for _, c := range []*http.Cookie{NewSessionCookie("token", now, now), ClearSessionCookie()} {
		if c.Name != "__session" || c.Value != "" || c.MaxAge >= 0 {
			t.Errorf("削除の Cookie になっていない: %+v", c)
		}
		if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
			t.Errorf("削除の Cookie の属性が違う: %+v", c)
		}
	}
}

// ダミーの照合は、本物と同じ形式のハッシュに対して照合する（所要時間を揃えるため。ADR 0009）。
func TestVerifyDummyPassword(t *testing.T) {
	h, err := dummyPasswordHash()
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword(h, "dummy password for constant time"); err != nil || !ok {
		t.Fatalf("ダミーのハッシュが照合できない: ok=%v err=%v", ok, err)
	}
	VerifyDummyPassword("anything at all")
}
