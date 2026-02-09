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
- テスト: go test / Vitest（不変条件・境界前提の固定）
- 構造検査: 依存方向/越境の静的検査（探索空間の制約）

## ファイル命名/配置のルール
- OpenAPIは `contracts/openapi.yaml` を単一参照点にする
- 破壊的変更の移行定義は `contracts/migrations/*.yaml`
- 境界テストは `backend/tests/boundary/`
- 不変条件テストは `backend/tests/domain/`
- 構造検査ルールは `backend/tools/arch-rules/` と `tools/arch/`

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
- 境界前提テストを通過したものだけ統合
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
- 境界前提テスト例
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
- [x] `backend/tests/boundary/` に最小境界前提テストを先行作成（順序補正）
- [ ] CIで不変条件テストを実行（placeholder削除）
- [ ] CIで境界前提テストを実行（placeholder削除）
- [ ] CIで差分→生成→構造→不変条件→境界前提の通し順を確認（上記ツール前提）
- [ ] 構造検査ルールの配置と運用方法を確定（tools/arch or lint設定）

#### 成果物
- `contracts/openapi.yaml`
- `contracts/migrations/template.yaml`
- `tools/openapi/` の実装済みスクリプト
- `tools/arch/` の実装済みスクリプト
- `.github/workflows/ci.yml`（実運用の関所）

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界前提:

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
- 境界前提: 変更なし

---

### Phase 2: バックエンド最小実装（TDD）

#### 目標
GinでAPIの最小動作を作り、固定化条件の受け皿を用意する

#### 実装内容
- [x] 仕様テスト（Domainの最小仕様）から着手
- [x] Handler / UseCase / Domain の最小レイヤ
- [x] 注文作成 / 参照 / 確定 / キャンセル
- [x] 仮のインメモリリポジトリ
- [ ] 起動エントリの作成（`backend/cmd/api/main.go`）

#### 成果物
- `backend/internal/{adapter,usecase,domain,infra}/`
- `backend/cmd/api/main.go`

#### セルフチェック
- 契約: 変更なし
- 差分: 変更なし
- 生成: 変更なし
- 構造: 依存方向は維持（Handler→UseCase→Domain）
- 不変条件: Domain仕様テストで固定
- 境界前提: Handlerの境界テストで固定

---

### Phase 3: 永続化の実装（DBリポジトリ）

#### 目標
インメモリ実装をDB実装に置き換え、永続化を成立させる

#### 実装内容
- [x] マイグレーションの作成（orders / order_items / payments / inventory）
- [x] DBリポジトリ実装（注文/決済/在庫）
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
- 境界前提: 変更なし

---

### Phase 4: ドメイン不変条件の固定

#### 目標
「性質」をテストとして固定し、実装変更で崩れないことを保証する

#### 実装内容
- [ ] Inventoryドメイン（SKU在庫・数量>=0）を実装する
- [ ] 在庫確保ユースケースを実装する（不足時は失敗）
- [ ] 在庫戻しユースケースを実装する（キャンセル時）
- [ ] DB在庫リポジトリの在庫更新をトランザクション化する
- [ ] 在庫負数禁止を不変条件テストで固定する
- [ ] 二重確定禁止を不変条件テストで固定する
- [ ] 支払い二重計上禁止を不変条件テストで固定する
- [x] 不変条件テストは `backend/tests/domain/` に集約する方針で固定
- [ ] 既存のドメインテストを `backend/tests/domain/` に移動して整理
- [ ] `backend/internal/domain/order_test.go` を `backend/tests/domain/` に移動する
- [ ] `backend/internal/usecase/order_usecase_test.go` の不変条件ケースを `backend/tests/domain/` に移動する
- [x] CustomerID は Domain に持たせる方針で固定
- [ ] Order に CustomerID を保持し、NewOrder で必須化する
- [ ] CI接続は Phase 0 の完了条件に従属（ここでは不変条件テストの内容拡張に専念する）

#### 成果物
- `backend/tests/domain/` の不変条件テスト

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界前提:

---

### Phase 5: 境界前提の固定

#### 目標
外形では検出できない互換性を境界テストで固定する

#### 実装内容
- [ ] 200の意味（accepted → confirmed）を固定
- [ ] エラー分類（404/400）の固定
- [ ] 冪等性の固定（同一操作を2回）
- [x] 境界前提テストは `backend/tests/boundary/` に集約する方針で固定
- [ ] 既存の境界テストを `backend/tests/boundary/` に移動して整理
- [ ] `backend/internal/adapter/handler/order_handler_test.go` の境界前提ケースを `backend/tests/boundary/` に移動する
- [ ] 境界前提テストに観測結果（HTTPステータス/レスポンス/後続状態/副作用）を明記する
- [ ] `403` 分類は Phase 7（認可導入）で固定する
- [ ] CI接続は Phase 0 の完了条件に従属（ここでは境界前提テストの内容拡張に専念する）

#### 成果物
- `backend/tests/boundary/` の前提テスト

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界前提:

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
- 境界前提:

---

### Phase 7: 認可導入（403固定の前提）

#### 目標
`403` を境界の固定条件として扱える前提（認証/認可）を成立させる

#### 実装内容
- [ ] 認証方式と権限モデルを定義する（誰が何にアクセス可能か）
- [ ] Handler / UseCase に認可チェックを導入する
- [ ] `contracts/openapi.yaml` に `403` レスポンスを追加する
- [ ] 境界前提テストで `403` 分類を固定する

#### 成果物
- 認可仕様を反映した `contracts/openapi.yaml`
- `backend/tests/boundary/` の `403` 前提テスト

#### セルフチェック
- 契約:
- 差分:
- 生成:
- 構造:
- 不変条件:
- 境界前提:

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
- 境界前提:

---

## 確認ポイント（固定化の関所）
- OpenAPI差分検査が破壊的変更を検出できる
- 破壊的変更が移行定義に従属している
- 生成整合が崩れたらCIが落ちる
- 依存方向・越境の違反がCIで落ちる
- 不変条件/境界前提の違反がCIで落ちる

---

最終更新: 2026-02-09
