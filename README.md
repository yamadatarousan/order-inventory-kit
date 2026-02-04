# order-inventory-kit

 ## 注文・在庫ミニEC

  - 境界: 注文作成/在庫確保/決済のAPI契約が明確
  - 不変条件: 在庫は負にならない、同一注文は二重確定しない、支払いは一度だけ
  - 構造: Handler→UseCase→Domainの依存方向がはっきり
  - 境界前提テスト例: POST /orders が accepted を返したら即座に GET /orders/{id} が confirmed になっている等