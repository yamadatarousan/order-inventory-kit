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
- DBマイグレーションで新規/変更したテーブル・カラムにコメントが無い変更は統合しない（CI検査）
- ドメイン不変条件テストを通過したものだけ統合
- 境界観測一貫性テストを通過したものだけ統合
- 仕様テストが先に存在し、失敗から修正されていること

---

## TDD運用ルール（全フェーズ共通）
- Red: 仕様をテストで先に表現し、失敗することを確認
- Green: 最小限の実装でテストを通す
- Refactor: 依存方向/命名/重複を整理（テストは維持）
- 仕様テストは「性質」や「前提」を固定することを優先
- 着手ゲート:
  - 仕様未確定の領域は、Red/Green の作業着手を禁止する
  - Red（先行テスト）が存在しない領域は、Green（実装）着手を禁止する
  - Green着手前に、対象Redテストが失敗することを確認する
  - 完了報告は Green 通過だけでなく、Refactor 後の再実行通過まで含める

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
- [x] DBマイグレーションのコメント検査をCIに追加（`tools/db/check_migration_comments.sh` / `.github/workflows/ci.yml`）

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
- 契約: `Order.CustomerID` を OpenAPI/DB/Repository に反映済み
- 差分: 本フェーズ内では破壊的変更の新規追加なし（移行定義の追加不要）
- 生成: 生成フローの追加変更なし（既存レールで整合維持）
- 構造: 依存方向（Handler→UseCase→Domain）違反なし
- 不変条件: `backend/tests/domain/` で対象性質を固定済み
- 境界観測一貫性: 本フェーズでは未対応（Phase 5で固定）

---

### Phase 5: 境界観測一貫性の固定

#### 目標
外形では検出できない互換性を、エンドポイントごとの網羅統合テストで固定する

#### 実装内容
- [x] 網羅マトリクスを先に定義する（対象API × 入力分類 × 期待HTTP × 観測項目）
- [x] 網羅マトリクスに沿って `backend/tests/boundary/*_integration_test.go` のケース一覧を固定する
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
- [x] エラー分類のうち 400（不正入力）はエンドポイントごとに網羅固定する
- [x] `POST /orders` の統合境界テストを網羅する（200/400、入力組み合わせの境界値を含む）
- [x] `POST /orders` の 400 は入力3項目（`customerId` / `items[*].sku` / `items[*].quantity`）の無効組み合わせを全列挙で固定する（1項目無効/2項目無効/3項目無効）
- [x] `GET /orders/{id}` の統合境界テストを網羅する（200/404）
- [x] `POST /payments/confirm` の統合境界テストを網羅する（200/400/404/冪等）
- [x] `POST /payments/confirm` の 400 は入力3項目（`orderId` / `amount` / `idempotencyKey`）の無効組み合わせを全列挙で固定する（1項目無効/2項目無効/3項目無効）
- [x] 冪等性（同一操作を2回）を統合境界テストで固定
- [x] `customerId` を主要フィールド観測として固定する（POST /orders 入力値と GET /orders/{id} 応答値の同値を検証する）
- [x] 観測対象の副作用（orders/payments/inventory）を統合境界テストで固定する（orders状態・payments件数・inventory数量をDBで検証する）
- [x] 各ケースで4観測をテストコード上で必須化する（1ケースにつき HTTPステータス/主要レスポンス項目/後続API状態/副作用DB のアサーションを最低1つずつ実装し、欠けるケースを未完了として残さない）
- [x] テスト関数と網羅マトリクスの対応表を `docs/testing-roles.md` に維持する（欠落ケースを可視化する）
- [x] 境界観測一貫性テストは `backend/tests/boundary/` に集約する方針で固定
- [x] 既存の境界テストを役割別に整理する（単体境界は `*_unit_test.go`、統合境界は `*_integration_test.go`）
- [x] `backend/internal/adapter/handler/order_handler_test.go` を「Handler単体の責務（HTTP変換/分類）」に限定し、通しシナリオを残さない
- [x] `backend/internal/adapter/handler/order_handler_test.go` から移した通しシナリオを `backend/tests/boundary/*_integration_test.go` で固定する
- [x] 境界一貫性統合テストに観測結果を明記する（HTTPステータス/主要レスポンス項目/後続API状態/副作用DB）
- [x] CIで boundary テスト全体（unit/integration）を実行しつつ、統合境界テスト（`go test ./tests/boundary -run Integration`）を必須化して rails 通過条件に含める（Integrationテスト0件を失敗扱いにする）
- 注記: `403` 分類は認可導入を前提とするため、Phase 7で固定する
- 注記: CI接続は Phase 0 の完了条件に従属（ここでは境界観測一貫性テストの内容拡張に専念する）
- 運用ルール: Phase 5 の完了判定は「網羅マトリクスの完了」を必須とし、代表ケース1本だけでは完了にしない
- 運用ルール: 局所固定タスク（例: 代表ケース1本、単一分類固定）は、対象APIの網羅後続タスク（入力分類・完了条件付き）が `docs/plan.md` に未完了タスクとして明示されている場合にのみ着手可。未明示なら着手禁止

#### Phase 5 完了後に着手する拡張タスク（EC在庫・決済金額整合）
- 決定参照: `ADR-0001` `docs/adr/0001-inventory-stock-model.md`（在庫モデル/状態遷移）
- 決定参照: `ADR-0002` `docs/adr/0002-payment-amount-consistency.md`（`amount` 整合/`409` 分類）
- [x] 目的1: 仕様を確定する（契約定義を含む）
  - [x] 在庫モデル方針を標準モデルで固定する（`OnHand` / `Reserved` / `Available=OnHand-Reserved`）
  - [x] 標準モデルの用語定義を明記する（`OnHand=実在庫`、`Reserved=引当済み在庫`、`Available=販売可能在庫`）
  - [x] 在庫状態遷移を固定する（注文作成で `Reserve`、キャンセル/期限切れで `Release`、決済確定では在庫を減算しない）
  - [x] 出荷未実装期間の運用前提を明記する（今回の対象外は「出荷確定でOnHand減算」。未実装期間は `OnHand` を変更しない）
  - [x] `amount` の意味と単位を固定する（`amount=決済要求額`、最小通貨単位の整数）
  - [x] 通貨と丸め規則を固定する（単一通貨運用、端数処理なし/ありの規則を明記）
  - [x] 価格の決定元を固定する（クライアント入力価格を信頼せず、サーバ側価格情報で `unit_price` を決定する）
  - [x] OpenAPI 契約を標準モデルに合わせて更新する（在庫表現と状態遷移の前提を反映）
  - [x] OpenAPI 契約に価格モデルを反映する（注文入力/応答の金額項目と金額算出前提を追加・更新）
  - [x] OpenAPI 契約に `POST /payments/confirm` の `409` 分類を追加する（金額不一致/冪等異額再送）
  - [x] 契約の破壊的変更に対する移行定義を追加する（`contracts/migrations/*.yaml`）

- [x] 目的2: Red（仕様テストを先に作成する）
  - [x] 不変条件テストを標準モデルへ更新する（負数禁止、過剰確保失敗、戻し時整合）
  - [x] 不変条件テストに在庫標準モデルの成功時整合を追加する（`Reserve` 成功で `reserved` 増/`available` 減、`Release` 成功で `reserved` 減/`available` 増、`on_hand` 不変）
  - [x] 不変条件テストに在庫操作の入力境界を追加する（`Reserve`/`Release` の `quantity<=0` は失敗）
  - [x] 不変条件テストに戻し境界を追加する（`Release` 後に `reserved` が負数になる戻しは失敗）
  - [x] 不変条件テストで在庫恒等式を固定する（各操作後に `available = on_hand - reserved` を検証する）
  - [x] 不変条件テストに生成時入力制約を追加する（`sku` 必須）
  - [x] 不変条件テストに金額整合を追加する（合計算出、一致時成功、不一致時失敗、冪等異額時失敗）
  - [x] 境界一貫性統合テストのDB副作用観測のあり方を最新の在庫モデル（標準在庫モデル）に合わせて更新する（orders/payments/inventory の観測基準を on_hand/reserved/available 基準へ移行する）
  - [x] 境界一貫性統合テストに金額系ケースを追加する（`200/400/404/409` と冪等同額/異額）
  - [x] `docs/testing-roles.md` の網羅マトリクスと観測項目を標準モデルへ同期する

- [ ] 目的3: Green（実装を追従させる）
  - [x] Green着手条件を満たす（目的1完了、目的2のRed失敗確認済み）
  - [x] 在庫スキーマ移行を追加する（`inventory.quantity` から `inventory.on_hand` / `inventory.reserved` へ移行）
  - [x] 既存データ移行方針を固定する（`on_hand=旧quantity`、`reserved=0` でバックフィル）
  - [x] seed/init データを標準モデルへ更新する（`on_hand` / `reserved` を投入する）
  - [x] 価格モデルを導入する（価格情報と `order_items.unit_price` を追加し、注文時に価格スナップショットを保存する）
  - [x] 決済永続化に `amount` を保存する（`payments.amount` 追加と既存データ移行方針を定義）
  - [x] 既存注文データの金額移行方針を固定する（`order_items.unit_price` 導入時のバックフィル/扱いを定義）
  - [x] Inventory ドメインを標準モデルへ更新する（単一 `Quantity` 依存を廃止し、`Reserve` / `Release` の不変条件を再定義）
  - [x] InventoryRepository 契約を標準モデルへ更新する（戻り値/永続状態の検証軸を `on_hand` / `reserved` / `available` に変更）
  - [x] 同時実行制御を標準モデル前提で再検証する（`Reserve` / `Release` の行ロックと競合時の不変条件）
  - [x] OrderUsecase と在庫の接続を更新する（`CreateOrder` で確保、`CancelOrder` で戻し、失敗時の副作用を残さない）
  - [x] 注文作成と在庫引当の整合方針を固定する（同一トランザクションまたは補償処理を定義し実装する）
  - [x] 在庫引当の追跡方式を確定する（`inventory_reservations` テーブルの要否を判定する）
  - [x] `inventory_reservations` テーブルを追加し、引当/戻しの参照整合を実装する（migration/seed/参照制約）
  - [x] キャンセル時の戻し条件を固定する（`accepted` / `confirmed` など注文状態ごとの `Release` 実行可否を明記）
  - [x] 期限切れ時の戻し方針を固定する（期限の定義、判定タイミング、`Release` 実行方式）
  - [x] `POST /payments/confirm` で金額照合を実装する（`amount != サーバ算出の注文合計` は `409`）
  - [x] 同一 `idempotencyKey` の再送は同額のみ成功とし、異額は `409` で失敗させる
  - [x] OpenAPI 生成物を再生成して整合させる（Go/TS）

- [x] 目的4: Refactor（設計整理）/運用で回帰確認（回帰テスト）を固定する
  - [x] Refactor着手前の再実行通過を確認する（Redで追加した全テストを再実行して通過）
  - [x] `ConfirmPayment` のエラー契約を型安全化する（`err.Error()` 文字列比較を廃止し、UseCase公開エラーで分類する）
  - [x] `confirmPayment` Handler のHTTP分類責務を整理する（エラー→HTTPステータス変換を関数化し分岐重複を削減する）
  - [x] 境界一貫性統合テストのエラー系共通検証を整理する（`400/404/409` の重複観測ロジックを共通化する）
  - [x] Refactor実施後の再実行通過を確認する（対象差分のテストと `go test ./... -p 1 -count=1` を通す）
  - [x] CIの通し順で標準モデル移行後の回帰確認（回帰テスト）を確認する（差分→生成→構造→不変条件→境界一貫性）

- [x] 補完目的: 価格決定元の運用主体を商品マスタに統合する（`product_prices` 暫定運用の解消）
  - [x] タスク1: 商品マスタの最小項目と価格決定ルールを確定し反映する（Red: 未存在SKU/販売停止SKUの失敗を不変条件・境界統合で先行追加 / Green: `products` 相当テーブル追加・seed投入・注文保存時の参照元切替を実装 / Refactor: 命名・責務整理と再実行通過）
  - [x] タスク2: 価格スナップショットの不変条件を固定する（Red: 商品マスタ価格変更後も既存注文明細 `unit_price` 不変の契約テストを先行追加 / Green: 保存・取得実装を追従 / Refactor: 重複クエリ整理と再実行通過）
  - [x] タスク3: `product_prices` 暫定運用を解消する（Red: 旧参照経路が使われると失敗する回帰テストを先行追加 / Green: データ移行と参照コード一本化を実装 / Refactor: 不要コード削除と運用手順更新）
  - [x] タスク4: 契約/生成物整合を回復する（Red: 生成差分検出を確認 / Green: OpenAPI更新とGo/TS生成物再生成 / Refactor: 生成フロー整理とCI再実行通過）

- [x] 補完目的: 顧客参照整合を `customers` マスタで固定する（`orders.customer_id` の孤立を防ぐ）
  - [x] タスク1: 顧客参照整合ルールを確定し反映する（Red: 未存在/無効 customerId の失敗を不変条件・境界統合で先行追加 / Green: `customers` テーブル追加・seed投入・注文作成時参照検証を実装 / Refactor: 参照責務整理と再実行通過）
  - [x] タスク2: `orders.customer_id` の永続整合を固定する（Red: 不整合時に注文が保存されない契約テストを先行追加 / Green: FKまたは参照検証の永続化実装 / Refactor: エラー分類整理と再実行通過）
  - [x] タスク3: 契約/生成物/運用手順を同期する（Red: 生成差分検出を確認 / Green: OpenAPIと生成物更新 / Refactor: 顧客データ運用手順の明文化とCI再実行通過）

- 注記: この拡張ブロックは Phase 5 の既存未完了タスク（境界整理・CI必須化）完了後に着手する

#### 完了条件（必須）
- エンドポイント単位の網羅が完了していること
  - `POST /orders`: `200/400`
  - `GET /orders/{id}`: `200/404`
  - `POST /payments/confirm`: `200/400/404/409/冪等（同額/異額）`
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
- 契約: `contracts/openapi.yaml` を単一参照点として維持し、`./tools/openapi/diff.sh` 通過
- 差分: 破壊的変更は `contracts/migrations/2026-02-16-price-model-breaking.yaml` で移行定義を追加済み
- 生成: `./tools/openapi/generate.sh --check` 通過（Go/TS 生成物整合あり）
- 構造: `./tools/arch/check.sh` 通過（依存方向/越境ルール違反なし）
- 不変条件: `cd backend && go test ./tests/domain/...` 通過
- 境界観測一貫性: `cd backend && go test ./tests/boundary/...` / `go test ./tests/boundary -run Integration -count=1` 通過、網羅マトリクスと統合テスト関数の対応差分なし

---

### Phase 6: フロントエンド最小実装

#### 目標
OpenAPI生成クライアントを用いてUIから操作できる状態にする

#### 事前分類（`docs/user-stories.md` 参照 / 現バックエンド実装基準）
- 着手可能（バックエンドAPI実装済み）:
  - `US-ORD-01` 価格同意つき注文作成（`POST /orders`）
  - `US-ORD-02` 注文参照（`GET /orders/{orderId}`）
  - `US-ORD-03` 注文キャンセル（`POST /orders/{orderId}/cancel`）
  - `US-PAY-01` 決済確定（`POST /payments/confirm`）
- 現状は着手不可（バックエンド未実装 or 受け入れ条件未充足）:
  - `US-PROD-01` `US-PROD-02` `US-PROD-03`（商品閲覧系API未実装）
  - `US-CART-01` `US-CART-02` `US-CART-03` `US-CART-04`（カートAPI未実装）
  - `US-CHK-03`（最終確認に必要な小計/送料/手数料/合計の取得API未実装）
  - `US-HIS-01`（顧客単位の履歴一覧API未実装）
  - `US-HIS-02`（履歴詳細の金額内訳/認可前提が未実装）

#### 実行方針（2レーン）
- 本実装レーン（完了判定対象）:
  - 対象: `US-ORD-01` `US-ORD-02` `US-ORD-03` `US-PAY-01`
  - 方針: 実バックエンドAPIに接続して実装する
- 仮実装レーン（完了判定対象外）:
  - 対象: `US-PROD-*` `US-CART-*` `US-CHK-03` `US-HIS-*`
  - 方針: フロントエンド完結のモック（fixture/MSW）で画面導線を先行実装する
- 仮実装ルール:
  - 仮実装であることをタスク名と画面内注記で明示する
  - 差し替え先APIと削除条件を `docs/plan.md` に明記する
  - Phase 6 の完了判定には含めない

#### 実装内容
- [ ] タスク1: 2レーン運用と完了判定を固定する（Red: 完了判定が曖昧な状態を確認 / Green: 本実装/仮実装の対象・除外・差し替え条件を明文化 / Refactor: 用語を統一して誤読を防ぐ）
- [ ] タスク2: 本実装レーンの注文作成を実装する（`US-ORD-01`）（Red: 入力/送信/`400/409` 表示テストを先行追加 / Green: `POST /orders` 接続と成功導線を実装 / Refactor: フォーム責務を整理して再実行通過）
- [ ] タスク3: 本実装レーンの注文参照・キャンセル・決済確定を実装する（`US-ORD-02` `US-ORD-03` `US-PAY-01`）（Red: 詳細表示/キャンセル/決済確定のUIテストを先行追加 / Green: `GET /orders/{id}` `POST /orders/{id}/cancel` `POST /payments/confirm` を接続 / Refactor: 状態管理とAPI呼び出し責務を整理して再実行通過）
- [ ] タスク4: 仮実装レーンの画面をモックで先行実装する（`US-PROD-*` `US-CART-*` `US-CHK-03` `US-HIS-*`）（Red: 受け入れ条件のUI観測テストを先行追加 / Green: fixture/MSWで画面導線を実装 / Refactor: モック生成と画面ロジックの重複を整理）
- [ ] タスク5: 仮実装レーンの差し替え契約を明記する（Red: 差し替え先が曖昧な状態を確認 / Green: 各画面ごとに差し替え先API・削除条件・移行先フェーズを `docs/plan.md` に記載 / Refactor: 依存順を整理）
- [ ] タスク6: 生成クライアント接続を固定する（Red: 生成差分検出を確認 / Green: `frontend/src/api/` 再生成と呼び出し更新 / Refactor: API層の集約とCI回帰確認）

#### 成果物
- `frontend/src/app/`
- `frontend/src/features/`
- `frontend/src/api/`（生成物）

#### 完了条件（必須）
- 本実装レーン（`US-ORD-01` `US-ORD-02` `US-ORD-03` `US-PAY-01`）が実API接続で動作する
- 仮実装レーンは「仮実装」であることと差し替え条件を明記済みである
- 仮実装レーンは完了判定に含めない

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
- [ ] タスク1: 認可仕様と `403` 契約を固定する（Red: `403` ケースの境界統合/Handler単体テストを先行追加 / Green: 認証方式・権限モデル・OpenAPI `403` 契約を実装 / Refactor: 分類規則整理と再実行通過）
- [ ] タスク2: 認可チェックを実装する（Red: 権限不足時の拒否テストを先行追加 / Green: Handler/UseCase に認可処理を接続 / Refactor: 共通ミドルウェア化と再実行通過）
- [ ] タスク3: 契約整合と回帰運用を固定する（Red: 生成差分検出を確認 / Green: OpenAPI生成物再生成と `docs/testing-roles.md` 同期 / Refactor: CI必須ケース整理と回帰確認）

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
- [ ] タスク1: 破壊的変更検出シナリオを固定する（Red: 破壊的変更を入れて移行定義なし失敗を確認 / Green: 判定基準と受け入れ基準を明文化 / Refactor: シナリオ重複を整理して再確認）
- [ ] タスク2: 移行定義適用で復旧する（Red: 失敗ログを証跡として保存 / Green: `contracts/migrations/` を追加してCI通過 / Refactor: 移行テンプレ整備と再実行通過）
- [ ] タスク3: 実地検証の再現性を固定する（Red: 手順抜けがあると再現不能になることを確認 / Green: 実行手順と証跡管理を文書化 / Refactor: 運用手順を最短化して再実行通過）

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

## 横断保留タスク（全フェーズ共通）
- [ ] 保留目的: EC運用拡張テーブルの要否を後続フェーズで確定する（現スコープ外だが未管理で放置しない）
  - [ ] タスク1: 配送テーブル群の要否を確定する（Red: 配送要件欠落を示すテスト観点整理 / Green: `shipments`/`shipment_items` 要否と導入時期を決定 / Refactor: 用語・依存関係整理）
  - [ ] タスク2: 住所テーブルの要否を確定する（Red: 住所要件欠落を示すテスト観点整理 / Green: `customer_addresses` 要否と導入時期を決定 / Refactor: 顧客参照整合との責務整理）
  - [ ] タスク3: 返金/返品テーブル群の要否を確定する（Red: 返金/返品要件欠落を示すテスト観点整理 / Green: `refunds`/`returns` 要否と導入時期を決定 / Refactor: 決済・在庫との整合整理）
  - [ ] タスク4: 税/割引テーブル群の要否を確定する（Red: 金額計算要件欠落を示すテスト観点整理 / Green: `tax_rates`/`coupons`/`promotions` 要否と導入時期を決定 / Refactor: 金額モデルとの境界整理）
  - [ ] タスク5: 履歴/監査テーブルの要否を確定する（Red: 監査追跡欠落を示すテスト観点整理 / Green: `order_status_history` またはイベントログの要否と導入時期を決定 / Refactor: 監査粒度の整理）
  - [ ] タスク6: 要導入テーブルを着手可能状態へ分解する（Red: 欠落契約ケースを先行列挙 / Green: 契約反映・migration・実装タスクへ分解 / Refactor: 優先順位を整理）
- [ ] 保留目的: テストDBスキーマ準備を migration に一本化する（`ensureSchema` との二重管理を解消する）
  - [ ] タスク1: migration一本化の置換計画を固定する（Red: `ensureSchema` 依存箇所を列挙して失敗再現 / Green: migration適用へ統一し `ensureSchema` 廃止 / Refactor: テスト初期化責務を整理し再実行通過）

---

## 確認ポイント（固定化の関所）
- OpenAPI差分検査が破壊的変更を検出できる
- 破壊的変更が移行定義に従属している
- 生成整合が崩れたらCIが落ちる
- 依存方向・越境の違反がCIで落ちる
- DBマイグレーションで新規/変更したテーブル・カラムにコメントが無いとCIで落ちる
- 不変条件/境界観測一貫性の違反がCIで落ちる

---
