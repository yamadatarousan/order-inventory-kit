# ADR（Architecture Decision Record）

## 目的
- 設計上の意思決定（なぜそうするか、何を固定したか）を履歴として残す
- `docs/plan.md` のタスク実装時に、判断の根拠を参照できるようにする

## 運用ルール
- 1ファイル1決定で記録する
- ファイル名は `NNNN-<topic>.md`（例: `0001-inventory-stock-model.md`）
- ステータスは `Proposed` / `Accepted` / `Superseded` を使う
- 実装・契約・テストへ反映するタスクは `docs/plan.md` に残す

## ADR一覧
- `0001-inventory-stock-model.md`
- `0002-payment-amount-consistency.md`
