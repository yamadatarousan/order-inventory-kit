#!/usr/bin/env bash
# 目的:
# OpenAPI 差分を検査し、破壊的変更が移行定義なしで混入することを防ぐ。
# 実施内容:
# base/rev の契約を比較し、breaking 検出時は contracts/migrations の追加有無で可否を判定する。
set -euo pipefail

# BASE_SPEC / REV_SPEC は比較対象のOpenAPIパス
# 引数が無い場合はgitの参照から自動解決する
BASE_SPEC="${1:-}"
REV_SPEC="${2:-}"
BASE_REF=""

resolve_base_spec() {
  local ref="${1:-origin/main}"
  local tmp
  # gitの指定リビジョンから contracts/openapi.yaml を取得する
  tmp="$(mktemp /tmp/openapi_base.XXXXXX.yaml)"
  if git show "${ref}:contracts/openapi.yaml" > "${tmp}" 2>/dev/null; then
    BASE_REF="${ref}"
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
  local status

  if command -v oasdiff >/dev/null 2>&1; then
    # oasdiffが入っていればそのまま使う
    set +e
    oasdiff breaking --fail-on WARN "$base" "$rev"
    status=$?
    set -e
    return $status
  fi

  if command -v docker >/dev/null 2>&1; then
    # oasdiffが無い場合はDockerイメージで実行する
    set +e
    docker run --rm \
      -v "$PWD":/work \
      -v /tmp:/tmp \
      -w /work \
      tufin/oasdiff:latest \
      breaking --fail-on WARN "$base" "$rev"
    status=$?
    set -e
    return $status
  fi

  # どちらも無い場合は失敗させる
  echo "[openapi-diff] oasdiff not found and docker not available."
  return 1
}

run_oasdiff "$BASE_SPEC" "$REV_SPEC"
status=$?
if [[ $status -eq 0 ]]; then
  exit 0
fi

# 破壊的変更が検出された場合は移行定義の追加を要求する
if [[ -n "$BASE_REF" ]]; then
  if git diff --name-only "$BASE_REF" -- contracts/migrations | grep -E '\\.ya?ml$' >/dev/null; then
    echo "[openapi-diff] breaking detected, migration file found -> allow"
    exit 0
  fi
fi

echo "[openapi-diff] breaking detected and no migration file found"
exit 1
