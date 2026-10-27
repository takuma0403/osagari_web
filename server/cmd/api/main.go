// API サーバーの起動。設定の読み込みと依存の組み立て（DI）だけを行う。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"gorm.io/gorm"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/clock"
	"github.com/takuma0403/osagari_web/server/internal/db"
	"github.com/takuma0403/osagari_web/server/internal/domain"
	"github.com/takuma0403/osagari_web/server/internal/handler"
	"github.com/takuma0403/osagari_web/server/internal/jobs"
	"github.com/takuma0403/osagari_web/server/internal/logging"
	"github.com/takuma0403/osagari_web/server/internal/mail"
	"github.com/takuma0403/osagari_web/server/internal/payment"
	"github.com/takuma0403/osagari_web/server/internal/repository"
	"github.com/takuma0403/osagari_web/server/internal/server"
	"github.com/takuma0403/osagari_web/server/internal/service"
	"github.com/takuma0403/osagari_web/server/internal/storage"
)

// config は環境変数から読む設定。一覧は .env.example。
type config struct {
	Port        string
	DatabaseURL string
	SMTP        mail.SMTPConfig
	Storage     storage.GCSConfig
	// StudentCardBucket は学生証の非公開バケット（STORAGE）
	StudentCardBucket string
	// ItemImageBucket は商品写真のバケット（STORAGE）
	ItemImageBucket string
	// Admins は管理者の学籍番号の許可リスト（ADMIN, ADR 0015）
	Admins domain.AdminList
	// Jobs は内部用エンドポイントの呼び出し元の確認。本番は Cloud Scheduler の OIDC、ローカルは共有シークレット（JOBS-1, ADR 0027）
	Jobs server.JobAuthenticator
}

func main() {
	logger := logging.New(os.Stdout, slog.LevelInfo)
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("起動に失敗しました", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	sqlDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	gdb, err := db.NewGORM(sqlDB)
	if err != nil {
		return err
	}

	clk := clock.System()
	mailer, err := mail.NewSMTP(cfg.SMTP, clk)
	if err != nil {
		return err
	}
	store, err := storage.NewGCS(context.Background(), cfg.Storage)
	if err != nil {
		return err
	}
	defer store.Close()
	h, sessions := wire(cfg, gdb, clk, mailer, store, logger)
	e := server.New(server.Deps{
		Logger:   logger,
		Clock:    clk,
		Handler:  h,
		Sessions: sessions,
		Admins:   cfg.Admins,
		Jobs:     cfg.Jobs,
	})
	return e.Start(":" + cfg.Port)
}

// wire は repository → service → handler の順に依存を組み立てる。機能を足すときはここに追加する。
// セッションの検証（server のミドルウェアが使う）も返す。
func wire(cfg config, gdb *gorm.DB, clk clock.Clock, mailer mail.Mailer, store storage.Storage, logger *slog.Logger) (api.StrictServerInterface, server.Authenticator) {
	tx := repository.NewTransactor(gdb)
	users := repository.NewUserRepository(gdb)
	codes := repository.NewVerificationCodeRepository(gdb)
	sessions := repository.NewSessionRepository(gdb)
	devices := repository.NewTrustedDeviceRepository(gdb)
	tokens := repository.NewPasswordSetupTokenRepository(gdb)
	failures := repository.NewLoginFailureRepository(gdb)
	challenges := repository.NewLoginChallengeRepository(gdb)

	signup := service.NewSignupService(service.SignupDeps{
		Tx:       tx,
		Users:    users,
		Codes:    codes,
		Tokens:   tokens,
		Sessions: sessions,
		Devices:  devices,
		Mailer:   mailer,
		Clock:    clk,
		Logger:   logger,
	})
	login := service.NewLoginService(service.LoginDeps{
		Tx:         tx,
		Users:      users,
		Codes:      codes,
		Challenges: challenges,
		Failures:   failures,
		Sessions:   sessions,
		Devices:    devices,
		Mailer:     mailer,
		Clock:      clk,
		Logger:     logger,
	})
	reset := service.NewPasswordResetService(service.PasswordResetDeps{
		Tx:       tx,
		Users:    users,
		Codes:    codes,
		Tokens:   tokens,
		Failures: failures,
		Sessions: sessions,
		Devices:  devices,
		Mailer:   mailer,
		Clock:    clk,
		Logger:   logger,
	})
	session := service.NewSessionService(service.SessionDeps{
		Sessions: sessions,
		Users:    users,
		Clock:    clk,
	})
	cards := repository.NewStudentCardRepository(gdb)
	studentCard := service.NewStudentCardService(service.StudentCardDeps{
		Tx:      tx,
		Users:   users,
		Cards:   cards,
		Storage: store,
		Bucket:  cfg.StudentCardBucket,
		Clock:   clk,
	})
	adminStudentCard := service.NewAdminStudentCardService(service.AdminStudentCardDeps{
		Tx:      tx,
		Users:   users,
		Cards:   cards,
		Views:   repository.NewStudentCardViewRepository(gdb),
		Storage: store,
		Bucket:  cfg.StudentCardBucket,
		Admins:  cfg.Admins,
		Clock:   clk,
	})
	items := repository.NewItemRepository(gdb)
	transactions := repository.NewTransactionRepository(gdb)
	handoverTokens := repository.NewHandoverTokenRepository(gdb)
	settings := repository.NewSettingRepository(gdb)
	// フェーズ1の決済はモック。payments に記録するだけで外部の決済APIは呼ばない（JOBS-2）
	paymentRecords := repository.NewPaymentRepository(gdb)
	payments := payment.NewMock(paymentRecords, clk)
	item := service.NewItemService(service.ItemDeps{
		Tx:           tx,
		Items:        items,
		Users:        users,
		Transactions: transactions,
		Payments:     payments,
		Storage:      store,
		Bucket:       cfg.ItemImageBucket,
		Clock:        clk,
	})
	deal := service.NewDealService(service.DealDeps{
		Tx:             tx,
		Items:          items,
		Users:          users,
		Transactions:   transactions,
		HandoverTokens: handoverTokens,
		Settings:       settings,
		Payments:       payments,
		Storage:        store,
		Bucket:         cfg.ItemImageBucket,
		Clock:          clk,
	})
	sales := service.NewSalesService(service.SalesDeps{
		Tx:           tx,
		Items:        items,
		Transactions: transactions,
		Payouts:      paymentRecords,
		Payments:     payments,
		Storage:      store,
		Bucket:       cfg.ItemImageBucket,
		Clock:        clk,
	})
	cleanup := service.NewAuthCleanupService(service.AuthCleanupDeps{
		Failures:    failures,
		Codes:       codes,
		Challenges:  challenges,
		SetupTokens: tokens,
		Clock:       clk,
	})
	expiry := service.NewAccountExpiryService(service.AccountExpiryDeps{
		Tx:           tx,
		Users:        users,
		StudentCards: cards,
		Items:        items,
		Transactions: transactions,
		Sessions:     sessions,
		Payments:     payments,
		Storage:      store,
		Bucket:       cfg.StudentCardBucket,
		Clock:        clk,
	})
	return handler.New(handler.Deps{
		Signup:        signup,
		Login:         login,
		Session:       session,
		PasswordReset: reset,
		StudentCard:   studentCard,
		Item:          item,
		Deal:          deal,
		Sales:         sales,
		Setting:       service.NewSettingService(service.SettingDeps{Settings: settings}),

		AdminStudentCard: adminStudentCard,
		// 自動完了（DEAL-5）は導入するか未定のため入れない
		ShortJob: jobs.NewShort(logger,
			jobs.Step{Name: "expire_requests", Run: deal.ExpireRequests},
			jobs.Step{Name: "delete_stale_login_failures", Run: cleanup.DeleteStaleLoginFailures},
			jobs.Step{Name: "delete_expired_codes", Run: cleanup.DeleteExpiredCodes},
			jobs.Step{Name: "delete_expired_handover_tokens", Run: deal.DeleteExpiredHandoverTokens},
		),
		// DB のバックアップは日次ジョブに入れず別に扱う（ADR 0033）
		DailyJob: jobs.NewDaily(logger,
			jobs.Step{Name: "expire_accounts", Run: expiry.ExpireAccounts},
			jobs.Step{Name: "delete_student_card_images", Run: expiry.DeleteStudentCardImages},
			jobs.Step{Name: "delete_unregistered_student_card_images", Run: expiry.DeleteUnregisteredStudentCardImages},
		),
	}), session
}

func loadConfig() (config, error) {
	cfg := config{
		Port:        envOr("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		SMTP: mail.SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     envOr("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
	}
	if cfg.DatabaseURL == "" {
		return config{}, fmt.Errorf("環境変数 DATABASE_URL が設定されていません")
	}
	cfg.StudentCardBucket = os.Getenv("STORAGE_BUCKET_STUDENT_CARDS")
	if cfg.StudentCardBucket == "" {
		return config{}, fmt.Errorf("環境変数 STORAGE_BUCKET_STUDENT_CARDS が設定されていません")
	}
	cfg.ItemImageBucket = os.Getenv("STORAGE_BUCKET_ITEM_IMAGES")
	if cfg.ItemImageBucket == "" {
		return config{}, fmt.Errorf("環境変数 STORAGE_BUCKET_ITEM_IMAGES が設定されていません")
	}
	// エミュレータ（fake-gcs-server）を使うときは、署名付きURLにブラウザから届くホストを入れる（STORAGE）
	if host := os.Getenv("STORAGE_EMULATOR_HOST"); host != "" {
		cfg.Storage.EmulatorPublicHost = envOr("STORAGE_PUBLIC_HOST", host)
	}
	// 管理者の学籍番号。不正な値なら起動しない（値はログに出さない。ADR 0015）
	admins, err := domain.ParseAdminList(os.Getenv("ADMIN_STUDENT_NUMBERS"))
	if err != nil {
		return config{}, fmt.Errorf("環境変数 ADMIN_STUDENT_NUMBERS: %w", err)
	}
	cfg.Admins = admins
	// 本番（Gmail SMTP）は STARTTLS を必須にする。無効にするのはローカルの Mailpit だけ（AUTHN-7）
	startTLS, err := strconv.ParseBool(envOr("SMTP_STARTTLS", "true"))
	if err != nil {
		return config{}, fmt.Errorf("環境変数 SMTP_STARTTLS が true / false ではありません")
	}
	cfg.SMTP.StartTLS = startTLS
	jobAuth, err := loadJobAuth()
	if err != nil {
		return config{}, err
	}
	cfg.Jobs = jobAuth
	return cfg, nil
}

// loadJobAuth は内部用エンドポイントの呼び出し元の確認方法を選ぶ（JOBS-1, ADR 0027）。
// 本番は JOBS_OIDC_AUDIENCE と JOBS_SERVICE_ACCOUNT_EMAIL、ローカルは JOBS_LOCAL_SECRET のどちらか一方だけを設定する。
func loadJobAuth() (server.JobAuthenticator, error) {
	audience := os.Getenv("JOBS_OIDC_AUDIENCE")
	account := os.Getenv("JOBS_SERVICE_ACCOUNT_EMAIL")
	secret := os.Getenv("JOBS_LOCAL_SECRET")
	oidc := audience != "" || account != ""
	switch {
	case oidc && secret != "":
		return nil, fmt.Errorf("環境変数 JOBS_LOCAL_SECRET は JOBS_OIDC_AUDIENCE・JOBS_SERVICE_ACCOUNT_EMAIL と同時に設定できません")
	case secret != "":
		return jobs.SecretVerifier{Secret: secret}, nil
	case audience != "" && account != "":
		return jobs.OIDCVerifier{Audience: audience, ServiceAccount: account}, nil
	default:
		return nil, fmt.Errorf("環境変数 JOBS_OIDC_AUDIENCE と JOBS_SERVICE_ACCOUNT_EMAIL（ローカルは JOBS_LOCAL_SECRET）が設定されていません")
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
