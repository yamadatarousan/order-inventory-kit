# order-inventory-kit

## 注文・在庫ミニEC

探索（生成）と固定（統合）を分離し、契約/不変条件/構造のレールで統合可否を機械判定するための検証用ミニEC。

### 機能スコープ（最小）
- 注文作成（在庫確保まで）/ 注文参照 / 注文キャンセル
- 在庫確保 / 在庫戻し
- 支払い確定（冪等）

### 境界・不変条件・構造
- 境界: 注文作成/在庫確保/決済のAPI契約が明確
- 不変条件: 在庫は負にならない、同一注文は二重確定しない、支払いは一度だけ
- 構造: Handler → UseCase → Domain の依存方向がはっきり

### 境界前提テスト例
- `POST /orders` が `accepted` を返したら即座に `GET /orders/{id}` が `confirmed`
- 存在しないIDは `404`、権限なしは `403`

### DB (Docker / PostgreSQL)
- 起動: `docker compose up -d`
- 接続情報（開発）: `postgres://order_inventory:order_inventory@localhost:5434/order_inventory_dev`
- 接続情報（テスト）: `postgres://order_inventory:order_inventory@localhost:5434/order_inventory_test`

### テストの役割
- `docs/testing-roles.md` を参照
