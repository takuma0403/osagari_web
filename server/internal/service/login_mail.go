package service

import (
	"fmt"

	"github.com/takuma0403/osagari_web/server/internal/mail"
)

// ログインの再確認で送るメール（AUTHN-3）。
func loginRecheckCodeMail(to, code string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: "【佐賀大お下がりフリマ】ログインの認証コード",
		Body: fmt.Sprintf(`佐賀大お下がりフリマのログインの確認コードです。

認証コード：%s

ログイン画面にこのコードを入力してください。コードの有効期限は10分です。
ログインした覚えがない場合は、パスワードが他人に知られている可能性があります。パスワードを再設定してください。
`, code),
	}
}
