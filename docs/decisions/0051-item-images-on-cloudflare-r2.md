# ADR 0051: 商品写真を Cloudflare R2 に移す

- 日付: 2026-11-08
- 状態: 採用
- 関連する章ID: STORAGE, ARCH, COST

## 背景

[ADR 0041](0041-production-regions-and-domain.md)・[ADR 0043](0043-production-cloud-storage-settings.md) で、商品写真と学生証はどちらも Cloud Storage（us-central1）に置いた。学生証は管理者しか見ないが、商品写真は利用者が一覧・詳細を開くたびに読まれ、そのたびに Cloud Storage の転送料（インターネットへの送信）と操作料がかかる。

API は応答のたびに有効期限 15分 の閲覧用署名付きURLを新しく作るので、同じ写真でも URL が毎回変わり、ブラウザのキャッシュが効かない。さらに署名付きURLは有効期限内なら何回でも読めるため、URL を手に入れた人がスクリプトで読み続けると、転送料が上限なく増える。予算アラート（[ADR 0046](0046-cloud-scheduler-and-budget-alert.md)）は通知するだけで、課金を止めない。

概算（1枚 約250KB）：

| 場面 | Cloud Storage | Cloudflare R2 |
| --- | --- | --- |
| 普段（月15万回・約38GB） | ほぼ0円（無料枠内） | 0円 |
| 利用が増えた（月300万回・約750GB） | 約1.2万円（転送 約650GB） | 0円（読み取りの無料枠 月1,000万回以内） |
| 1つの URL を1億回読まれた | 約45万円（約25TB の転送） | 約5,000円（無料枠を超えた読み取り 9,000万回） |

## 決定

- 商品写真は Cloudflare R2 に置く。学生証は Cloud Storage（us-central1）に残す。
- R2 のバケットには location hint にアジア太平洋（APAC）を付ける。保存する国は Cloudflare が決め、指定できない。プライバシーポリシー第６条にその旨を書く。
- バケットは公開しない（r2.dev のURL・独自ドメインを付けない）。今までどおり API が署名付きURLを発行し、ログインした利用者だけに見せる（STORAGE）。
- アップロード用の署名には `Content-Type`・`If-None-Match: *`（まだないオブジェクトだけ作れる）・`Content-Length` を含める。R2 には Cloud Storage の `x-goog-content-length-range` に当たる条件がなく、POST のポリシーにも対応していないため、ブラウザが縮小後の画像の大きさ（バイト）をアップロードURLの発行時に API に送り、API が 5MB 以下かを確かめてから、その値を `Content-Length` として署名する。登録時の確認（本人のものか・存在するか・形式・大きさ）は今までどおり行う。
- API は R2 の S3 互換 API を使う。署名には R2 のアクセスキーを使い、Secret Manager に保存する。アクセスキーの権限は商品写真のバケットの読み書きだけにする。
- ローカルでは、商品写真に S3 互換の MinIO を使う。学生証は今までどおり fake-gcs-server。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| 商品写真だけ R2 に移す（採用） | 転送料が0円で、読み取りの無料枠が大きい（月1,000万回）。悪用されても費用の増え方が小さい。S3 互換で SDK がそろっている | 事業者が増える。署名に長期のアクセスキーが要る。大きさの上限を署名で守るために API の契約を変える |
| 学生証も R2 に移す | 保存先が1つになる | 学生証は読まれる回数が少なく、費用の問題がない。鍵ファイルを持たない IAM signBlob の署名と、閲覧の記録・削除の仕組みが今のまま使える |
| Cloud Storage のまま、キャッシュが効くようにする | 事業者が増えない | 転送料の単価は変わらず、悪用への備えにならない |
| Cloud CDN を前に置く | 転送が速くなる | 有料。バケットを公開する必要があり、ログインした人だけに見せる方針と合わない |
| Backblaze B2 | 保存料が安い | 転送料が無料なのは保存量の3倍まで |
| Tigris | 転送料が0円・S3 互換 | 無料枠が小さく、サービスが新しい |
| Cloudinary・ImageKit | 縮小などの変換ができる | 無料の帯域を超えると止まるか有料プランが要る |
| location hint を北米東部にする | 保存する国を米国と書ける（今のポリシーのまま） | 佐賀からの表示が遅い |

- R2 は無料枠だけで使う場合もカードの登録が要る。費用の自動停止はないが、悪用されたときの増え方が Cloud Storage よりずっと小さい。

## 影響

- 更新したドキュメント：[tech/storage.md](../tech/storage.md)（STORAGE）、[tech/00-overview.md](../tech/00-overview.md)（ARCH・COST）、[legal/privacy.md](../legal/privacy.md)（第６条）、[ADR 0041](0041-production-regions-and-domain.md)・[ADR 0043](0043-production-cloud-storage-settings.md)（置き換えた箇所の注記）
- 実装への影響：
  - infra：R2 のバケット・CORS・アクセスキー（Secret Manager）、Cloud Run の環境変数、compose に MinIO、`infra/README.md`
  - API 契約：商品写真のアップロードURLの発行に画像の大きさ（バイト）を加える
  - server：`storage.Storage` の S3 互換の実装を足し、商品写真を扱う service に R2 の store を渡す。`server/CLAUDE.md` のストレージの行も直す
  - app：商品写真のアップロードURLの発行で、縮小後の画像の大きさを送る
  - 本番の商品写真を Cloud Storage から R2 にコピーし、切り替え後に Cloud Storage の商品写真のバケットを消す
