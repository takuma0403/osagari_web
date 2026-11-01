# ADR 0043: 本番の Cloud Storage のバケット設定

- 日付: 2026-11-01
- 状態: 採用
- 関連する章ID: STORAGE, ARCH, COST

## 背景

#69 で、本番の Cloud Storage に商品写真と学生証のバケットを作る。リージョンは [ADR 0041](0041-production-regions-and-domain.md) で us-central1 に決まっている。決めることは、公開範囲の守り方、削除したオブジェクトの扱い（Cloud Storage は既定で7日間ソフト削除として残す）、ブラウザからの直接アップロードを許す範囲（CORS）、署名と読み書きの権限。

## 決定

- 2つのバケットとも、バケット単位のアクセス制御と公開アクセス防止を設定する。商品写真も署名付きURLでしか見せないため（STORAGE）。
- 2つのバケットとも、ソフト削除を無効にする。
- CORS は Firebase Hosting の標準ドメイン（`<プロジェクトID>.web.app`・`<プロジェクトID>.firebaseapp.com`）からの PUT だけを許可し、ヘッダーは署名に含める `Content-Type`・`x-goog-if-generation-match`・`x-goog-content-length-range` を通す。画像の表示は `<img>` の GET なので CORS は要らない。
- 実行用サービスアカウント `osagari-api@…` に、バケットごとに `roles/storage.objectAdmin` を付ける（プロジェクト全体には付けない）。署名は鍵ファイルを持たず IAM signBlob で行うため、同じサービスアカウントに自分自身への `roles/iam.serviceAccountTokenCreator` を付ける。
- 設定は `infra/setup.sh` に入れ、何度実行しても同じ結果になるようにする（[ADR 0042](0042-cloud-run-secret-manager-deploy.md)）。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| ソフト削除を無効にする（採用） | 期限が来た学生証の画像が確実に消える。削除後に無料枠の容量を使わない | 誤って削除したオブジェクトを復元できない |
| ソフト削除を既定の7日のままにする | 誤削除から7日以内なら復元できる | 消したはずの学生証が7日残る。その間も保存料金がかかる |
| 公開アクセス防止を両方に付ける（採用） | 設定ミスで公開されない | なし（公開URLで見せる予定がない） |
| 学生証のバケットだけに付ける | issue の作業内容どおり | 商品写真が誤って公開されうる |
| 鍵ファイルで署名する | signBlob の呼び出しが要らない | 鍵の保管と入れ替えが必要になる |

## 影響

- 更新したドキュメント：[tech/storage.md](../tech/storage.md)（STORAGE：本番の設定）、[infra/README.md](../../infra/README.md)
- 実装への影響：Go のコードの変更はなし。`infra/config.sh`・`infra/setup.sh` にバケット・CORS・権限を追加
- Hosting のサイト名が `<プロジェクトID>` と異なるときは、#39 で `infra/config.sh` の `HOSTING_ORIGINS` を直す
