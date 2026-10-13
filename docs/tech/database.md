# DB（技術）

> 関連: [データモデル（要件）](../product/09-data-model.md) / [db-migration スキル](../../.claude/skills/db-migration/SKILL.md) / 実装規約: [server/CLAUDE.md](../../server/CLAUDE.md)

## DB DB

DBは Turso（SQLite互換）を使います。無料プランでもスリープしないため、夏休みなどの利用が少ない時期にも止まりません。

| 項目 | 無料プランの上限 |
| --- | --- |
| ストレージ | 合計5GB |
| 読み取り | 月5億行 |
| 書き込み | 月1,000万行 |
| ポイントインタイム復元 | 1日分 |

出典：[Turso Pricing](https://turso.tech/pricing.md)、[freetier.co](https://freetier.co/directory/products/turso)

- **DBアクセス**：Go から GORM を使って Turso に接続します。ドライバは go-libsql、GORM の方言は gormlite です。ローカルでは Turso 互換の libsql-server を使います（[ADR 0002](../decisions/0002-dev-environment.md)）。
- **複数文の SQL**：go-libsql は1回の実行に複数の文を渡すと2文目以降を黙って捨てます。マイグレーションは文ごとに分けて実行し、アプリのコードでも1回の実行には1文だけを渡します（[ADR 0002](../decisions/0002-dev-environment.md)）。
- **マイグレーション**：golang-migrate で、SQLファイルとしてバージョン管理します。GORM の自動マイグレーション（AutoMigrate）は本番では使わず、スキーマの変更は必ずマイグレーションファイルで行います。
- **二重処理の防止**：購入希望の承認や受け渡しの完了は、「現在のステータスが想定どおりのときだけ更新する」条件付きの更新で行います。同じ商品を2人に承認したり、同じ取引を2回完了させたりすることを防ぎます。
- **バックアップ**：無料プランの復元は1日分しかないため、定期処理で毎日DBを書き出して Cloud Storage に保存します。
- **比較した候補**：Supabase の無料プランは、1週間アクセスがないとプロジェクトが一時停止するため見送りました。Render の無料Postgresは作成から30日で期限切れになるため、本番には使えません。
