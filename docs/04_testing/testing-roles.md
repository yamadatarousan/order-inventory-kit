# テストの役割（レールの固定化）

## 目的
- テストが「何を固定しているか」を明示し、実装の自由度と固定条件を分離する
- Domain / UseCase / Handler の責務に対応させる

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

## Handler（境界前提の固定）
- 対象: `backend/tests/boundary/*_test.go`
- 固定するもの
  - HTTPの分類（200/400/404）の前提
  - 正常系の最小レスポンス形
  - 不正入力の扱い

## 運用ルール
- Domain/UseCase/Handler の境界を跨がない
- テストの追加・変更時は「どの固定条件を増やすか」を明示する
