# 未決事項（技術）

> **エージェントへ**：ここに載っている項目に関わる作業では、勝手に決めずに作業を止めてユーザーに確認すること。要件側の未決事項は [product/open-questions.md](../product/open-questions.md)。

## TECH-OPEN 未決事項

- [x] Cloud Storage のリージョン（東京にするか、無料枠のある米国にするか）→ 米国 us-central1（商品写真・学生証とも）（[ADR 0041](../decisions/0041-production-regions-and-domain.md)）
- [ ] 大学のメールでGmailからの送信が迷惑メール扱いされないかの確認
- [x] GORM と golang-migrate が Turso（libSQL）で問題なく動くかの検証 → go-libsql + gormlite で動作。複数文のマイグレーションは文ごとに分けて実行する（[ADR 0002](../decisions/0002-dev-environment.md)）
- [x] Turso のリージョン選択 → 東京 aws-ap-northeast-1（[ADR 0041](../decisions/0041-production-regions-and-domain.md)）
- [ ] 本番の決済サービスの選定（要件定義書の「決済サービスの候補」を参照）
- [x] 独自ドメインを使うか（使わない場合は Firebase Hosting の標準ドメイン）→ 使わない（[ADR 0041](../decisions/0041-production-regions-and-domain.md)）

- [x] 教職員用の画面・API・認証経路の設計 → 認証と身分証の提出だけを `/api/staff/**` と教職員用の画面に分ける（[AUTHN-8](auth.md)、[ADR 0051](../decisions/0051-staff-accounts.md)）

> 要件定義書の「決済サービスの候補」→ [product/06-payment.md](../product/06-payment.md) の PAY-4
