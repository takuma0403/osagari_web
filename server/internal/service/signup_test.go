package service_test

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/apperr"
	"github.com/takuma0403/osagari_web/server/internal/auth"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/mail"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/service"
	"github.com/takuma0403/osagari_web/server/internal/testutil/testdb"
)

var t0 = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

const (
	email    = "26238001@edu.cc.saga-u.ac.jp"
	password = "correct horse battery"
)

type fixture struct {
	svc    *service.SignupService
	gdb    *gorm.DB
	clock  *clock.Fake
	mailer *mail.Recorder
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	gdb := testdb.New(t)
	f := &fixture{gdb: gdb, clock: clock.NewFake(t0), mailer: &mail.Recorder{}}
	f.svc = service.NewSignupService(service.SignupDeps{
		Tx:       repository.NewTransactor(gdb),
		Users:    repository.NewUserRepository(gdb),
		Codes:    repository.NewVerificationCodeRepository(gdb),
		Tokens:   repository.NewPasswordSetupTokenRepository(gdb),
		Sessions: repository.NewSessionRepository(gdb),
		Devices:  repository.NewTrustedDeviceRepository(gdb),
		Mailer:   f.mailer,
		Clock:    f.clock,
	})
	return f
}

var codeInMail = regexp.MustCompile(`認証コード：([0-9]{6})`)

// lastCode は最後に送ったメールからコードを取り出す。コードが書かれていなければ空。
func (f *fixture) lastCode(t *testing.T) string {
	t.Helper()
	m, ok := f.mailer.Last()
	if !ok {
		t.Fatal("メールが送られていない")
	}
	if g := codeInMail.FindStringSubmatch(m.Body); g != nil {
		return g[1]
	}
	return ""
}

// signup は登録を最後まで行い、作られたユーザーを返す。
func (f *fixture) signup(t *testing.T, email, deviceID string) service.CompletedSignup {
	t.Helper()
	ctx := context.Background()
	if err := f.svc.SendCode(ctx, email); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.VerifyCode(ctx, email, f.lastCode(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.svc.Complete(ctx, v.SetupToken, password, deviceID)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *fixture) count(t *testing.T, table string) int64 {
	t.Helper()
	var n int64
	if err := f.gdb.Table(table).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func wantCode(t *testing.T, err error, code apperr.Code) {
	t.Helper()
	if got := apperr.CodeOf(err); got != code {
		t.Fatalf("エラーコード=%q want=%q（err=%v）", got, code, err)
	}
}

func wrongCode(code string) string {
	if code == "000000" {
		return "000001"
	}
	return "000000"
}

func TestEstimate(t *testing.T) {
	f := newFixture(t)
	est, err := f.svc.Estimate(" 26211001@EDU.cc.saga-u.ac.jp ")
	if err != nil {
		t.Fatal(err)
	}
	if est.AdmissionYear != 2026 || est.Department == nil || est.Department.Name != "医学部 医学科" ||
		est.ExpiresOn.Format("2006-01-02") != "2038-03-31" {
		t.Errorf("est=%+v", est)
	}
	_, err = f.svc.Estimate("teacher@edu.cc.saga-u.ac.jp")
	wantCode(t, err, service.CodeInvalidEmail)
	// 有効期限（2026年3月31日）を過ぎた学籍番号（AUTH-3, ADR 0047）
	_, err = f.svc.Estimate(expiredEmail)
	wantCode(t, err, service.CodeSignupPeriodEnded)
}

// expiredEmail は有効期限（2026年3月31日）を過ぎた学籍番号のアドレス。t0（2026年10月）では登録できない。
const expiredEmail = "18238001@edu.cc.saga-u.ac.jp"

func TestSignupPeriodEnded(t *testing.T) {
	ctx := context.Background()

	t.Run("有効期限を過ぎた学籍番号にはコードを送らない（AUTH-3, ADR 0047）", func(t *testing.T) {
		f := newFixture(t)
		wantCode(t, f.svc.SendCode(ctx, expiredEmail), service.CodeSignupPeriodEnded)
		_, err := f.svc.VerifyCode(ctx, expiredEmail, "123456")
		wantCode(t, err, service.CodeSignupPeriodEnded)
		if n := len(f.mailer.Messages()); n != 0 {
			t.Errorf("%d通送られた", n)
		}
		if n := f.count(t, "verification_codes"); n != 0 {
			t.Errorf("コードが%d件保存された", n)
		}
	})

	t.Run("登録済みでも未登録と同じ応答にする（AUTHN-2）", func(t *testing.T) {
		f := newFixture(t)
		// 有効期限の当日に登録し、翌日にもう一度コードを求める
		f.clock.Set(time.Date(2026, time.March, 31, 14, 0, 0, 0, time.UTC)) // JST 23:00
		f.signup(t, expiredEmail, "")
		f.clock.Advance(2 * time.Hour)
		wantCode(t, f.svc.SendCode(ctx, expiredEmail), service.CodeSignupPeriodEnded)
		wantCode(t, f.svc.SendCode(ctx, "18238002@edu.cc.saga-u.ac.jp"), service.CodeSignupPeriodEnded)
	})

	t.Run("コードの確認の後に有効期限を過ぎたら作らず、トークンも消費しない", func(t *testing.T) {
		f := newFixture(t)
		f.clock.Set(time.Date(2026, time.March, 31, 14, 55, 0, 0, time.UTC)) // JST 23:55
		if err := f.svc.SendCode(ctx, expiredEmail); err != nil {
			t.Fatal(err)
		}
		v, err := f.svc.VerifyCode(ctx, expiredEmail, f.lastCode(t))
		if err != nil {
			t.Fatal(err)
		}
		f.clock.Advance(10 * time.Minute) // JST 4月1日 0:05
		_, err = f.svc.Complete(ctx, v.SetupToken, password, "")
		wantCode(t, err, service.CodeSignupPeriodEnded)
		if n := f.count(t, "users"); n != 0 {
			t.Errorf("ユーザーが%d件作られた", n)
		}
		var used int64
		if err := f.gdb.Table("password_setup_tokens").Where("consumed_at IS NOT NULL").Count(&used).Error; err != nil {
			t.Fatal(err)
		}
		if used != 0 {
			t.Errorf("トークンが消費された")
		}
	})
}

func TestSendCode(t *testing.T) {
	ctx := context.Background()

	t.Run("形式が違うアドレスには送らない", func(t *testing.T) {
		f := newFixture(t)
		for _, e := range []string{"", "2623800@edu.cc.saga-u.ac.jp", "26238001@gmail.com", "yamada@edu.cc.saga-u.ac.jp"} {
			wantCode(t, f.svc.SendCode(ctx, e), service.CodeInvalidEmail)
		}
		if n := len(f.mailer.Messages()); n != 0 {
			t.Errorf("%d通送られた", n)
		}
	})

	t.Run("未登録と登録済みで応答と処理が同じで、本文だけが違う（AUTHN-2）", func(t *testing.T) {
		f := newFixture(t)
		f.signup(t, email, "")
		f.clock.Advance(auth.CodeSendInterval) // 登録のときのコード送信から間隔をあける（AUTHN-6）
		before := f.count(t, "verification_codes")
		sentBefore := len(f.mailer.Messages())

		// 登録済みのアドレス
		errRegistered := f.svc.SendCode(ctx, email)
		registeredMail, _ := f.mailer.Last()
		// 未登録のアドレス
		const other = "26238002@edu.cc.saga-u.ac.jp"
		errNew := f.svc.SendCode(ctx, other)
		newMail, _ := f.mailer.Last()

		if errRegistered != nil || errNew != nil {
			t.Fatalf("応答が違う: 登録済み=%v 未登録=%v", errRegistered, errNew)
		}
		if got := len(f.mailer.Messages()) - sentBefore; got != 2 {
			t.Errorf("送ったメール=%d通 want=2（どちらも1通ずつ）", got)
		}
		if got := f.count(t, "verification_codes") - before; got != 2 {
			t.Errorf("保存したコード=%d件 want=2（登録済みでもコードを保存する）", got)
		}
		if registeredMail.To != email || newMail.To != other || registeredMail.Subject != newMail.Subject {
			t.Errorf("宛先・件名: %+v / %+v", registeredMail, newMail)
		}
		if codeInMail.MatchString(registeredMail.Body) || !strings.Contains(registeredMail.Body, "登録済み") {
			t.Errorf("登録済みの案内になっていない: %s", registeredMail.Body)
		}
		if !codeInMail.MatchString(newMail.Body) {
			t.Errorf("コードが書かれていない: %s", newMail.Body)
		}
	})

	t.Run("登録済みのアドレスに保存したコードでは登録できない", func(t *testing.T) {
		f := newFixture(t)
		f.signup(t, email, "")
		f.clock.Advance(auth.CodeSendInterval)
		if err := f.svc.SendCode(ctx, email); err != nil {
			t.Fatal(err)
		}
		// 誰にも届いていないコードは当てられず、入力ミスとして数えられる
		_, err := f.svc.VerifyCode(ctx, email, "123456")
		wantCode(t, err, service.CodeInvalidVerificationCode)
	})

	t.Run("メールを送れなければ 500", func(t *testing.T) {
		f := newFixture(t)
		f.mailer.Err = context.DeadlineExceeded
		wantCode(t, f.svc.SendCode(ctx, email), apperr.CodeInternal)
	})
}

func TestVerifyCode(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		// run はコードを送った後の操作。最後の VerifyCode のエラーを返す
		run  func(t *testing.T, f *fixture, code string) error
		want apperr.Code // 空なら成功
	}{
		{"一致すれば一時トークンを発行する", func(t *testing.T, f *fixture, code string) error {
			v, err := f.svc.VerifyCode(ctx, email, code)
			if err == nil && (v.SetupToken == "" || v.Estimate.StudentNumber != "26238001") {
				t.Errorf("v=%+v", v)
			}
			return err
		}, ""},
		{"大文字・空白を含むアドレスでも同じコードとして扱う", func(t *testing.T, f *fixture, code string) error {
			_, err := f.svc.VerifyCode(ctx, " 26238001@EDU.CC.SAGA-U.AC.JP", code)
			return err
		}, ""},
		{"違うコードは残り回数を返す", func(t *testing.T, f *fixture, code string) error {
			_, err := f.svc.VerifyCode(ctx, email, wrongCode(code))
			if err != nil && !strings.Contains(err.Error(), "あと4回") {
				t.Errorf("残り回数がない: %v", err)
			}
			return err
		}, service.CodeInvalidVerificationCode},
		{"6桁の数字でなければ回数を数えずに 400", func(t *testing.T, f *fixture, code string) error {
			_, err := f.svc.VerifyCode(ctx, email, "12345a")
			return err
		}, apperr.CodeBadRequest},
		{"5回間違えたら正しいコードも使えない", func(t *testing.T, f *fixture, code string) error {
			for i := range auth.MaxCodeFailures {
				_, err := f.svc.VerifyCode(ctx, email, wrongCode(code))
				wantCode(t, err, service.CodeInvalidVerificationCode)
				if i == auth.MaxCodeFailures-1 && !strings.Contains(err.Error(), "使えなくなりました") {
					t.Errorf("最後のミスの文言: %v", err)
				}
			}
			_, err := f.svc.VerifyCode(ctx, email, code)
			return err
		}, service.CodeVerificationCodeExpired},
		{"10分たったら使えない", func(t *testing.T, f *fixture, code string) error {
			f.clock.Advance(auth.CodeTTL)
			_, err := f.svc.VerifyCode(ctx, email, code)
			return err
		}, service.CodeVerificationCodeExpired},
		{"9分59秒なら使える", func(t *testing.T, f *fixture, code string) error {
			f.clock.Advance(auth.CodeTTL - time.Second)
			_, err := f.svc.VerifyCode(ctx, email, code)
			return err
		}, ""},
		{"再送したら古いコードは使えない", func(t *testing.T, f *fixture, code string) error {
			f.clock.Advance(auth.CodeSendInterval)
			if err := f.svc.SendCode(ctx, email); err != nil {
				t.Fatal(err)
			}
			if newCode := f.lastCode(t); newCode == code {
				t.Skip("偶然同じコードになった")
			}
			_, err := f.svc.VerifyCode(ctx, email, code)
			return err
		}, service.CodeInvalidVerificationCode},
		{"一度使ったコードは使えない", func(t *testing.T, f *fixture, code string) error {
			if _, err := f.svc.VerifyCode(ctx, email, code); err != nil {
				t.Fatal(err)
			}
			_, err := f.svc.VerifyCode(ctx, email, code)
			return err
		}, service.CodeVerificationCodeExpired},
		{"コードを送っていないアドレス", func(t *testing.T, f *fixture, code string) error {
			_, err := f.svc.VerifyCode(ctx, "26238099@edu.cc.saga-u.ac.jp", code)
			return err
		}, service.CodeVerificationCodeExpired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if err := f.svc.SendCode(ctx, email); err != nil {
				t.Fatal(err)
			}
			err := tc.run(t, f, f.lastCode(t))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("err=%v", err)
				}
				return
			}
			wantCode(t, err, tc.want)
		})
	}

	t.Run("同じコードで同時に確認しても一時トークンは1つだけ", func(t *testing.T) {
		f := newFixture(t)
		if err := f.svc.SendCode(ctx, email); err != nil {
			t.Fatal(err)
		}
		code := f.lastCode(t)
		var ok atomic.Int32
		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				if _, err := f.svc.VerifyCode(ctx, email, code); err == nil {
					ok.Add(1)
				}
			})
		}
		wg.Wait()
		if ok.Load() != 1 || f.count(t, "password_setup_tokens") != 1 {
			t.Errorf("成功=%d 一時トークン=%d", ok.Load(), f.count(t, "password_setup_tokens"))
		}
	})
}

func TestComplete(t *testing.T) {
	ctx := context.Background()

	// verified はコードの確認まで済ませて一時トークンを返す。
	verified := func(t *testing.T, f *fixture, email string) string {
		t.Helper()
		if err := f.svc.SendCode(ctx, email); err != nil {
			t.Fatal(err)
		}
		v, err := f.svc.VerifyCode(ctx, email, f.lastCode(t))
		if err != nil {
			t.Fatal(err)
		}
		return v.SetupToken
	}

	t.Run("アカウントを作り、学籍番号から推定した値で登録してログイン状態にする", func(t *testing.T) {
		f := newFixture(t)
		token := verified(t, f, "26211001@edu.cc.saga-u.ac.jp")
		f.clock.Advance(time.Minute)
		out, err := f.svc.Complete(ctx, token, password, "device-1")
		if err != nil {
			t.Fatal(err)
		}
		u := out.User
		now := t0.Add(time.Minute)
		if u.ID == "" || u.Email != "26211001@edu.cc.saga-u.ac.jp" || u.StudentNumber != "26211001" ||
			u.AdmissionYear != 2026 || u.DepartmentCode != "211" || u.Department == nil || *u.Department != "211" ||
			u.ExpiresOn.Format("2006-01-02") != "2038-03-31" || u.Status != domain.UserStatusActive ||
			u.SellerStatus != domain.SellerStatusNone || !u.CodeVerifiedAt.Equal(now) {
			t.Errorf("user=%+v", u)
		}
		if ok, err := auth.VerifyPassword(u.PasswordHash, password); err != nil || !ok {
			t.Errorf("パスワードのハッシュが一致しない: ok=%v err=%v", ok, err)
		}
		if out.Session.Token == "" || !out.Session.ExpiresAt.Equal(now.Add(auth.SessionTTL)) {
			t.Errorf("session=%+v", out.Session)
		}
		var tokenHash, deviceHash string
		f.gdb.Raw("SELECT token_hash FROM sessions WHERE user_id = ?", u.ID).Scan(&tokenHash)
		f.gdb.Raw("SELECT device_hash FROM trusted_devices WHERE user_id = ?", u.ID).Scan(&deviceHash)
		if tokenHash != auth.HashToken(out.Session.Token) {
			t.Errorf("セッションのハッシュが保存されていない: %q", tokenHash)
		}
		if deviceHash != auth.HashToken("device-1") {
			t.Errorf("端末が登録されていない: %q", deviceHash)
		}
	})

	t.Run("表にない学部コードは不明・4年で登録する", func(t *testing.T) {
		f := newFixture(t)
		out, err := f.svc.Complete(ctx, verified(t, f, "24726001@edu.cc.saga-u.ac.jp"), password, "")
		if err != nil {
			t.Fatal(err)
		}
		if out.User.Department != nil || out.User.DepartmentCode != "726" || out.User.ExpiresOn.Format("2006-01-02") != "2032-03-31" {
			t.Errorf("user=%+v", out.User)
		}
		if f.count(t, "trusted_devices") != 0 {
			t.Error("端末識別子がないのに端末が登録された")
		}
	})

	cases := []struct {
		name string
		run  func(t *testing.T, f *fixture, token string) error
		want apperr.Code
	}{
		{"パスワードが9文字", func(t *testing.T, f *fixture, token string) error {
			_, err := f.svc.Complete(ctx, token, "123456789", "")
			// トークンは消費されず、入力し直せる
			if _, err2 := f.svc.Complete(ctx, token, password, ""); err2 != nil {
				t.Errorf("入力し直せない: %v", err2)
			}
			return err
		}, service.CodeInvalidPasswordLength},
		{"パスワードが129文字", func(t *testing.T, f *fixture, token string) error {
			_, err := f.svc.Complete(ctx, token, strings.Repeat("a", 129), "")
			return err
		}, service.CodeInvalidPasswordLength},
		{"15分たったトークン", func(t *testing.T, f *fixture, token string) error {
			f.clock.Advance(auth.SetupTokenTTL)
			_, err := f.svc.Complete(ctx, token, password, "")
			return err
		}, service.CodeInvalidSetupToken},
		{"使用済みのトークン", func(t *testing.T, f *fixture, token string) error {
			if _, err := f.svc.Complete(ctx, token, password, ""); err != nil {
				t.Fatal(err)
			}
			_, err := f.svc.Complete(ctx, token, password, "")
			return err
		}, service.CodeInvalidSetupToken},
		{"存在しないトークン", func(t *testing.T, f *fixture, token string) error {
			_, err := f.svc.Complete(ctx, token+"x", password, "")
			return err
		}, service.CodeInvalidSetupToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			wantCode(t, tc.run(t, f, verified(t, f, email)), tc.want)
		})
	}

	t.Run("登録済みのアドレスは作らず、無効なトークンと同じ応答にしてロールバックする", func(t *testing.T) {
		f := newFixture(t)
		f.signup(t, email, "")
		// 登録済みのアドレスの一時トークン（コードを当てた場合に相当）を直接作る
		token, _ := auth.GenerateToken()
		err := repository.NewPasswordSetupTokenRepository(f.gdb).Create(ctx, &domain.PasswordSetupToken{
			Email: email, Purpose: domain.PasswordSetupPurposeSignup, TokenHash: auth.HashToken(token),
			ExpiresAt: t0.Add(auth.SetupTokenTTL), CreatedAt: t0, UpdatedAt: t0,
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.svc.Complete(ctx, token, password, "")
		wantCode(t, err, service.CodeInvalidSetupToken)
		if n := f.count(t, "users"); n != 1 {
			t.Errorf("users=%d", n)
		}
		if n := f.count(t, "sessions"); n != 1 {
			t.Errorf("セッションが増えた: %d", n)
		}
	})
}
