# テスト運用規約（判定基準の一次情報）

## 目的
- テスト分類・観測項目・完了判定を固定し、実装時の判断ぶれを防ぐ
- `docs/plan.md` の Phase 4 / Phase 5 を実装可能な基準に落とし込む

## 用語定義
- 不変条件テスト:
  - ドメインの性質（破ってはいけない状態）を固定するテスト
- 境界一貫性統合テスト:
  - 実Router + 実UseCase + 実DB を通し、同じ入力条件で同じ観測結果になることを固定するテスト
- 境界単体テスト:
  - Stub を使って HTTP 変換/分類の局所仕様を固定するテスト

## テスト配置ルール
- `backend/tests/domain/*_test.go`
  - 不変条件テストを配置する
- `backend/tests/boundary/*_unit_test.go`
  - 境界単体テスト（Stub可）を配置する
- `backend/tests/boundary/*_integration_test.go`
  - 境界一貫性統合テスト（実Router+実UseCase+実DB）を配置する
- `backend/internal/usecase/*_test.go`
  - ユースケース局所仕様（入出力・呼び出し）に限定する
- `backend/internal/adapter/handler/*_test.go`
  - Handler単体の責務（HTTP変換/分類）に限定する

## 観測項目（境界一貫性統合テスト）
- HTTPステータス
- 主要レスポンス項目
- 後続APIで観測される状態
- 副作用DB（orders/payments/inventory）

主要レスポンス項目の例:
- `POST /orders`: `orderId`, `status`
- `GET /orders/{id}`: `id`, `customerId`, `status`, `items`
- `POST /payments/confirm`: `orderId`, `paymentStatus`

## Phase 5 最小DoD
- `backend/tests/boundary/*_integration_test.go` が1本以上存在する
- `TestIntegration_` 命名の統合テストが存在する
- 1つ以上の統合テストで次を検証する
  - HTTPステータス
  - 主要レスポンス項目
  - 後続API状態
  - 副作用DB
- `customerId` の同値観測を検証する
  - `POST /orders` 入力値と `GET /orders/{id}` 応答値が一致する
- CIで統合テストを必須実行する
  - `go test ./tests/boundary -run Integration`
  - Integrationテスト0件は失敗扱いにする

## Phase 5 統合境界テスト前提テンプレ
- 対象API:
  - 例: `POST /orders`, `GET /orders/{id}`, `POST /payments/confirm`
- 観測項目:
  - HTTPステータス / 主要レスポンス項目 / 後続API状態 / 副作用DB
- 非対象:
  - 例: 認可（403）は Phase 7 で固定
- テストデータ準備:
  - 初期在庫、注文ID、`customerId`、idempotency key
- DB初期化・後片付け:
  - migrate適用済み前提、テーブルリセット手順
- 非決定要素の扱い:
  - ID/時刻は注入または検証対象から除外するルール

## 運用ルール
- テスト追加時は、冒頭コメントに「固定対象・対象範囲・根拠」を記載する
- 新規テストは first failing test を確認してから実装を修正する（Red→Green→Refactor）
- 完了報告では、DoD達成項目と未対応項目を分けて明示する
