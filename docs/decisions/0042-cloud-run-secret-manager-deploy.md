# ADR 0042: Cloud Run・Secret Manager・Turso の本番環境の作り方

- 日付: 2026-11-01
- 状態: 採用
- 関連する章ID: ARCH, COST, DB, API

## 背景

M7 デプロイの最初の issue（#38）で、API を Cloud Run に載せ、本番の Turso と Secret Manager をつなぐ必要があった。リージョンとプロジェクトは [ADR 0041](0041-production-regions-and-domain.md) で決まっている。決めることは、環境の作り方の残し方、秘密情報の渡し方、本番へのマイグレーションの適用方法、無料枠に収める Cloud Run の設定。

## 決定

- 本番環境は `infra/` の gcloud スクリプト（`setup.sh`・`set-secret.sh`・`deploy-server.sh`）と手順書（[infra/README.md](../../infra/README.md)）で作る。スクリプトは何度実行しても同じ結果になるようにする。CI からのデプロイ（#39）も `deploy-server.sh` を使う。
- 秘密情報・個人情報（Turso の接続URL、Gmail のアプリパスワード、管理者の学籍番号）は Secret Manager に置き、Cloud Run の「シークレットを環境変数として参照する」設定で渡す。アプリは環境変数を読むだけにする。管理者の学籍番号は、公開予定のリポジトリに書かないためシークレットにする。
- シークレットはシークレット単位で実行用サービスアカウント `osagari-api@…` に読み取り権限を付け、有効なバージョンは各1つだけにする。
- 本番へのマイグレーションは、API と同じイメージの `/migrate up` を Cloud Run ジョブ `osagari-migrate` で、API の更新より先に実行する。マイグレーションは前のバージョンの API でも動くように書く。
- Cloud Run のサービス `osagari-api` は、未認証の呼び出しを許可（Firebase Hosting の rewrites から呼ぶため）、最小0台・最大2台、1 vCPU・512MiB、CPU はリクエストの処理中だけ割り当てる。
- イメージは Artifact Registry（asia-northeast1）に git のコミットをタグにして置き、最新3件より古いものは自動で消す。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| gcloud スクリプト＋手順書（採用） | 規模に合っていて読みやすい。CI から流用できる | 状態の差分管理はできない |
| Terraform | 宣言的に管理できる | state の置き場所が要り、この規模には重い |
| シークレットを環境変数として参照（採用） | アプリの変更が要らない。ローカルと同じ読み方 | 値を変えたら新しいリビジョンを出す必要がある |
| アプリが Secret Manager の API を読む | 再デプロイなしで値を変えられる | 依存とコードが増え、ローカルとの差が出る |
| マイグレーションを Cloud Run ジョブで実行（採用） | 接続トークンが手元に出ない。無料枠内 | ジョブの定義が1つ増える |
| マイグレーションを手元から実行 | 単純 | 接続トークンを手元の環境変数に置く必要がある |
| API の起動時にマイグレーション | 手順が減る | 複数台が同時に起動したときに競合しうる。起動が遅くなる |

## 影響

- 更新したドキュメント：[tech/00-overview.md](../tech/00-overview.md)（ARCH・COST）、[tech/api.md](../tech/api.md)（API：秘密情報の一覧から実在しない「セッション署名鍵」を外した、Cloud Run の台数・CPU・メモリ）、[tech/database.md](../tech/database.md)（DB：本番のマイグレーション）、ルート CLAUDE.md・server/CLAUDE.md（infra/ の追加）
- 実装への影響：Go のコードの変更はなし。`server/Dockerfile` の prod ステージを API とマイグレーションのジョブで共用する
- 後続：Cloud Storage のバケット・権限（#69）、Cloud Scheduler・予算アラート（#40）、Gmail の差出人とアプリパスワード（#41）、Firebase Hosting と CI からのデプロイ（#39）
