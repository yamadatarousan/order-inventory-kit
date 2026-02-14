# docs 配置・命名規約

## 目的
- ドキュメントの配置と命名を統一し、参照先を迷わない状態を維持する

## 配置ルール
- ルート直下はプロジェクト入口文書のみを置く（`README.md`, `AGENTS.md`）
- 全体方針・仕様・計画・テスト方針は `docs/` に置く
- サブシステム固有の運用手順は対象ディレクトリ直下の `README.md` に置く
  - 例: `tools/arch/README.md`

## 命名ルール
- ディレクトリ案内は `README.md` を使う
- それ以外のドキュメントは `kebab-case.md` で命名する
- 連番プレフィックス（`01_`, `02_` など）は使わない

## 現在の全体ドキュメント
- `docs/context.md`
- `docs/context-mapping.md`
- `docs/spec.md`
- `docs/plan.md`
- `docs/testing-roles.md`
- `docs/adr/README.md`
- `docs/adr/0001-inventory-stock-model.md`
- `docs/adr/0002-payment-amount-consistency.md`
