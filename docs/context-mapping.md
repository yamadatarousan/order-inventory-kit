# 抽象から具体への写像ルール

## 目的
- `docs/context.md` の原則を、実装タスク・成果物・CI判定へ変換する
- 「抽象の正しさ」ではなく「統合可否の判定可能性」を優先する
- フェーズ着手時に、必要な成果物とDoDを先に確定する

## 用語同期（この文書で使う語）
- 不変条件テスト:
  - `backend/tests/domain/` で固定する性質テスト
- 境界一貫性統合テスト:
  - 実Router + 実UseCase + 実DB を通し、同一条件で同一観測を固定するテスト
- 境界単体テスト:
  - Stub前提でHTTP変換/分類のみを固定するテスト
- 主要レスポンス項目:
  - 仕様判断に使う中核フィールド（例: `status`, `customerId`）

## 抽象→具体マップ（固定）
| 抽象（context） | 具体成果物 | CI/検証 |
| --- | --- | --- |
| 境界を単一参照点で固定 | `contracts/openapi.yaml` | OpenAPI diff / generate 整合 |
| 破壊的変更は移行に従属 | `contracts/migrations/*.yaml` | diffで破壊検知時に移行定義必須 |
| 生成整合を保つ | `backend/internal/adapter/generated/*`, `frontend/src/api/schema.d.ts` | 再生成差分ゼロ |
| 構造違反を統合前に落とす | `tools/arch/*`, lint設定 | 構造検査で失敗 |
| 不変条件を固定する | `backend/tests/domain/*_test.go` | `go test ./tests/domain/...` |
| 外形外の互換性を固定する | `backend/tests/boundary/*_integration_test.go` | `go test ./tests/boundary -run Integration` |

## 実装開始前チェック（毎タスク）
1. 対象タスクの固定条件を1-3個に限定する
2. 各固定条件に対応する成果物パスを決める
3. 各成果物のCI判定コマンドを決める
4. DoD（最小完了判定）とDoD外タスクを分離する

## 完了時チェック（毎タスク）
1. 追加した固定条件を列挙する
2. 追加/変更した成果物を列挙する
3. 実行した検証コマンドと結果を示す
4. 未対応（DoD外）を `docs/plan.md` の未完了タスクとして残す

## Phase 5 写像ルール（必読）
### 前提
- `docs/plan.md` の Phase 5 タスク
- `docs/testing-roles.md` の Phase 5 最小DoD / 前提テンプレ

### 最小DoD（Phase 5）
- `backend/tests/boundary/*_integration_test.go` が1本以上ある
- `TestIntegration_` 命名テストが存在する
- 次を1シナリオ以上で検証する
  - HTTPステータス
  - 主要レスポンス項目（`status`, `customerId` など）
  - 後続API状態
  - 副作用DB（orders/payments/inventory）
- CIで次を必須化する
  - `go test ./tests/boundary/...`
  - `go test ./tests/boundary -run Integration`（0件は失敗）

### DoD外（例）
- 404/400の追加シナリオ拡張
- 冪等性シナリオのバリエーション追加
- 副作用観測対象の拡張（全テーブル厳密化）

DoD外項目は削除せず、`docs/plan.md` 未完了タスクに追加して管理する。

## 代表的な失敗パターンと補正
- 症状: 境界テストがStubだけで統合未検証
  - 補正: `*_unit_test.go` と `*_integration_test.go` を分離し、IntegrationをCI必須化
- 症状: 契約にある主要項目が境界テストで未観測
  - 補正: 主要レスポンス項目を観測項目として明記し、同値検証を追加
- 症状: DoDが広がりすぎて完了判定が不明
  - 補正: 最小DoDに限定し、残件は未完了タスクへ分離

## 文書間の役割
- Why: `docs/context.md`
- How-to-decide: `docs/context-mapping.md`（本書）
- What must be true: `docs/spec.md`
- When/進捗: `docs/plan.md`
- How-to-test: `docs/testing-roles.md`
- Agent behavior: `AGENTS.md`
