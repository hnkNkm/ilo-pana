# pana — 付加価値機能リサーチまとめ（curl便利版の次の一手）

> 対象: すでに「HTTPリクエストを送れる」基本機能ができている API ツール（CLI）  
> 目的: 次に差別化できる “付加価値機能” を整理し、実装ロードマップを作る

---

## 全体像：付加価値が出る領域（収束先）

curl互換の「送れる」段階の次に差が付く機能は、概ね以下に収束します。

1. **セッション/認証の自動化**
2. **コレクション＋テスト実行（シナリオ・アサーション）**
3. **仕様（OpenAPI）駆動（補完・検証・認証フロー）**
4. **スクリプト/プラグイン拡張**
5. **運用・性能・ガバナンス（SLO/しきい値、セキュアログ）**
6. **マルチプロトコル（gRPC/WS/GraphQL）**

---

## 付加価値機能候補（ROI順）

### 1) セッション（Cookie / ヘッダ / 認証の永続化）【最優先】
**ユーザー価値**
- ログイン後の一連のAPI操作が「毎回ヘッダ付け直し」から解放される
- dev/stg/prod を切り替えても “同じセッション” を維持しやすい

**MVP要件**
- Cookie jar（Set-Cookie を保持して次回以降に送る）
- セッションファイル（Cookie + 任意ヘッダ + 変数 + token）
- マスクルール（token/cookie 等のログ出力を安全に）

**CLI案**
- `pana --session ./dev.session.json send GET {{base_url}}/me`
- `pana session show`
- `pana session clear`

**実装メモ**
- `net/http` の `cookiejar` を利用
- セッション優先順位: `--var` > session > env > global

---

### 2) シナリオ実行 + capture + assert（APIテスト化）【最優先】
**ユーザー価値**
- 単発叩きツールから「回帰テスト/CI」で使える道具へ進化
- “期待値” をコード化でき、品質担保・チーム利用に強い

**MVP要件**
- 複数リクエストを順番に実行
- `capture`: レスポンスから値を抽出して変数へ（JSONPath/簡易パス/正規表現）
- `assert`: status / header / body の検証
- レポート出力（JUnit/JSON）

**CLI案**
- `pana test ./scenarios/login.pana`
- `pana test ./scenarios --junit out.xml --fail-fast`

**フォーマット案（例）**
```pana
# login
POST {{base_url}}/login
Content-Type: application/json

{"user":"{{user}}","pass":"{{pass}}"}

> capture $.token as token
> assert status == 200

# get profile
GET {{base_url}}/me
Authorization: Bearer {{token}}

> assert json $.id != null
```

---

### 3) 互換インポート / エクスポート（採用障壁を下げる）【高ROI】
**ユーザー価値**
- 既存資産（Postman/Insomnia/curl/HAR/OpenAPI）を取り込めると導入が一気に楽
- “移行コスト” を下げると継続利用されやすい

**優先順（実装しやすい順）**
1. `curl` import（よく使うフラグのみから開始）
2. HAR import/export（Web検証との親和性が高い）
3. Postman collection import（v2）
4. Insomnia export import

**CLI案**
- `pana import curl "curl ..."`
- `pana import postman collection.json`
- `pana export har --out out.har`
- `pana export curl <name>`

---

### 4) OpenAPI駆動（補完・バリデーション・OAuth自動化）【差別化】
**ユーザー価値**
- 仕様から “正しい入力” に誘導（補完/型/必須項目）
- 仕様違反の早期検出（破壊的変更、必須ヘッダ、schema違反）
- OAuth2 トークン取得/更新を設定だけで自動化

**MVP要件**
- OpenAPI を登録して、エンドポイント一覧/簡易補完
- 実行リクエストの schema validation（オプション）
- OAuth2（client credentials から）自動取得 → session に保存

**CLI案**
- `pana api add myapi ./openapi.yaml`
- `pana api ls myapi`
- `pana api call myapi GET /users/{id} --var id=1`
- `pana auth oauth2 client-credentials --env dev`

**実装メモ**
- OpenAPIパーサ（Go）で `paths` からテンプレ生成
- OAuthトークンは session/secrets に格納し、期限前更新（refresh）

---

### 5) pre/post（軽量スクリプト / 拡張ポイント）【中ROI】
**ユーザー価値**
- 署名生成、nonce/timestamp 付与など “現場の面倒” を吸収
- レスポンスから token を抜いて次へ、が自然にできる

**MVP要件**
- 組み込み関数だけで pre/post を表現（スクリプト言語は後回し）
- pre: uuid, timestamp, base64, hmac など
- post: JSON path 抽出、正規表現抽出

**CLI案**
- `pana run login --post 'set token=$.token'`
- `pana send ... --pre 'set ts=now()'`

**将来**
- プラグイン（AWS SigV4、GCP、Azure…）
- JS/Lua/Expr などのスクリプトエンジン

---

### 6) マルチプロトコル（GraphQL / WebSocket / gRPC）【拡張】
**ユーザー価値**
- APIツールとして守備範囲が広がる（特に gRPC）
- “1つの道具で済む” 体験

**MVP案**
- GraphQL: `POST /graphql` の sugar（query/mutation のテンプレ）
- WebSocket: connect/send/receive の最低限
- gRPC: proto読み込み → メソッド呼び出し（実装コストは高め）

**CLI案**
- `pana gql query ./q.graphql`
- `pana ws connect wss://...`
- `pana grpc call --proto ./a.proto pkg.Svc/Method`

---

### 7) 性能・SLO（しきい値・軽負荷テスト）【用途が刺さると強い】
**ユーザー価値**
- 「正しい」だけでなく「劣化してない」までCIで守れる
- p95 などの指標を “落とせる条件” にできる

**MVP要件**
- 簡易ベンチ: N回実行/並列度/時間、latency分位点、失敗率
- thresholds: `p95<200ms` のような条件

**CLI案**
- `pana bench <name> --vus 10 --duration 30s --threshold p95<200ms`

**補足**
- 本格負荷は k6 等と連携（panaがシナリオ→k6生成）でも良い

---

### 8) ガバナンス/セキュリティ（スキーマ検査・ログ安全化）【チーム導入向け】
**ユーザー価値**
- ルール（ヘッダ必須、命名規約、破壊的変更）をCIで担保
- シークレット漏洩を防ぐ（マスク・出力制御）

**MVP要件**
- OpenAPI lint（最低限のルールセット）
- secrets マスク（stdout/log/history 全て）
- “危険な出力” の防止（例: `--show-secrets` を明示しない限り非表示）

---

## pana向けおすすめ実装ロードマップ（迷わない順）

1. **Session（cookie jar + token保存 + マスク）**
2. **Scenario + capture + assert（= テストランナー化）**
3. **Import/Export（curl → pana、pana → curl/HAR）**
4. **OpenAPI連携（補完 + OAuth自動化）**
5. **pre/post（組み込み関数）→ プラグインへ**
6. （必要なら）**GraphQL/WS/gRPC**
7. （刺さるなら）**bench + thresholds**
8. **ガバナンス（lint/セキュアログ）**

---

## 実装の落とし穴（最初に設計しておくと得）
- **変数の優先順位**（CLI > session > env > global）を固定する
- **シークレットの扱い**を最初から設計（保存/表示/マスク/履歴）
- scenario 実行は「状態（変数）を持つ」ので、実行コンテキストを明確に（`RunContext`）
- import は完璧を目指さず、まず “よく使うオプションだけ” を堅牢に

---

## References（調査に用いた既存ツール/公式情報）
※ URLはそのまま貼らず、コードブロックにまとめています。

```text
HTTPie session / CLI docs
- https://httpie.io/docs/cli/environment-variables

Hurl manual (capture/assert の思想が近い)
- https://hurl.dev/docs/manual.html

Postman Newman（コレクションをCLI実行）
- https://learning.postman.com/docs/collections/using-newman-cli/command-line-integration-with-newman/

Insomnia import/export（Postman/HAR/OpenAPI/cURLなど）
- https://developer.konghq.com/insomnia/import-export/

HTTPie Desktop import（Postman/Insomnia取り込み）
- https://httpie.io/docs/desktop/postman-and-insomnia-import

Restish（OpenAPI駆動CLIの代表例）
- https://rest.sh/

Restish利用例（OAuthログイン/トークン保存）
- https://help.doit.com/docs/cli

httpYac scripting / multi-protocol（pre/post や拡張の参考）
- https://httpyac.github.io/guide/scripting.html
- https://github.com/AnWeber/httpyac

k6 thresholds（SLO/しきい値の考え方）
- https://grafana.com/docs/k6/latest/using-k6/thresholds/

Postman CLI（CI/ガバナンスの方向性）
- https://blog.postman.com/introducing-the-postman-cli-to-automate-your-api-testing/
```
