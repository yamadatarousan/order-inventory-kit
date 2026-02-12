# order-inventory-kit 開発プラン（注文・在庫ミニEC）

## 大前提
- AIは探索（生成）と固定（統合）を分離して扱う
- 固定化の可否は成果物（契約/不変条件/構造）とCIに従属させる
- 境界はOpenAPIを単一参照点として固定する
- 破壊的変更は移行定義（YAML）に従属させる
- 外形では検出できない互換性は境界テストで固定する
- 探索空間の制約（探索の前提）と固定化の条件（探索結果の選別基準）を最優先で囲う

## 前提
- フロントエンド: TypeScript + React + Vite
- バックエンド: Go + Gin
- リポジトリ直下に `contracts/`, `backend/`, `frontend/`, `tools/` を配置
- 生成物はCIで再生成し、差分が出たら失敗させる
- 依存方向（Handler → UseCase → Domain）を構造検査で固定する
- 開発はTDD（Red→Green→Refactor）を基本とする

## 技術スタック（厳選）
- 契約: OpenAPI（`contracts/openapi.yaml` を単一参照点）
- 差分検査: OpenAPI差分ツール（破壊的変更の検出に必須）
- 生成: Go/TSの型・クライアント生成（契約との差分固定のため）
- バックエンド: Go + Gin（HTTP境界の実装と検証）
- フロントエンド: TypeScript + React + Vite（契約からの型利用と最小UI）
- テスト: go test / Vitest（不変条件・境界観測一貫性の固定）
- 構造検査: 依存方向/越境の静的検査（探索空間の制約）

## ファイル命名/配置のルール
- OpenAPIは `contracts/openapi.yaml` を単一参照点にする
- 破壊的変更の移行定義は `contracts/migrations/*.yaml`
- 境界テストは `backend/tests/boundary/`
- 不変条件テストは `backend/tests/domain/`
- 構造検査ルールは `backend/.golangci.yml` / `frontend/eslint.config.cjs` / `tools/arch/`

---

## 探索空間の制約（探索の前提）
- 境界の単一参照点は `contracts/openapi.yaml`
- 依存方向は `Handler → UseCase → Domain` の片方向
- 越境/禁止 import は構造検査で落とす
- 生成物は手書きしない（CIで再生成し差分ゼロを要求）
- 仕様はテストで先に固定し、実装は後から追従させる（TDD）

## 固定化の条件（探索結果の選別基準）
- OpenAPI差分で破壊的変更を検出したら移行定義が必須
- 生成整合が崩れた変更は統合しない
- 構造検査に違反した変更は統合しない
- ドメイン不変条件テストを通過したものだけ統合
- 境界観測一貫性テストを通過したものだけ統合
- 仕様テストが先に存在し、失敗から修正されていること

---

## TDD運用ルール（全フェーズ共通）
- Red: 仕様をテストで先に表現し、失敗することを確認
- Green: 最小限の実装でテストを通す
- Refactor: 依存方向/命名/重複を整理（テストは維持）
- 仕様テストは「性質」や「前提」を固定することを優先

---

## 目的（検証したいこと）
- 境界差分 → 移行定義 → 生成整合 の固定化フローが機械的に機能する
- ドメイン不変条件が実装の変形を許容しつつ違反を落とせる
- 依存方向・越境ルールが探索空間の制約として働く
- 外形では検出できない互換性を境界テストで固定できる

---

## 全体構成（注文・在庫ミニEC）
- 注文作成 / 在庫確保 / 支払い確定 / 注文参照
- 注文キャンセル / 在庫戻し
- 在庫はSKU単位で管理し、初期在庫は固定テーブルから供給する
- ドメイン不変条件例
  - 在庫は負にならない
  - 同一注文の二重確定は禁止
  - 支払いは一度だけ
- 境界観測一貫性テスト例
  - `POST /orders` が `accepted` を返したら `GET /orders/{id}` が `confirmed`
  - 存在しないIDは `404`、権限なしは `403`（`403` の固定は Phase 7）

---

## フェーズ別プラン
### Phase 0: レールの核（CI/差分/生成/構造）

#### 目標
固定化条件を機械的に運用できる状態にする

#### 方針（ツール選定）
- OpenAPI差分: oasdiff
- 生成（Go）: oapi-codegen
- 生成（TS）: openapi-typescript + openapi-fetch
- 構造検査（Go）: golangci-lint + depguard
- 構造検査（TS）: eslint import/no-restricted-paths

#### 実装内容
- [x] `contracts/openapi.yaml` の初期定義
- [x] `contracts/migrations/` のテンプレ作成
- [x] OpenAPI差分検査を oasdiff で実装
- [x] `tools/openapi/diff.sh` を oasdiff で実運用化する
- [x] レール自己テストを追加（fixtures + diff_test）
- [x] 破壊的変更 → 移行定義必須の判定を自動化
- [x] 生成整合（Go/TSクライアント）を oapi-codegen / openapi-typescript + openapi-fetch で実装
- [x] 依存方向/越境の構造検査ルールを golangci-lint + depguard / eslint import/no-restricted-paths で実装
- [x] CIで差分→生成→構造を実行（rails job）
- [x] `.github/workflows/ci.yml` の rails job に Postgres service を追加し、migrate適用後に domain/boundary テストを実行可能にする
- [x] `backend/tests/domain/` に最小不変条件テストを先行作成（順序補正）
- [x] `backend/tests/boundary/` に最小境界観測一貫性テストを先行作成（順序補正）
- [x] CIで不変条件テストを実行（placeholder削除）
- [x] CIで境界観測一貫性テストを実行（placeholder削除）
- [x] CIで差分→生成→構造→不変条件→境界観測一貫性の通し順を確認（上記ツール前提）
- [x] 構造検査ルールの配置と運用方法を確定（`tools/arch/README.md` / `backend/.golangci.yml` / `frontend/eslint.config.cjs`）
- [x] OpenAPIチェックの配置と運用方法を確定（`tools/openapi/README.md` / `tools/openapi/*.sh`）

#### 成果物
- `contracts/openapi.yaml`
- `contracts/migrations/template.yaml`
- `tools/openapi/` の実装済みスクリプト
- `tools/openapi/README.md`（OpenAPIチェックの配置と運用）
- `tools/arch/` の実装済みスクリプト
- `tools/arch/README.md`（構造検査ルールの配置と運用）
- `.github/workflows/ci.yml`（実運用の関所）

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界観測一貫性:

---

### Phase 1: 永続化の土台（DBセットアップ）

#### 目標
PostgreSQLをDockerで起動できる状態にし、テスト/実装の前提を整える

#### 実装内容
- [x] docker-compose で PostgreSQL を起動できるようにする
- [x] 開発/テスト用の接続情報を明記する
- [x] マイグレーションの置き場を決める（例: `backend/migrations/`）
- [x] マイグレーションツールを選定する（golang-migrate）
- [x] CIでDBを起動する方式を定義する（GitHub ActionsのPostgres service）

#### 成果物
- `docker-compose.yml`
- `backend/migrations/`（空でも可）
- `docs/` か `README.md` の接続情報
- 採用するマイグレーションツールの明記
- CIでDBを起動する方式の明記

#### CIでのDB起動方式（具体）
- GitHub ActionsのjobにPostgres serviceを追加
- `POSTGRES_DB=order_inventory_test` / `POSTGRES_USER=order_inventory` / `POSTGRES_PASSWORD=order_inventory`
- テストは `localhost:5432` で接続（CI内サービスのデフォルト）

#### セルフチェック
- 契約: 変更なし
- 差分: 変更なし
- 生成: 変更なし
- 構造: 変更なし
- 不変条件: 変更なし
- 境界観測一貫性: 変更なし

---

### Phase 2: バックエンド最小実装（TDD）

#### 目標
GinでAPIの最小動作を作り、固定化条件の受け皿を用意する

#### 実装内容
- [x] 仕様テスト（Domainの最小仕様）から着手
- [x] Handler / UseCase / Domain の最小レイヤ
- [x] 注文作成 / 参照 / 確定 / キャンセル
- [x] 仮のインメモリリポジトリ
- [x] 起動エントリの作成（`backend/cmd/api/main.go`）

#### 成果物
- `backend/internal/{adapter,usecase,domain,infra}/`
- `backend/cmd/api/main.go`

#### セルフチェック
- 契約: 変更なし
- 差分: 変更なし
- 生成: 変更なし
- 構造: 依存方向は維持（Handler→UseCase→Domain）
- 不変条件: Domain仕様テストで固定
- 境界観測一貫性: Handlerの境界テストで固定

---

### Phase 3: 永続化の実装（DBリポジトリ）

#### 目標
インメモリ実装をDB実装に置き換え、永続化を成立させる

#### 実装内容
- [x] マイグレーションの作成（orders / order_items / payments / inventory）
- [x] DBリポジトリ実装（注文/決済）
- [x] DB在庫リポジトリを実装する（`GetBySKU` / `Reserve` / `Release`）
- [x] InventoryRepository の更新責務を `Reserve` / `Release` に固定し、`Update` を廃止する
- [x] `inventory_usecase` を `repo.Reserve` / `repo.Release` 呼び出しに切り替える
- [x] `inventory_usecase_test` の `memoryInventoryRepo` を `Reserve` / `Release` 前提へ更新する
- [x] DB在庫リポジトリの在庫更新をトランザクション化する（同時実行制御を含む）
- [x] 在庫テーブルの初期データ投入を実装する（seed/init）
- [x] 在庫DBリポジトリの契約テストを追加する
- [x] DB接続設定（env/設定ファイル）

#### 成果物
- `backend/migrations/` の初期マイグレーション
- `backend/internal/infra/db/` の実装
- `backend/.env.example`（DB接続設定）

#### セルフチェック
- 契約: 変更なし
- 差分: 変更なし
- 生成: 変更なし
- 構造: 変更なし
- 不変条件: DBリポジトリの契約テストで固定
- 境界観測一貫性: 変更なし

---

### Phase 4: ドメイン不変条件の固定

#### 目標
「性質」をテストとして固定し、実装変更で崩れないことを保証する

#### 実装内容
- [x] Inventoryドメイン（SKU在庫・数量>=0）を実装する
- [x] 在庫確保ユースケースを実装する（不足時は失敗）
- [x] 在庫戻しユースケースを実装する（キャンセル時）
- [x] 在庫負数禁止を不変条件テストで固定する
- [x] 二重確定禁止（再確定時に状態・副作用が増えない）を不変条件テストで固定する
- [x] 支払い二重計上禁止を不変条件テストで固定する
- [x] 不変条件テストは `backend/tests/domain/` に集約する方針で固定
- [x] 既存のドメインテストを `backend/tests/domain/` に移動して整理
- [x] `backend/internal/domain/order_test.go` を `backend/tests/domain/` に移動する
- [x] `backend/internal/usecase/order_usecase_test.go` の不変条件ケースを `backend/tests/domain/` に移動する
- [x] `backend/internal/usecase/inventory_usecase_test.go` の不変条件ケースを `backend/tests/domain/` に移動する
- [x] `backend/internal/usecase/inventory_usecase_test.go` をユースケース局所仕様（入出力/呼び出し）に限定して整理する
- [x] CustomerID は Domain に持たせる方針で固定
- [x] Order に CustomerID を保持し、NewOrder で必須化する
- [x] Order.CustomerID の不変条件を境界/永続化へ反映する（DB列追加・Repository保存取得・OpenAPI反映）
- 注記: CI接続は Phase 0 の完了条件に従属（ここでは不変条件テストの内容拡張に専念する）

#### 成果物
- `backend/tests/domain/` の不変条件テスト

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界観測一貫性:

---

### Phase 5: 境界観測一貫性の固定

#### 目標
外形では検出できない互換性を、エンドポイントごとの網羅統合テストで固定する

#### 実装内容
- [x] 網羅マトリクスを先に定義する（対象API × 入力分類 × 期待HTTP × 観測項目）
- [ ] 網羅マトリクスに沿って `backend/tests/boundary/*_integration_test.go` のケース一覧を固定する
- [x] 統合境界テストの前提を定義する（対象API/観測項目/非対象を明文化）
- [x] 境界一貫性統合テストの着手DoDを明文化する（実装開始前の前提条件を定義する）
- [x] 「統合境界テストの前提を定義する」の成果物形式を固定する（記載先とテンプレ: `docs/testing-roles.md`）
- [x] `backend/tests/boundary` を二層化する（`*_unit_test.go` / `*_integration_test.go`）
- [x] 統合境界テスト用の testkit を追加する（実Router + 実UseCase + 実DB Repository を組み立て、Stub前提と分離する）
- [x] 統合境界テスト用の DB 初期化/後片付け手順を固定する（migrate適用、seed投入、テーブルリセット）
- [x] 非決定要素の扱いを固定する（ID/時刻などの注入または検証方法）
- [x] 境界一貫性統合テストを1本先行追加し、実Router+実UseCase+実DBで通し検証できることを固定する（`backend/tests/boundary/*_integration_test.go`）
- [x] 200の意味（accepted → confirmed）を統合境界テストで固定
- [x] エラー分類のうち 404（未存在）を統合境界テストで固定
- [ ] エラー分類のうち 400（不正入力）はエンドポイントごとに網羅固定する
- [ ] `POST /orders` の統合境界テストを網羅する（200/400、入力組み合わせの境界値を含む）
- [ ] `POST /orders` の 400 は入力3項目（`customerId` / `items[*].sku` / `items[*].quantity`）の無効組み合わせを全列挙で固定する（1項目無効/2項目無効/3項目無効）
- [ ] `GET /orders/{id}` の統合境界テストを網羅する（200/404）
- [ ] `POST /payments/confirm` の統合境界テストを網羅する（200/400/404/冪等）
- [ ] `POST /payments/confirm` の 400 は入力3項目（`orderId` / `amount` / `idempotencyKey`）の無効組み合わせを全列挙で固定する（1項目無効/2項目無効/3項目無効）
- [ ] 冪等性（同一操作を2回）を統合境界テストで固定
- [ ] `customerId` を主要フィールド観測として固定する（POST /orders 入力値と GET /orders/{id} 応答値の同値を検証する）
- [ ] 観測対象の副作用（orders/payments/inventory）を統合境界テストで固定する（orders状態・payments件数・inventory数量をDBで検証する）
- [ ] 各ケースで4観測を必須化する（HTTPステータス/主要レスポンス項目/後続API状態/副作用DB）
- [ ] テスト関数と網羅マトリクスの対応表を `docs/testing-roles.md` に維持する（欠落ケースを可視化する）
- [x] 境界観測一貫性テストは `backend/tests/boundary/` に集約する方針で固定
- [ ] 既存の境界テストを役割別に整理する（単体境界は `*_unit_test.go`、統合境界は `*_integration_test.go`）
- [ ] `backend/internal/adapter/handler/order_handler_test.go` を「Handler単体の責務（HTTP変換/分類）」に限定し、通しシナリオを残さない
- [ ] `backend/internal/adapter/handler/order_handler_test.go` から移した通しシナリオを `backend/tests/boundary/*_integration_test.go` で固定する
- [ ] 境界一貫性統合テストに観測結果を明記する（HTTPステータス/主要レスポンス項目/後続API状態/副作用DB）
- [ ] CIで boundary テスト全体（unit/integration）を実行しつつ、統合境界テスト（`go test ./tests/boundary -run Integration`）を必須化して rails 通過条件に含める（Integrationテスト0件を失敗扱いにする）
- [ ] `403` 分類は Phase 7（認可導入）で固定する
- 注記: CI接続は Phase 0 の完了条件に従属（ここでは境界観測一貫性テストの内容拡張に専念する）
- 運用ルール: Phase 5 の完了判定は「網羅マトリクスの完了」を必須とし、代表ケース1本だけでは完了にしない
- 運用ルール: 局所固定タスク（例: 代表ケース1本、単一分類固定）は、対象APIの網羅後続タスク（入力分類・完了条件付き）が `docs/plan.md` に未完了タスクとして明示されている場合にのみ着手可。未明示なら着手禁止

#### 完了条件（必須）
- エンドポイント単位の網羅が完了していること
  - `POST /orders`: `200/400`
  - `GET /orders/{id}`: `200/404`
  - `POST /payments/confirm`: `200/400/404/冪等`
- `400` は各エンドポイントで入力検証項目の無効組み合わせを全列挙していること
  - n項目なら `2^n - 1`（空集合除く）ケース以上
  - 3項目なら 7 ケース（1項目無効/2項目無効/3項目無効）
- 各ケースで4観測（HTTPステータス/主要レスポンス項目/後続API状態/副作用DB）を検証していること
- `docs/testing-roles.md` の網羅マトリクス行と `backend/tests/boundary/*_integration_test.go` のテスト関数が1対1で対応していること
- 未対応ケースが1件でも残っている場合、Phase 5 を完了扱いにしないこと

#### 成果物
- `backend/tests/boundary/` の境界観測一貫性テスト
- `docs/testing-roles.md` の「Phase 5 統合境界テスト前提テンプレ」（対象API/観測項目/非対象/データ準備/後片付け/非決定要素）
- `docs/testing-roles.md` の「Phase 5 着手DoD」（実装開始の前提項目）
- `docs/testing-roles.md` の「Phase 5 網羅マトリクス」（対象API別ケース一覧と対応するテスト関数）

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界観測一貫性:

---

### Phase 6: フロントエンド最小実装

#### 目標
OpenAPI生成クライアントを用いてUIから操作できる状態にする

#### 実装内容
- [ ] 注文作成フォーム
- [ ] 注文一覧/詳細
- [ ] 主要APIの呼び出し

#### 成果物
- `frontend/src/app/`
- `frontend/src/features/`
- `frontend/src/api/`（生成物）

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界観測一貫性:

---

### Phase 7: 認可導入（403固定の前提）

#### 目標
`403` を境界の固定条件として扱える前提（認証/認可）を成立させる

#### 実装内容
- [ ] 認証方式と権限モデルを定義する（誰が何にアクセス可能か）
- [ ] Handler / UseCase に認可チェックを導入する
- [ ] `contracts/openapi.yaml` に `403` レスポンスを追加する
- [ ] 境界観測一貫性テストで `403` 分類を固定する

#### 成果物
- 認可仕様を反映した `contracts/openapi.yaml`
- `backend/tests/boundary/` の `403` 境界観測一貫性テスト

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界観測一貫性:

---

### Phase 8: 移行定義の実地検証

#### 目標
破壊的変更 → 移行定義 → CI通過のフローを検証する

#### 実装内容
- [ ] OpenAPIで破壊的変更を作る
- [ ] 移行定義を追加してCI通過
- [ ] 移行なしのときにCIが落ちることを確認

#### 成果物
- `contracts/migrations/` の実例
- CIログの確認

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界観測一貫性:

---

## 確認ポイント（固定化の関所）
- OpenAPI差分検査が破壊的変更を検出できる
- 破壊的変更が移行定義に従属している
- 生成整合が崩れたらCIが落ちる
- 依存方向・越境の違反がCIで落ちる
- 不変条件/境界観測一貫性の違反がCIで落ちる

---

最終更新: 2026-02-09
