package service

import (
	"fmt"

	"github.com/takuma0403/osagari_web/server/internal/mail"
)

// パスワードの再設定で送るメール（AUTHN-4）。未登録・削除済みのアドレスにはコードの代わりに案内を送り、件名は同じにする。
const passwordResetMailSubject = "【佐賀大お下がりフリマ】パスワード再設定の認証コード"

func passwordResetCodeMail(to, code string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: passwordResetMailSubject,
		Body: fmt.Sprintf(`佐賀大お下がりフリマのパスワード再設定の認証コードです。

認証コード：%s

再設定の画面にこのコードを入力してください。コードの有効期限は10分です。
新しいパスワードを設定すると、ほかの端末ではログアウトされます。
再設定を申し込んだ覚えがない場合は、このメールを破棄してください。パスワードは変わりません。
`, code),
	}
}

func passwordResetNotRegisteredMail(to string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: passwordResetMailSubject,
		Body: `パスワードの再設定を受け付けましたが、このメールアドレスは佐賀大お下がりフリマに登録されていません。

はじめて使う場合は、ログイン画面の「新規登録」から登録してください。

心当たりがない場合は、このメールを破棄してください。
`,
	}
}
