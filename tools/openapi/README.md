# OpenAPIチェックの配置と運用

## 目的
- この文書は、CIで自動判定するOpenAPIチェック（差分検査・生成整合・移行定義）の配置場所と運用手順を定義する
- 契約変更時に「どの手順で整合を確認するか」を迷わない状態を維持する

## 対象範囲
- 対象は `contracts/openapi.yaml` を基準にした差分検査と生成整合の運用
- 実装ロジックやUI仕様の詳細は対象外（`backend/`, `frontend/`, `docs/` で扱う）

## 方針（確定）
- OpenAPIチェックの実行入口は `tools/openapi/` 配下のスクリプトに集約する
- 契約の単一参照点は `contracts/openapi.yaml` とする
- 差分検査は `tools/openapi/diff.sh`、生成整合は `tools/openapi/generate.sh` で判定する
- CIは `.github/workflows/ci.yml` から上記スクリプトを実行する

## 配置
- 差分検査: `tools/openapi/diff.sh`
- 差分自己テスト: `tools/openapi/diff_test.sh`
- 生成整合: `tools/openapi/generate.sh`
- 生成自己テスト: `tools/openapi/generate_test.sh`
- 比較用fixture: `contracts/fixtures/*.yaml`
- 移行定義: `contracts/migrations/*.yaml`

## 運用
- 差分検査: `./tools/openapi/diff.sh`
- 生成更新: `./tools/openapi/generate.sh --write`
- 生成整合チェック: `./tools/openapi/generate.sh --check`
- 差分自己テスト: `./tools/openapi/diff_test.sh`
- 生成自己テスト: `./tools/openapi/generate_test.sh`

## 変更時のルール
- `contracts/openapi.yaml` を変更した場合は `./tools/openapi/generate.sh --write` を実行し、生成物差分を反映する
- 破壊的変更を含む場合は `contracts/migrations/` に移行定義を追加する
- 変更後は `./tools/openapi/diff_test.sh` と `./tools/openapi/generate_test.sh` を実行して整合を確認する
