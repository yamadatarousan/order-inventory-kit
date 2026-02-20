# ADR-0001: 在庫モデルを OnHand/Reserved/Available で扱う

- Status: Accepted
- Date: 2026-02-14

## Context
- 単一 `quantity` モデルでは、引当と実在庫の意味が混在し、在庫更新タイミングの判断がぶれる
- 注文作成/キャンセル/決済/出荷の各操作で、どの在庫値を変更すべきかを明確にする必要がある

## Decision
- 在庫は次の3値で扱う
  - `OnHand`: 実在庫
  - `Reserved`: 引当済み在庫
  - `Available`: 販売可能在庫（`OnHand - Reserved`）
- 売り越し防止は `Available` を基準に判定する
- 状態遷移は次を採用する
  - 注文作成: `Reserve`（`Reserved` を増やす）
  - キャンセル/期限切れ: `Release`（`Reserved` を減らす）
  - 決済確定: 在庫を減算しない
  - 出荷確定: `OnHand` を減算し、対応する `Reserved` を減算する
- 出荷未実装期間は `OnHand` を変更しない
- 注文作成と在庫引当の整合は「補償処理」を採用する
  - 方針: `CreateOrder` では `Reserve` を先に実行し、その後 `orders.Create` を実行する
  - `Reserve` 途中失敗、または `orders.Create` 失敗時は、先行して確保した明細を逆順で `Release` して補償する
  - 補償が失敗した場合は `compensation failed` として失敗を返し、再実行/運用対応の対象として扱う
- 既存データ移行は次で固定する
  - `on_hand = 旧 quantity`
  - `reserved = 0`

## Consequences
- DBは `inventory.quantity` 依存を廃止し、`inventory.on_hand` / `inventory.reserved` を導入する
- 既存 `inventory` レコードは `on_hand=旧quantity` / `reserved=0` でバックフィルする
- Repository/Domainの契約は `Reserve` / `Release` と `on_hand/reserved/available` 観測へ更新する
- 不変条件テストと境界一貫性統合テストの期待値を標準モデルへ更新する

## Related
- `docs/plan.md` の「Phase 5 完了後に着手する拡張タスク（EC在庫・決済金額整合）」
