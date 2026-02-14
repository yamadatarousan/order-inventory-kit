# ADR-0002: 決済 `amount` の意味と整合判定

- Status: Accepted
- Date: 2026-02-14

## Context
- `POST /payments/confirm` の `amount` は存在するが、注文金額との整合ルールが未定義だと挙動がぶれる
- クライアント入力値のみを信頼すると、改ざんや整合崩れを検出できない

## Decision
- `amount` は「決済要求額」を表す（最小通貨単位の整数）
- 通貨は単一通貨運用とし、`JPY` に固定する
- 丸め規則は「端数処理なし」を採用する
  - `amount` は整数（円）で受け取り、四捨五入/切り上げ/切り捨ては行わない
- サーバは注文合計を自前で算出し、`amount` と照合する
  - 算出元はサーバ側の価格情報と `order_items.unit_price` のスナップショットを正とする
  - クライアント入力の価格は決定元にしない
- `POST /payments/confirm` の判定
  - `amount == サーバ算出の注文合計`: 成功
  - `amount != サーバ算出の注文合計`: `409`（金額不一致）
- 冪等再送（同一 `idempotencyKey`）
  - 同額再送: 成功（同一結果）
  - 異額再送: `409`

## Consequences
- OpenAPIに `409` 分類（不一致/異額再送）を追加する
- 永続化に `payments.amount` と `order_items.unit_price` を導入する
- 境界一貫性統合テストに `409` と冪等同額/異額の観測を追加する

## Related
- `docs/plan.md` の「Phase 5 完了後に着手する拡張タスク（EC在庫・決済金額整合）」
