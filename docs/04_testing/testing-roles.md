# テストの役割（レールの固定化）

## 目的
- テストが「何を固定しているか」を明示し、実装の自由度と固定条件を分離する
- Domain / UseCase / Handler の責務に対応させる

## 境界観測一貫性の定義
- ここでいう「一貫性」は、同じ入力条件に対して同じ観測結果になること
- 観測結果に含めるもの
  - HTTPステータス
  - レスポンス本文（分類と主要フィールド）
  - 後続APIで観測される状態（直前レスポンスだけでなく、続けて呼ぶAPIで確認できる状態）
    - 例: `POST /orders` が `accepted` を返した直後、`GET /orders/{id}` で `status=confirmed` を観測できる
    - 例: `POST /orders/{id}/cancel` の後、`GET /orders/{id}` で `status=canceled` を観測できる
  - 副作用（件数・金額・在庫など）
- この定義に基づき、壊れると困る代表シナリオを `backend/tests/boundary/` で固定する

## Domain（不変条件の固定）
- 対象: `backend/tests/domain/*_test.go`
- 固定するもの
  - 値の制約（SKU空禁止、数量>=1）
  - 同一SKUの重複禁止
  - 注文作成時の初期状態（accepted）

## UseCase（振る舞いの固定）
- 対象: `backend/internal/usecase/*_test.go`（必要に応じて `backend/tests/` に移行）
- 固定するもの
  - ユースケースの入出力と状態遷移
  - 依存（Repository）の呼び出しと結果反映
  - 冪等性（同一キーの再実行）

## Handler（境界観測一貫性の固定）
- 対象: `backend/tests/boundary/*_test.go`
- 固定するもの
  - HTTPの分類（200/400/404）の前提
  - 正常系の最小レスポンス形
  - 不正入力の扱い

## 運用ルール
- Domain/UseCase/Handler の境界を跨がない
- テストの追加・変更時は「どの固定条件を増やすか」を明示する
