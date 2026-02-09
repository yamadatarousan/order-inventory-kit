#!/usr/bin/env bash
# 途中失敗と未定義変数を即時検知し、パイプ失敗も伝播させる。
set -euo pipefail

# どこから実行してもリポジトリ基準で動くようにルートへ移動する。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

# OpenAPIの単一参照点と、生成物の出力先を固定する。
OPENAPI_SPEC="contracts/openapi.yaml"
GO_OUT="backend/internal/adapter/generated/openapi.gen.go"
TS_SCHEMA_OUT="frontend/src/api/schema.d.ts"
TS_CLIENT_OUT="frontend/src/api/client.ts"

# auto: CIでは整合チェック、ローカルでは生成書き込みをデフォルトにする。
MODE="${1:-auto}"
case "${MODE}" in
  auto|--write|--check) ;;
  *)
    echo "usage: $0 [--write|--check]"
    exit 1
    ;;
esac

# 引数未指定時の挙動を環境変数 CI で切り替える。
if [[ "${MODE}" == "auto" ]]; then
  if [[ "${CI:-}" == "true" || "${CI:-}" == "1" ]]; then
    MODE="--check"
  else
    MODE="--write"
  fi
fi

# 生成先ディレクトリが無い場合でも失敗しないように事前作成する。
mkdir -p "$(dirname "${GO_OUT}")"
mkdir -p "$(dirname "${TS_SCHEMA_OUT}")"

# Go生成物を作る。
# 優先順位:
# 1) 既存の oapi-codegen バイナリ
# 2) go install で一時導入したバイナリ
# どちらも使えなければ失敗にする。
generate_go() {
  if command -v oapi-codegen >/dev/null 2>&1; then
    oapi-codegen \
      --package generated \
      --generate types,gin \
      -o "${GO_OUT}" \
      "${OPENAPI_SPEC}"
    return 0
  fi

  if command -v go >/dev/null 2>&1; then
    # go install の成果物/キャッシュはワークスペースを汚さないよう /tmp 配下に置く。
    local cache_root="${TMPDIR:-/tmp}/order-inventory-kit-openapi-generate/go"
    local bin_dir="${cache_root}/bin"
    mkdir -p "${bin_dir}" "${cache_root}/gomod" "${cache_root}/gocache"
    GOBIN="${bin_dir}" \
      GOMODCACHE="${cache_root}/gomod" \
      GOCACHE="${cache_root}/gocache" \
      go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1
    "${bin_dir}/oapi-codegen" \
      --package generated \
      --generate types,gin \
      -o "${GO_OUT}" \
      "${OPENAPI_SPEC}"
    return 0
  fi

  echo "[openapi-generate] oapi-codegen not found"
  return 1
}

# TypeScriptのschema型を作る。
# 優先順位:
# 1) ローカルの openapi-typescript
# 2) npx
# 3) Docker上の node + npx
# 生成キャッシュは /tmp 配下を使う。
generate_ts_schema() {
  local cache_root="${TMPDIR:-/tmp}/order-inventory-kit-openapi-generate"
  mkdir -p "${cache_root}/npm"

  if command -v openapi-typescript >/dev/null 2>&1; then
    openapi-typescript "${OPENAPI_SPEC}" --output "${TS_SCHEMA_OUT}"
    return 0
  fi

  if command -v npx >/dev/null 2>&1; then
    if NPM_CONFIG_CACHE="${cache_root}/npm" \
      npx --yes openapi-typescript@7.12.0 "${OPENAPI_SPEC}" --output "${TS_SCHEMA_OUT}"; then
      return 0
    fi
  fi

  if command -v docker >/dev/null 2>&1; then
    docker run --rm \
      -v "${ROOT_DIR}:/work" \
      -w /work \
      node:22-alpine \
      sh -lc "npx --yes openapi-typescript@7.12.0 ${OPENAPI_SPEC} --output ${TS_SCHEMA_OUT}"
    return 0
  fi

  echo "[openapi-generate] openapi-typescript not found"
  return 1
}

# openapi-fetch 利用の薄い共通クライアントを生成する。
# 手書き差分を避けるため、毎回この内容で上書きする。
generate_ts_client() {
  cat > "${TS_CLIENT_OUT}" <<'EOF'
import createClient from "openapi-fetch";
import type { paths } from "./schema";

const baseUrl = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export const apiClient = createClient<paths>({ baseUrl });
EOF
}

# --check 時の整合判定。
# 既存追跡ファイルの差分と、未追跡の生成物をどちらも検査する。
check_generated_diff() {
  local changed=0

  if ! git diff --quiet -- "${GO_OUT}" "${TS_SCHEMA_OUT}" "${TS_CLIENT_OUT}"; then
    changed=1
  fi

  if [[ -n "$(git ls-files --others --exclude-standard -- "${GO_OUT}" "${TS_SCHEMA_OUT}" "${TS_CLIENT_OUT}")" ]]; then
    changed=1
  fi

  if [[ "${changed}" -eq 1 ]]; then
    # CIログで原因追跡しやすいよう、差分と未追跡一覧を出す。
    echo "[openapi-generate] generated files are out of date"
    git --no-pager diff -- "${GO_OUT}" "${TS_SCHEMA_OUT}" "${TS_CLIENT_OUT}" || true
    local untracked
    untracked="$(git ls-files --others --exclude-standard -- "${GO_OUT}" "${TS_SCHEMA_OUT}" "${TS_CLIENT_OUT}")"
    if [[ -n "${untracked}" ]]; then
      echo "[openapi-generate] untracked generated files:"
      echo "${untracked}"
    fi
    exit 1
  fi

  echo "[openapi-generate] generated files are up to date"
}

# 生成を実行する。どれか1つでも失敗したら set -e で停止する。
generate_go
generate_ts_schema
generate_ts_client

# --write は生成更新、--check は差分ゼロの検証に専念する。
if [[ "${MODE}" == "--check" ]]; then
  check_generated_diff
else
  echo "[openapi-generate] generated files updated"
fi
