# 構造検査ルールの配置と運用

## 方針（確定）
- 構造検査の実行入口は `tools/arch/check.sh` に集約する
- Go のルール定義は `backend/.golangci.yml`（depguard）に置く
- TypeScript のルール定義は `frontend/eslint.config.cjs`（import/no-restricted-paths）に置く
- CI は `.github/workflows/ci.yml` の `Architecture checks` で `./tools/arch/check.sh` を実行する

## 配置
- 実行スクリプト: `tools/arch/check.sh`
- 自己テスト: `tools/arch/check_test.sh`
- 検証用fixture: `tools/arch/fixtures/`
- Goルール: `backend/.golangci.yml`
- TSルール: `frontend/eslint.config.cjs`

## 運用
- ローカル実行（両方）: `./tools/arch/check.sh`
- Goのみ: `./tools/arch/check.sh --go-only`
- TSのみ: `./tools/arch/check.sh --ts-only`
- レール自己テスト: `./tools/arch/check_test.sh`

## 変更時のルール
- 依存方向・越境ルールの変更は、必ず対応する設定ファイルとfixtureを更新する
- ルール変更後は `./tools/arch/check_test.sh` と `./tools/arch/check.sh` の両方を実行して整合を確認する
