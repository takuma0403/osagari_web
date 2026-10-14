package service

import (
	"fmt"

	"github.com/takuma0403/osagari_web/server/internal/mail"
)

// 新規登録で送るメール（AUTHN-2）。登録済みのアドレスにはコードの代わりに案内を送り、件名は同じにする。
const signupMailSubject = "【佐賀大お下がりフリマ】新規登録の認証コード"

func signupCodeMail(to, code string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: signupMailSubject,
		Body: fmt.Sprintf(`佐賀大お下がりフリマの新規登録の認証コードです。

認証コード：%s

登録画面にこのコードを入力してください。コードの有効期限は10分です。
心当たりがない場合は、このメールを破棄してください。
`, code),
	}
}

func signupAlreadyRegisteredMail(to string) mail.Message {
	return mail.Message{
		To:      to,
		Subject: signupMailSubject,
		Body: `このメールアドレスは、佐賀大お下がりフリマに登録済みです。

新規登録は必要ありません。ログイン画面から、メールアドレスとパスワードでログインしてください。
パスワードを忘れた場合は、ログイン画面の「パスワードを忘れた場合」から再設定できます。

心当たりがない場合は、このメールを破棄してください。
`,
	}
}
