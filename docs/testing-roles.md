# テスト運用規約（判定基準の一次情報）

## 目的
- テスト分類・観測項目・完了判定を固定し、実装時の判断ぶれを防ぐ
- `docs/plan.md` の Phase 4 / Phase 5 を実装可能な基準に落とし込む
- Phase 5 は代表ケース1本では完了にせず、網羅マトリクス完了で判定する

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
  - inventory は `on_hand` / `reserved` / `available`（`available = on_hand - reserved`）で観測する

主要レスポンス項目の例:
- `POST /orders`: `orderId`, `status`
- `GET /orders/{id}`: `id`, `customerId`, `status`, `items`
- `POST /payments/confirm`: `orderId`, `paymentStatus`

## Phase 5 着手DoD
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

## Phase 5 網羅DoD（完了判定）
- エンドポイント単位の網羅が完了していること
  - `POST /orders`: `200/400`
  - `GET /orders/{id}`: `200/404`
  - `POST /payments/confirm`: `200/400/404/409/冪等（同額/異額）`
- `400` は各エンドポイントで入力検証項目の無効組み合わせを全列挙していること
  - n項目なら `2^n - 1`（空集合除く）ケース以上
  - 3項目なら 7 ケース（1項目無効/2項目無効/3項目無効）
- 各ケースで4観測（HTTPステータス/主要レスポンス項目/後続API状態/副作用DB）を検証していること
- `docs/testing-roles.md` の網羅マトリクス行と `backend/tests/boundary/*_integration_test.go` のテスト関数が1対1で対応していること
- `404` 分類は「未存在」の意味固定として代表ケース1本を最低ラインとする
- 未対応ケースが1件でも残っている場合、Phase 5 を完了扱いにしないこと

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
  - ID:
    - testkitで固定ID生成関数を注入し、実行ごとの差分をなくす
    - 統合テストでは固定ID（例: `integration-order-1`）を同値検証する
  - 時刻:
    - 現在の注文APIレスポンスに時刻項目はないため、当面は対象外
    - 時刻項目を追加した場合は「完全一致」は行わず、形式（RFC3339）と因果順（作成 <= 更新）のみ検証する

## Phase 5 網羅マトリクス（初版）
- 記載先:
  - `docs/testing-roles.md` 内の本セクションを単一参照点として運用する
- 記載列:
  - ケースID / 対象API / 入力分類 / 期待HTTP / 4観測（主要項目・後続API状態・副作用DB） / 対応テスト関数 / 状態

| ケースID | 対象API | 入力分類 | 期待HTTP | 4観測（主要項目・後続API状態・副作用DB） | 対応テスト関数 | 状態 |
|---|---|---|---|---|---|---|
| P5-ORD-200-01 | POST /orders | 正常入力 | 200 | `orderId/status`、後続GETで同一ID観測、orders作成/`reserved`増/`available`減/`on_hand`不変 | `TestIntegration_OrderBoundary_注文作成から決済確定と注文参照まで通し検証` | 完了 |
| P5-ORD-200-02 | POST /orders | quantity境界値（1） | 200 | `status=accepted`、後続状態作成可、DB副作用あり | `TestIntegration_OrderBoundary_POST_orders_200_quantity境界値1は受理される` | 完了 |
| P5-ORD-400-01 | POST /orders | customerId無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_customerId無効` | 完了 |
| P5-ORD-400-02 | POST /orders | items[*].sku無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_sku無効` | 完了 |
| P5-ORD-400-03 | POST /orders | items[*].quantity無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_quantity無効` | 完了 |
| P5-ORD-400-04 | POST /orders | customerId+sku無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_customerIdとsku無効` | 完了 |
| P5-ORD-400-05 | POST /orders | customerId+quantity無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_customerIdとquantity無効` | 完了 |
| P5-ORD-400-06 | POST /orders | sku+quantity無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_skuとquantity無効` | 完了 |
| P5-ORD-400-07 | POST /orders | customerId+sku+quantity無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_全項目無効` | 完了 |
| P5-ORD-400-08 | POST /orders | items空 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_items空` | 完了 |
| P5-ORD-400-09 | POST /orders | 重複SKU | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_orders_400_重複SKU` | 完了 |
| P5-GET-200-01 | GET /orders/{id} | 既存ID | 200 | `id/customerId/status/items`、前段POSTとの同値、DB読取整合 | `TestIntegration_OrderBoundary_GET_orders_id_200_既存IDは主要項目を返す` | 完了 |
| P5-GET-404-01 | GET /orders/{id} | 未存在ID | 404 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_404の意味_未存在注文参照は404を返す` | 完了 |
| P5-GET-CID-01 | GET /orders/{id} | customerId同値観測 | 200 | POST入力の`customerId`とGET応答の`customerId`同値、後続状態整合、DB読取整合 | `TestIntegration_OrderBoundary_customerId同値観測_POSTとGETで一致する` | 完了 |
| P5-SFX-01 | POST /orders + POST /payments/confirm | 副作用DB観測（orders/payments/inventory） | 200 | orders状態（accepted→confirmed）、payments件数（0→1）、inventoryは作成時`reserved`増/`available`減・決済時不変をDBで検証 | `TestIntegration_OrderBoundary_副作用DB観測_orders_payments_inventoryを固定する` | 完了 |
| P5-PAY-200-01 | POST /payments/confirm | 正常入力 | 200 | `paymentStatus`、後続GETで`confirmed`、payments更新 | `TestIntegration_OrderBoundary_200の意味_acceptedからconfirmedへの遷移を固定する` | 完了 |
| P5-PAY-200-02 | POST /payments/confirm | 金額一致 | 200 | `paymentStatus`、後続GETで`confirmed`、payments更新、inventory不変 | `TestIntegration_OrderBoundary_POST_payments_confirm_200_金額一致` | 完了 |
| P5-PAY-400-01 | POST /payments/confirm | orderId無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_orderId無効` | 完了 |
| P5-PAY-400-02 | POST /payments/confirm | amount無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_amount無効` | 完了 |
| P5-PAY-400-03 | POST /payments/confirm | idempotencyKey無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_key無効` | 完了 |
| P5-PAY-400-04 | POST /payments/confirm | orderId+amount無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_orderIdとamount無効` | 完了 |
| P5-PAY-400-05 | POST /payments/confirm | orderId+idempotencyKey無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_orderIdとkey無効` | 完了 |
| P5-PAY-400-06 | POST /payments/confirm | amount+idempotencyKey無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_amountとkey無効` | 完了 |
| P5-PAY-400-07 | POST /payments/confirm | orderId+amount+idempotencyKey無効 | 400 | エラー応答、後続状態なし、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_400_全項目無効` | 完了 |
| P5-PAY-404-01 | POST /payments/confirm | 未存在orderId | 404 | エラー応答、注文状態不変、payments件数不変 | `TestIntegration_OrderBoundary_POST_payments_confirm_404_未存在orderId` | 完了 |
| P5-PAY-409-01 | POST /payments/confirm | 金額不一致 | 409 | エラー応答、注文状態不変（accepted）、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_409_金額不一致` | 完了 |
| P5-PAY-IDEMP-01 | POST /payments/confirm | 同一キー同額再送 | 200 | 応答同値、後続状態不変、payments二重計上なし | `TestIntegration_OrderBoundary_POST_payments_confirm_冪等_同一キー再送` | 完了 |
| P5-PAY-IDEMP-02 | POST /payments/confirm | 同一キー異額再送 | 409 | エラー応答、後続状態不変（confirmed）、DB副作用なし | `TestIntegration_OrderBoundary_POST_payments_confirm_冪等_異額再送は409` | 完了 |

- 完了判定ルール:
  - 状態が `未着手` または `実装中` の行が1つでもある間は Phase 5 を完了扱いにしない

## 対応表の維持手順（欠落ケース可視化）
- 目的:
  - 網羅マトリクスの `対応テスト関数` と実テスト実装の不一致を可視化し、欠落ケースを見逃さない
- 手順:
  - マトリクス上の関数名を抽出:
    - `rg -o 'TestIntegration_[^`]+' docs/testing-roles.md | sort -u`
  - 実装上の関数名を抽出:
    - `rg -o '^func (TestIntegration_[^(]+)' backend/tests/boundary/order_boundary_integration_test.go -r '$1' | sort -u`
  - `comm` で差分を確認し、片側だけに存在する関数を欠落ケースとして扱う
- 現在の可視化結果（2026-02-16）:
  - `MISSING_IN_TESTS`: なし
  - `MISSING_IN_MATRIX`: なし

## 400入力組み合わせ網羅ルール
- 対象:
  - `POST /orders`
  - `POST /payments/confirm`
- 方針:
  - 入力検証対象の各項目をビットとして扱い、無効化ビット集合を全列挙する
  - 1項目無効 / 2項目無効 / ... / 全項目無効 まで欠落なく作る
- `POST /orders`（3項目）の最小全列挙:
  - `customerId` 無効
  - `items[*].sku` 無効
  - `items[*].quantity` 無効
  - `customerId + items[*].sku` 無効
  - `customerId + items[*].quantity` 無効
  - `items[*].sku + items[*].quantity` 無効
  - `customerId + items[*].sku + items[*].quantity` 無効
- `POST /payments/confirm`（3項目）の最小全列挙:
  - `orderId` 無効
  - `amount` 無効
  - `idempotencyKey` 無効
  - `orderId + amount` 無効
  - `orderId + idempotencyKey` 無効
  - `amount + idempotencyKey` 無効
  - `orderId + amount + idempotencyKey` 無効

## 運用ルール
- テスト追加時は、冒頭コメントに「固定対象・対象範囲・根拠」を記載する
- 新規テストは first failing test を確認してから実装を修正する（Red→Green→Refactor）
- 完了報告では、DoD達成項目と未対応項目を分けて明示する
