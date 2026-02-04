# Webシステム スターターキット案（Frontend: TypeScript + React / Backend: Go + Gin）

作業ディレクトリ直下の想定（`/Users/user/Development/docs`）。

## 目的
- 生成（探索）と統合（固定化）を分離し、CIが統合可否を機械判定できる構成にする。
- 境界（契約）・ドメイン（不変条件）・構造（依存方向/越境）を成果物と検証で固定する。

---

## 技術スタック（最小）

### Frontend
- TypeScript
- React
- Vite
- React Router
- APIクライアント：OpenAPI生成（例：openapi-typescript + openapi-fetch）
- テスト：Vitest + Testing Library
- 型/品質：ESLint + Prettier
  
### Backend
- Go
- Gin
- OpenAPI生成：oapi-codegen
- DB（例）：PostgreSQL
- マイグレーション：goose もしくは sqlc + migrations
- テスト：go test（ユニット/統合）

### 契約・境界
- OpenAPI（単一参照点）
- 破壊的変更の移行定義（YAML）
- 差分検査：openapi-diff（例：oasdiff）

### 構造検査
- 依存方向・越境の静的検査
  - Go：go list + custom lint（または depguard / golangci-lint）
  - TS：eslintのimport/no-restricted-paths

### CI（固定化の関所）
- GitHub Actions
  - OpenAPI差分 → 破壊的変更なら移行定義必須
  - 型/クライアント生成の整合
  - 構造検査（依存方向・越境）
  - ドメイン不変条件テスト
  - 境界の前提テスト

---

## ディレクトリ構造（案）

```
.
├── contracts/
│   ├── openapi.yaml                # 境界の単一参照点
│   └── migrations/
│       └── 2026-02-04-...yaml       # 破壊的変更の移行定義
│
├── backend/
│   ├── cmd/
│   │   └── api/                     # エントリ
│   ├── internal/
│   │   ├── adapter/
│   │   │   ├── handler/             # Gin handlers (HTTP)
│   │   │   └── repo/                # DB実装
│   │   ├── usecase/                 # アプリケーション層
│   │   ├── domain/                  # エンティティ/値オブジェクト/不変条件
│   │   └── infra/                   # DB/外部サービス
│   ├── tests/
│   │   ├── boundary/                # 境界前提テスト（HTTP）
│   │   └── domain/                  # 不変条件テスト
│   └── tools/
│       └── arch-rules/              # 依存方向/越境の検査定義
│
├── frontend/
│   ├── src/
│   │   ├── app/                     # ルーティング/アプリ組み立て
│   │   ├── features/                # 画面・ユースケース単位
│   │   ├── domain/                  # UI側のドメインモデル（必要なら）
│   │   ├── api/                     # 生成されたAPIクライアント
│   │   └── shared/                  # UI共通
│   └── tests/
│       └── boundary/                # UI側の境界前提（必要なら）
│
├── tools/
│   ├── openapi/                     # 生成/差分検査スクリプト
│   └── arch/                        # 依存方向/越境の検査
│
├── .github/workflows/ci.yml         # 固定化の関所
├── README.md
└── AGENTS.md
```

---

## 固定化ルールの最小セット（CI順序の例）

1. OpenAPI差分検査
   - 破壊的変更があれば `contracts/migrations/*.yaml` の存在と形式を要求
2. クライアント/型生成の整合
3. 構造検査（依存方向・越境）
4. ドメイン不変条件テスト
5. 境界前提テスト（外形で検出できない互換性）

---

## 補足
- 生成物はCIで再生成し、差分が出たら失敗させる（固定化のため）。
- 依存方向ルールは「探索空間の制約」でもあり「固定化条件」でもある。

