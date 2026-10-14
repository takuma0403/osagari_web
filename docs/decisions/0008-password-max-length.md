# ADR 0008: パスワードの最大長を128文字にする

- 日付: 2026-10-14
- 状態: 採用
- 関連する章ID: AUTHN-5, AUTHN-2, AUTHN-4, API-1

## 背景

AUTHN-5 はパスワードの長さを「10文字以上」とだけ決めており、上限がなかった。新規登録の完了（`POST /auth/signup/complete`）やパスワード再設定はログインせずに呼べる。上限がないと、巨大な文字列がそのまま argon2id（メモリ19MiB・低速）に渡され、サーバーの負荷を上げられる（#9 の仕様照合で指摘）。

## 決定

- パスワードの長さを **10文字以上128文字以下**にする。文字数（rune数）で数える。
- 範囲外は切り詰めずに 400 で返す。エラーコードは短すぎ・長すぎをまとめて `invalid_password_length` にする（[ADR 0007](0007-signup-flow.md) の `password_too_short` を置き換える）。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 上限なし | 利用者の制約がない | 巨大な入力で argon2id の負荷を上げられる |
| 64文字 | NIST SP 800-63B が「少なくとも64文字は許す」とする下限ちょうど | 長いパスフレーズやパスワードマネージャーの値を弾くことがある |
| **128文字（採用）** | 一般的な上限。パスフレーズにも足り、負荷も抑えられる | — |
| 上限で黙って切り詰める | エラーにならない | 利用者が知らないまま別のパスワードになる。OWASP も推奨しない |

## 影響

- 更新したドキュメント：[tech/auth.md](../tech/auth.md)（AUTHN-5）、[tech/api.md](../tech/api.md)（API-1 のエラーコード）、[api/openapi.yaml](../api/openapi.yaml)（`SignupCompletion.password` の `maxLength`）、ルートの CLAUDE.md の「守るべき決まり」
- 実装：`server/internal/auth`（`MaxPasswordLength`・`ErrPasswordLength`）、`server/internal/service/signup.go`。#11 のパスワード再設定も同じ `auth.ValidatePassword` を使う
- リクエスト本文のサイズ上限は未決（必要になったら別に決める）
