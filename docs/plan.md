# order-inventory-kit 開発プラン（注文・在庫ミニEC）

## 大前提
- AIは探索（生成）と固定（統合）を分離して扱う
- 固定化の可否は成果物（契約/不変条件/構造）とCIに従属させる
- 境界はOpenAPIを単一参照点として固定する
- 破壊的変更は移行定義（YAML）に従属させる
- 外形では検出できない互換性は境界テストで固定する

## 前提
- フロントエンド: TypeScript + React + Vite
- バックエンド: Go + Gin
- リポジトリ直下に `contracts/`, `backend/`, `frontend/`, `tools/` を配置
- 生成物はCIで再生成し、差分が出たら失敗させる
- 依存方向（Handler → UseCase → Domain）を構造検査で固定する

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

## 目的（検証したいこと）
- 境界差分 → 移行定義 → 生成整合 の固定化フローが機械的に機能する
- ドメイン不変条件が実装の変形を許容しつつ違反を落とせる
- 依存方向・越境ルールが探索空間の制約として働く
- 外形では検出できない互換性を境界テストで固定できる

---

## 全体構成（注文・在庫ミニEC）
- 注文作成 / 在庫確保 / 支払い確定 / 注文参照
- 注文キャンセル / 在庫戻し
- ドメイン不変条件例
  - 在庫は負にならない
  - 同一注文の二重確定は禁止
  - 支払いは一度だけ
- 境界前提テスト例
  - `POST /orders` が `accepted` を返したら `GET /orders/{id}` が `confirmed`
  - 存在しないIDは `404`、権限なしは `403`

---

## フェーズ別プラン

### Phase 1: レール（固定化条件）の骨格

#### 目標
固定化の関所（CI）に必要な成果物の配置と検証順序を確定する

#### 実装内容
- [ ] `contracts/openapi.yaml` の初期定義
- [ ] `contracts/migrations/` のテンプレ作成
- [ ] OpenAPI差分検査のスクリプト雛形
- [ ] 生成整合（Go/TSクライアント）のスクリプト雛形
- [ ] 構造検査ルール（依存方向/越境）の初期定義
- [ ] CIの最小パイプライン（差分→生成→構造→不変条件→境界前提）

#### 成果物
- `contracts/openapi.yaml`
- `contracts/migrations/template.yaml`
- `tools/openapi/` スクリプト
- `.github/workflows/ci.yml`

---

### Phase 2: バックエンド最小実装

#### 目標
GinでAPIの最小動作を作り、固定化条件の受け皿を用意する

#### 実装内容
- [ ] Handler / UseCase / Domain の最小レイヤ
- [ ] 注文作成 / 参照 / 確定 / キャンセル
- [ ] 仮のインメモリリポジトリ

#### 成果物
- `backend/cmd/api/`
- `backend/internal/{adapter,usecase,domain,infra}/`

---

### Phase 3: ドメイン不変条件の固定

#### 目標
「性質」をテストとして固定し、実装変更で崩れないことを保証する

#### 実装内容
- [ ] 在庫負数禁止
- [ ] 二重確定禁止
- [ ] 支払い二重計上禁止

#### 成果物
- `backend/tests/domain/` の不変条件テスト

---

### Phase 4: 境界前提の固定

#### 目標
外形では検出できない互換性を境界テストで固定する

#### 実装内容
- [ ] 200の意味（accepted → confirmed）を固定
- [ ] エラー分類（404/403/400）の固定
- [ ] 冪等性の固定（同一操作を2回）

#### 成果物
- `backend/tests/boundary/` の前提テスト

---

### Phase 5: フロントエンド最小実装

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

---

### Phase 6: 移行定義の実地検証

#### 目標
破壊的変更 → 移行定義 → CI通過のフローを検証する

#### 実装内容
- [ ] OpenAPIで破壊的変更を作る
- [ ] 移行定義を追加してCI通過
- [ ] 移行なしのときにCIが落ちることを確認

#### 成果物
- `contracts/migrations/` の実例
- CIログの確認

---

## 確認ポイント（固定化の関所）
- OpenAPI差分検査が破壊的変更を検出できる
- 破壊的変更が移行定義に従属している
- 生成整合が崩れたらCIが落ちる
- 依存方向・越境の違反がCIで落ちる
- 不変条件/境界前提の違反がCIで落ちる

---

最終更新: 2026-02-04
