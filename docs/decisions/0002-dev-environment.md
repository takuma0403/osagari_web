# ADR 0002: 開発環境の構成と libSQL まわりのライブラリ選定

- 日付: 2026-10-14
- 状態: 採用
- 関連する章ID: DB, STORAGE, FE, FE-1, API-1, ARCH

## 背景

app/・server/ の雛形を作るにあたり、次を決める必要があった（issue #1）。

- 記載されていた Go 1.23・Node 20 は、2026-10 時点でどちらもサポートが終了している。
- ローカルで DB・ストレージ・メールをどう用意するか。
- [tech/open-questions.md](../tech/open-questions.md) の「GORM と golang-migrate が Turso（libSQL）で問題なく動くかの検証」。

## 決定

1. **バージョン**：Go 1.27、Node 24 LTS（リポジトリ直下の `.nvmrc`）。フロントは Vite 8・React 19・TypeScript 6、サーバーは Echo v5。
2. **ローカル構成**：`server/compose.yaml` で次を一括起動する。app は compose に入れず、`npm run dev` で直接動かし `/api` を Vite の proxy で `localhost:8080` に転送する。

   | 本番 | ローカル |
   | --- | --- |
   | Turso | libsql-server（sqld、Turso と同じ Hrana プロトコル） |
   | Cloud Storage | fake-gcs-server（`STORAGE_EMULATOR_HOST`） |
   | Gmail SMTP | Mailpit |

3. **libSQL のドライバ**：`github.com/tursodatabase/go-libsql`（cgo）。`libsql-client-go` は公式に非推奨になったため使わない。
4. **GORM の方言**：`github.com/ncruces/go-sqlite3/gormlite` の `OpenDB` に go-libsql の `*sql.DB` を渡す。
5. **golang-migrate**：`database/sqlite` ドライバを使い、マイグレーションの実行（Run）だけを差し替えて、SQL を文ごとに分けて1つのトランザクションで実行する（`server/internal/migration`）。

## 理由と検討した選択肢

GORM の方言：

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| `gorm.io/driver/sqlite` | 公式 | mattn/go-sqlite3（cgo）を読み込み、go-libsql と SQLite のシンボルが衝突してリンクできない |
| `github.com/glebarez/sqlite` | pure Go | golang-migrate が読み込む modernc.org/sqlite とドライバ名 `sqlite` が衝突し、起動時に panic |
| **gormlite（採用）** | `OpenDB` で任意の接続を渡せる。ドライバ名 `sqlite3` で衝突しない | WASM 版 SQLite を含むためバイナリが大きくなる（約 45MB） |

検証（`server/internal/db/libsql_integration_test.go`、sqld に対して実行）の結果：

- 通った：golang-migrate の up/down、GORM の作成・取得（時刻の往復を含む）、条件付きUPDATE の更新件数（並行10件で成功1件）、トランザクションのコミット/ロールバック、一意制約違反がエラーになること。
- **問題**：go-libsql のリモート接続は、1回の `Exec` に複数の文を渡すと **2文目以降をエラーなしで捨てる**。golang-migrate はファイル全体を1回で `Exec` するため、マイグレーションの一部が黙って適用されない。→ 決定5で対策し、テストで回帰を確認している。
- **制限**：一意制約違反は `gorm.ErrDuplicatedKey` に変換されない（エラー文字列は `UNIQUE constraint failed`）。必要になったら repository 層で判定する。
- 未検証：Turso クラウド本体（sqld と同じプロトコルのため同様に動く想定）。

## 影響

- 更新したドキュメント：[tech/database.md](../tech/database.md)（DB）、[tech/open-questions.md](../tech/open-questions.md)、[server/CLAUDE.md](../../server/CLAUDE.md)、[app/CLAUDE.md](../../app/CLAUDE.md)、ルートの README.md・CLAUDE.md
- 実装への影響：go-libsql が cgo のため、本番イメージは `CGO_ENABLED=1` でビルドし glibc 入りのベース（distroless/cc）を使う。
