#!/usr/bin/env bash
# 目的:
# 生成整合レールの自己テストを行い、必要な生成物が揃うことを確認する。
# 実施内容:
# generate.sh を実行した後、Go 生成物・TS schema・openapi-fetch クライアントの存在を検証する。
set -euo pipefail

GO_OUT="backend/internal/adapter/generated/openapi.gen.go"
TS_SCHEMA_OUT="frontend/src/api/schema.d.ts"
TS_CLIENT_OUT="frontend/src/api/client.ts"

# 生成を実行する
./tools/openapi/generate.sh --write

# Go生成物の存在を確認する
if [[ ! -s "${GO_OUT}" ]]; then
  echo "[openapi-generate-test] missing generated Go file: ${GO_OUT}"
  exit 1
fi

# TS型生成物の存在を確認する
if [[ ! -s "${TS_SCHEMA_OUT}" ]]; then
  echo "[openapi-generate-test] missing generated TS schema: ${TS_SCHEMA_OUT}"
  exit 1
fi

# openapi-fetch クライアントの存在を確認する
if [[ ! -s "${TS_CLIENT_OUT}" ]]; then
  echo "[openapi-generate-test] missing TS client file: ${TS_CLIENT_OUT}"
  exit 1
fi

if ! rg -q "openapi-fetch" "${TS_CLIENT_OUT}"; then
  echo "[openapi-generate-test] TS client does not use openapi-fetch: ${TS_CLIENT_OUT}"
  exit 1
fi

echo "[openapi-generate-test] generated files are present"
