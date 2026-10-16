# ADR 0012: libSQL のリモート接続をプールに残さない

- 日付: 2026-10-16
- 状態: 採用
- 関連する章ID: DB, ARCH

## 背景

#59：API が DB を一度使ったあと約10秒リクエストがないと、以後の DB アクセスがすべて 500 になり、API を再起動するまで戻らなかった。エラーは次のとおり。

```
Hrana: `api error: `status=400 Bad Request, body={"message":"The stream has expired due to inactivity","code":"STREAM_EXPIRED"}``
```

- sqld・Turso への接続（Hrana over HTTP）は、接続ごとにサーバー側の「ストリーム」を持つ。sqld は使われないまま約10秒たったストリームを捨てる（ローカルで測定。8秒では起きず、11秒で起きた）。
- go-libsql は接続の有効性確認（`driver.Validator`・`driver.SessionResetter`）を実装しておらず、ストリームが捨てられても `driver.ErrBadConn` を返さない。そのため `database/sql` のプールは壊れた接続を使い回し続ける。
- Cloud Run は少しアクセスのない時間をはさむだけで同じ状態になるため、利用者の少ないこのアプリでは本番でもほとんどの時間帯で起きるおそれがある。

## 決定

リモート接続（`http://`・`https://`・`libsql://`）では `sqlDB.SetMaxIdleConns(0)` とし、使い終わった接続をプールに残さない。クエリ・トランザクションごとに新しい接続（新しいストリーム）を使う。ローカルの SQLite ファイル（`file:`、テスト用）はストリームを持たないので今までどおりにする。

回帰テストとして、libSQL の統合テストに「約10秒より長くあけてから再び使えること」を足した（`server/internal/db/libsql_integration_test.go` の `TestLibSQLIdleStream`）。

## 理由と検討した選択肢

| 選択肢 | 良い点 | 悪い点 |
| --- | --- | --- |
| **`SetMaxIdleConns(0)`（採用）** | 1行で済み、捨てられたストリームを使う隙がない。修正前に失敗し、修正後に通ることをテストで確認した | 接続を使い回さないぶん、リクエストごとに新しいストリームを開く。Turso 本体での遅延の増え方は未確認 |
| `SetConnMaxIdleTime` を数秒にする | 接続を短い間は使い回せる | 期限で閉じるのはプールの掃除の周期しだいで、閉じる前に取り出される隙が残る |
| ドライバを包み、`STREAM_EXPIRED` を `driver.ErrBadConn` に変える | 使い回しを続けられる | ドライバの内部（cgo）のエラー文字列に頼る。実装とテストが大きい |
| 後継の `tursogo` へ移る | 公式に案内されている新しいドライバ | 移行が大きく、GORM・golang-migrate との組み合わせ（ADR 0002）を検証し直す必要がある |

## 影響

- 更新したドキュメント：[tech/database.md](../tech/database.md)（DB）、[ADR 0002](0002-dev-environment.md)（検証結果に追記）
- 実装への影響：`server/internal/db/db.go` の `Open`。本番の Turso での遅延は、デプロイ（M7）のときに確かめる。遅延が問題になったら、`tursogo` への移行を検討する。
