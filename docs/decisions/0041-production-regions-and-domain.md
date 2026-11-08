# ADR 0041: 本番のリージョン・ドメイン・GCP プロジェクト

- 日付: 2026-11-01
- 状態: 採用
- 関連する章ID: ARCH, COST, DB, STORAGE

## 背景

M7 のデプロイ（#38・#39・#69）に着手するにあたり、[TECH-OPEN](../tech/open-questions.md) に残っていた次の項目を決める必要があった。

- Cloud Storage のリージョン（東京にするか、無料枠のある米国にするか）
- Turso のリージョン選択
- 独自ドメインを使うか（使わない場合は Firebase Hosting の標準ドメイン）

あわせて、プライバシーポリシー第６条の「保存する国」を埋める（[ADR 0036](0036-terms-and-privacy.md)）。

## 決定

- GCP プロジェクトは新しく `osagari-web-dev` を作り、既存の請求先アカウントに紐づける。Firebase も同じプロジェクトに追加する。
- Cloud Run は東京（asia-northeast1）、Turso は東京（aws-ap-northeast-1）に置く。
- Cloud Storage は商品写真・学生証のバケットとも米国（us-central1）に置く。（商品写真は [ADR 0051](0051-item-images-on-cloudflare-r2.md) で Cloudflare R2 に移した）
- 独自ドメインは使わず、Firebase Hosting の標準ドメイン（`<サイト名>.web.app`）で公開する。
- プライバシーポリシー第６条に、保存する国（日本・アメリカ合衆国）と、米国の制度については個人情報保護委員会の公表資料へのリンクを書く。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| Turso を東京に置く（採用） | Cloud Run（東京）からの往復が短く、API が速い | なし |
| Cloud Storage を米国に置く（採用） | 常時無料枠（5GB）に収まり、費用0円 | 画像の表示がやや遅い。学生証の画像も米国に置くことになり、ポリシーに外国を書く必要がある |
| Cloud Storage を東京に置く | 表示が速い。画像を国内に置ける | 少額ながら保存料金がかかる |
| 学生証だけ東京に置く | 機微な画像を国内に置ける | バケットごとに設定が分かれる。Google・Turso が米国の事業者である以上、ポリシーへの米国の記載は避けられない |
| 独自ドメインを使う | URL が覚えやすい | 費用と管理の手間がかかる |

## 影響

- 更新したドキュメント：[tech/open-questions.md](../tech/open-questions.md)、[tech/00-overview.md](../tech/00-overview.md)（ARCH・COST）、[tech/database.md](../tech/database.md)（DB）、[tech/storage.md](../tech/storage.md)（STORAGE）、[legal/privacy.md](../legal/privacy.md)（第６条）、[README.md](../README.md)
- 実装への影響：なし（インフラの設定は #38・#39・#69 で行う）
