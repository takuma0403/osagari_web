// Package auth は認証の部品（パスワード・認証コード・トークンのハッシュと照合、セッション Cookie）を提供する（AUTHN）。
// DB への保存や状態の判定は service・repository が行い、このパッケージは純粋な計算だけを持つ。
package auth

import "time"

// 認証の期限と回数（AUTHN-2〜AUTHN-6）。判定は service が clock.Clock の時刻で行う。
const (
	// SessionTTL はセッションの有効期間（AUTHN-3）
	SessionTTL = 30 * 24 * time.Hour
	// CodeTTL はメールで送る6桁コードの有効期限（AUTHN-2）
	CodeTTL = 10 * time.Minute
	// SetupTokenTTL はコードの確認後に発行するパスワード設定用の一時トークンの有効期限（AUTHN-2）
	SetupTokenTTL = 15 * time.Minute
	// LoginChallengeTTL はログインの再確認用のチャレンジの有効期限（AUTHN-3, ADR 0009）
	LoginChallengeTTL = 15 * time.Minute
	// RecheckInterval は、最後にコードで確認してからこの期間がたつとログイン時に再確認を求める（AUTHN-3）
	RecheckInterval = 30 * 24 * time.Hour
	// MaxCodeFailures はコードの入力ミスの上限。超えたらそのコードを無効にする（AUTHN-6）
	MaxCodeFailures = 5
	// CodeSendInterval は同じアドレスへのコード送信の間隔。用途をまとめて数える（AUTHN-6, ADR 0011）
	CodeSendInterval = time.Minute
	// CodeSendWindow の間に同じアドレスへ送れるコードは MaxCodeSends 回まで（AUTHN-6）
	CodeSendWindow = time.Hour
	MaxCodeSends   = 5
	// MailWarnWindow の間に全体で送ったコードのメールが MailWarnThreshold 通に達したら警告のログを出す。
	// Gmail の1日の送信数の上限（およそ500通）に近づいたことを管理者に知らせる（AUTHN-6, AUTHN-7）
	MailWarnWindow    = 24 * time.Hour
	MailWarnThreshold = 400
	// CodeRetention の間は、期限が切れても verification_codes の行を消さない。送信の制限と全体の送信数を行数で数えるため（AUTHN-6, JOBS-1）
	CodeRetention = MailWarnWindow
	// MaxPasswordFailures 回続けてパスワードを間違えたら、LoginLockDuration の間ログインを止める（AUTHN-6）
	MaxPasswordFailures = 10
	LoginLockDuration   = 15 * time.Minute
	// LoginFailureRetention の間入力ミスがなければ記録を消し、回数を0に戻す（止めている間は消さない。AUTHN-6, ADR 0011）
	LoginFailureRetention = 24 * time.Hour
)
