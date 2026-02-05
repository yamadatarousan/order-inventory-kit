#!/usr/bin/env bash
set -euo pipefail

# BASE_SPEC / REV_SPEC は比較対象のOpenAPIパス
# 引数が無い場合はgitの参照から自動解決する
BASE_SPEC="${1:-}"
REV_SPEC="${2:-}"

resolve_base_spec() {
  local ref="${1:-origin/main}"
  local tmp
  # gitの指定リビジョンから contracts/openapi.yaml を取得する
  tmp="$(mktemp /tmp/openapi_base.XXXXXX.yaml)"
  if git show "${ref}:contracts/openapi.yaml" > "${tmp}" 2>/dev/null; then
    echo "${tmp}"
    return 0
  fi
  rm -f "${tmp}"
  return 1
}

if [[ -z "${BASE_SPEC}" || -z "${REV_SPEC}" ]]; then
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    # 基準は origin/main を優先し、無ければ直前コミットを使う
    if BASE_SPEC="$(resolve_base_spec "origin/main")"; then
      REV_SPEC="contracts/openapi.yaml"
    elif BASE_SPEC="$(resolve_base_spec "HEAD~1")"; then
      REV_SPEC="contracts/openapi.yaml"
    else
      echo "[openapi-diff] no base spec found (origin/main or HEAD~1)."
      echo "[openapi-diff] pass BASE_SPEC and REV_SPEC or run after a base exists."
      exit 0
    fi
  else
    echo "[openapi-diff] not a git repository. provide BASE_SPEC and REV_SPEC."
    exit 1
  fi
fi

run_oasdiff() {
  local base="$1"
  local rev="$2"

  if command -v oasdiff >/dev/null 2>&1; then
    # oasdiffが入っていればそのまま使う
    oasdiff breaking --fail-on WARN "$base" "$rev"
    return 0
  fi

  if command -v docker >/dev/null 2>&1; then
    # oasdiffが無い場合はDockerイメージで実行する
    docker run --rm \
      -v "$PWD":/work \
      -v /tmp:/tmp \
      -w /work \
      tufin/oasdiff:latest \
      breaking --fail-on WARN "$base" "$rev"
    return 0
  fi

  # どちらも無い場合は失敗させる
  echo "[openapi-diff] oasdiff not found and docker not available."
  return 1
}

run_oasdiff "$BASE_SPEC" "$REV_SPEC"
