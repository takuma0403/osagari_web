# ADR 0045: Firebase Hosting の設定と CI からのデプロイ

- 日付: 2026-11-02
- 状態: 採用
- 関連する章ID: ARCH, COST, FE, API

## 背景

M7 デプロイの #39 で、SPA を Firebase Hosting で配信し、`/api/**` を Cloud Run に転送し、main への反映で自動デプロイする必要があった。ドメインは [ADR 0041](0041-production-regions-and-domain.md) で標準ドメインに決まっている。決めることは、Hosting の設定、GitHub Actions から GCP への認証方法、デプロイのきっかけと順序。

## 決定

- `firebase.json` と `.firebaserc` はリポジトリのルートに置く。配信するのは `app/dist`。
  - `/api/**` は Cloud Run `osagari-api`（asia-northeast1）に転送し、それ以外のパスはすべて `index.html` を返す（[FE](../tech/frontend.md)）。
  - `Cache-Control` は、ハッシュ付きの `/assets/**` を1年・immutable、それ以外（`index.html` と SPA のパス）を `no-cache` にする。デプロイ直後に古い画面が残らないようにするため。
- 本番に出すのは **main への push だけ**（dev → main のマージ）。dev への push では検査（`ci.yml`）だけが動く。手動実行（`workflow_dispatch`）も main からだけ通る。
- GitHub Actions から GCP へは **Workload Identity 連携**で入り、デプロイ用サービスアカウント `osagari-deployer@…` を借りる。サービスアカウントの鍵は作らない。プロバイダの条件で、このリポジトリの `refs/heads/main` の workflow だけを受け入れる。
- workflow（`.github/workflows/deploy.yml`）は手元と同じ `infra/deploy-server.sh`・`infra/deploy-app.sh` を呼ぶ。変更のあった側だけを出し、両方出すときは API → 画面の順にする（新しい画面が古い API を呼ばないように）。
- デプロイ用 SA の権限は、Cloud Run の管理（`--allow-unauthenticated` で IAM も設定するため admin）、Hosting の管理、プロジェクトの参照、API の有効状態の参照をプロジェクトに、イメージの push を Artifact Registry のリポジトリに、実行用 SA でのデプロイを `osagari-api@…` に付ける。
- Firebase のプランは、ここで選んだものではない。Firebase のプランは請求先アカウントに紐づいているかどうかで決まり、紐づいていれば Blaze（従量課金）になる。[ADR 0041](0041-production-regions-and-domain.md) で Firebase を、Cloud Run のためにすでに請求先アカウントに紐づけたプロジェクト `osagari-web-dev` に入れると決めたので、結果として Blaze になる。Spark にするには課金を外す必要があり、外すと Cloud Run が動かなくなる。新しく有料サービスを足すわけではなく、Hosting は無料枠（保存10GB・転送 月10GB）に収まる見込みで、費用は0円のまま。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| Workload Identity 連携（採用） | 鍵が漏れる心配がない。公開予定のリポジトリでも Secrets を置かずに済む。ブランチで絞れる | 初回の設定が少し多い |
| サービスアカウントの鍵を GitHub Secrets に置く | 設定が簡単 | 長期間有効な鍵の管理と入れ替えが要る。漏れたときの影響が大きい |
| main への push で自動デプロイ（採用） | dev で溜めた変更をリリースのときだけ出せる。手順を忘れない | main に入れたら必ず本番に出る |
| dev への push でも出す | 早く反映される | 確認前の変更が本番に出る |
| 手元からだけデプロイ | 仕組みが要らない | どのコミットが本番にあるか分かりにくい |

## 影響

- 更新したドキュメント：[tech/00-overview.md](../tech/00-overview.md)（ARCH・COST）、[tech/frontend.md](../tech/frontend.md)（FE：Firebase のプラン）、[infra/README.md](../../infra/README.md)、ルート CLAUDE.md（コマンド）
- 実装への影響：アプリのコードの変更はなし。`firebase.json`・`.firebaserc`・`infra/deploy-app.sh`・`.github/workflows/deploy.yml` を追加し、`infra/config.sh`・`infra/setup.sh` を変更
