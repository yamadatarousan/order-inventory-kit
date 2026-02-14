# 実装仕様（Spec）

## 目的
- この文書は、AIと人間が実装判断に使う「採用済み具体仕様（What must be true）」を定義する
- 方針説明は `docs/context.md`、方針の写像手順は `docs/context-mapping.md`、作業プロトコルは `AGENTS.md` を参照する

## スコープ
- 対象: 技術スタック、ディレクトリ構造、レイヤ責務、一次情報の参照順、変更時の必須条件
- 非対象: タスク順序や進行管理（`docs/plan.md`）、テスト運用詳細（`docs/testing-roles.md`）

## 一次情報の参照順
1. 契約: `contracts/openapi.yaml`
2. 実装計画: `docs/plan.md`
3. テスト運用規約: `docs/testing-roles.md`
4. 実装: `backend/`, `frontend/`, `tools/`

衝突時の扱い:
- API境界の事実は `contracts/openapi.yaml` を優先する
- タスクの完了/未完了は `docs/plan.md` を優先する
- テスト分類とDoDは `docs/testing-roles.md` を優先する

## 採用技術スタック（固定）
### Backend
- Go
- Gin
- PostgreSQL
- マイグレーション: SQLファイル（`backend/migrations/*.sql`）

### Frontend
- TypeScript
- React
- Vite
- OpenAPI生成型を利用（`frontend/src/api/schema.d.ts`）

### 契約/生成/検査
- OpenAPI: `contracts/openapi.yaml`（単一参照点）
- Go生成: `oapi-codegen`（生成物: `backend/internal/adapter/generated/openapi.gen.go`）
- TS生成: `openapi-typescript`（生成物: `frontend/src/api/schema.d.ts`）
- 差分/生成ツール: `tools/openapi/`
- 構造検査ツール: `tools/arch/`
- CI: `.github/workflows/ci.yml`

## ディレクトリ構造（固定）
```text
.
├── contracts/
│   ├── openapi.yaml
│   └── migrations/
├── backend/
│   ├── cmd/api/
│   ├── internal/
│   │   ├── adapter/{generated,handler}/
│   │   ├── usecase/
│   │   ├── domain/
│   │   └── infra/db/
│   ├── migrations/
│   └── tests/{domain,boundary}/
├── frontend/src/api/
├── tools/{openapi,arch,db}/
└── .github/workflows/ci.yml
```

## レイヤ責務（固定）
- `adapter/handler`: HTTP変換と分類
- `usecase`: アプリケーションフロー
- `domain`: 不変条件とドメイン性質
- `infra/db`: 永続化実装

依存方向:
- `handler -> usecase -> domain`
- `infra/db` は `domain` に依存してよい

## 在庫モデル方針（標準モデル）
- 決定記録: `docs/adr/0001-inventory-stock-model.md`（ADR-0001）
- 在庫は次の3値で定義する
  - `OnHand`: 実在庫
  - `Reserved`: 引当済み在庫
  - `Available`: 販売可能在庫（`OnHand - Reserved`）
- 売り越し防止の判定は `Available` を基準に行う
- `Reserved` は注文起点の引当管理に使い、`OnHand` は実在庫管理に使う
- 在庫数量に関する仕様判断・テスト観測・永続化設計は、この3値定義を単一の基準とする

## 在庫状態遷移（標準モデル）
- 決定記録: `docs/adr/0001-inventory-stock-model.md`（ADR-0001）
- この計画は当初「注文作成・決済確定・参照」の最小線を先に固める前提で切っている
- 最小線にしている理由は、契約/テスト/APIを先に閉じた範囲で確定し、未定義の出荷設計（出荷状態・部分出荷・返品・在庫減算タイミング）を混在させないため
- そのため現在の開発スコープに出荷機能（出荷確定API/UseCase/永続化）は含めない
- 注文作成時は `Reserve` を実行し、`Reserved` を増やす
- キャンセル/期限切れ時は `Release` を実行し、`Reserved` を減らす
- 決済確定時は在庫を減算しない（`OnHand` / `Reserved` を変更しない）
- 出荷機能が未実装の期間は、`出荷確定` を起点とする在庫更新を実行しない
- そのため未実装期間の在庫更新対象は `Reserved` のみとし、`OnHand` は変更しない
- `OnHand` 減算は、出荷機能を導入するフェーズで `出荷確定` とセットで実装する

## 決済 `amount` の意味と単位
- 決定記録: `docs/adr/0002-payment-amount-consistency.md`（ADR-0002）
- `amount` は「決済要求額」を表す
- 単位は最小通貨単位の整数で扱う（小数は扱わない）

## 通貨と丸め規則
- 決定記録: `docs/adr/0002-payment-amount-consistency.md`（ADR-0002）
- 通貨は単一通貨運用とし、`JPY` に固定する
- 丸め規則は「端数処理なし」を採用する
  - `amount` は整数（円）で扱い、四捨五入/切り上げ/切り捨ては行わない

## 価格の決定元
- 決定記録: `docs/adr/0002-payment-amount-consistency.md`（ADR-0002）
- `unit_price` はサーバ側価格情報を決定元として設定する
- クライアント入力の価格は決定元として採用しない

## テスト分類（固定）
- 不変条件テスト: `backend/tests/domain/`
- 境界一貫性統合テスト/境界単体テスト: `backend/tests/boundary/`
- `internal/*_test.go` は層ローカル仕様の検証を主目的とする

## 変更時の必須条件
- OpenAPI変更時:
  - 生成物を同期し、CIの生成整合チェックを通す
- 破壊的変更時:
  - `contracts/migrations/` に移行定義を追加する
- DBスキーマ変更時:
  - `backend/migrations/` を追加し、DB関連テストを更新する
- テスト追加/変更時:
  - `docs/testing-roles.md` の分類とDoDに従う

## 非目標（この文書で扱わない）
- フェーズの実行順、担当、進捗管理
- 実装手順の詳細
- 背景理論の説明
