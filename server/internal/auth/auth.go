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
)
