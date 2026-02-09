#!/usr/bin/env bash
# 目的:
# Go/TS の依存方向・越境ルールを静的検査し、構造違反を統合前に落とす。
# 実施内容:
# backend は golangci-lint(depguard)、frontend は eslint(import/no-restricted-paths) を実行する。
# エラー時に即終了し、未定義変数とパイプ失敗を見逃さない。
set -euo pipefail

# どのカレントディレクトリから実行しても同じ挙動になるよう、常にリポジトリルートに移動する。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

# 実行対象のディレクトリ/設定ファイル。
# 既定は本体（backend/frontend）だが、自己テストでは環境変数でfixtureに差し替える。
GO_DIR="${ARCH_GO_DIR:-backend}"
GO_CONFIG="${ARCH_GO_CONFIG:-${GO_DIR}/.golangci.yml}"
TS_DIR="${ARCH_TS_DIR:-frontend}"
TS_CONFIG="${ARCH_TS_CONFIG:-${TS_DIR}/eslint.config.cjs}"

# コマンド引数で Go だけ / TS だけを実行できるようにするフラグ。
RUN_GO=1
RUN_TS=1

# 引数を解釈して実行対象を絞る。
for arg in "$@"; do
  case "${arg}" in
    --go-only)
      RUN_TS=0
      ;;
    --ts-only)
      RUN_GO=0
      ;;
    *)
      echo "[arch-check] unknown argument: ${arg}"
      echo "usage: $0 [--go-only|--ts-only]"
      exit 1
      ;;
  esac
done

# Go 側の構造検査を実行する。
# 優先順位:
# 1) ローカルにある golangci-lint
# 2) go install で /tmp に導入した golangci-lint
# 3) docker イメージの golangci-lint
# いずれかで実行できればその結果を返し、違反時は非0で失敗させる。
run_go_check() {
  local dir="$1"
  local config="$2"

  # 対象ディレクトリが無い場合は「未対象」として成功扱いでスキップする。
  if [[ ! -d "${dir}" ]]; then
    echo "[arch-check][go] skip: directory not found (${dir})"
    return 0
  fi
  # 設定ファイルが無い場合は、検査の前提不足なので失敗扱いにする。
  if [[ ! -f "${config}" ]]; then
    echo "[arch-check][go] config not found: ${config}"
    return 1
  fi

  # 最短経路: すでに導入済みの golangci-lint を使う。
  if command -v golangci-lint >/dev/null 2>&1; then
    (cd "${dir}" && golangci-lint run --config "$(basename "${config}")" ./...)
    return 0
  fi

  # go があれば、ワークスペース外(/tmp)に一時導入して実行する。
  # CI/ローカル双方で「導入済みかどうか」に依存しない実行を狙う。
  if command -v go >/dev/null 2>&1; then
    local cache_root="${TMPDIR:-/tmp}/order-inventory-kit-arch-check/go"
    local bin_dir="${cache_root}/bin"
    local bin_path="${bin_dir}/golangci-lint"
    mkdir -p "${cache_root}/gomod" "${cache_root}/gocache"

    if [[ ! -x "${bin_path}" ]]; then
      GOBIN="${bin_dir}" \
        GOMODCACHE="${cache_root}/gomod" \
        GOCACHE="${cache_root}/gocache" \
        # ネットワーク不可などで導入に失敗する可能性があるため、ここでは握りつぶして次フォールバックへ進む。
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 >/dev/null 2>&1 || true
    fi

    # 導入に成功していれば、そのバイナリを使って本検査を実行する。
    # この実行が非0なら「実際に違反がある」ので、そのまま失敗を返す。
    if [[ -x "${bin_path}" ]]; then
      (cd "${dir}" && \
        GOMODCACHE="${cache_root}/gomod" \
        GOCACHE="${cache_root}/gocache" \
        "${bin_path}" run --config "$(basename "${config}")" ./...)
      return $?
    fi
  fi

  # 最終フォールバック: docker で golangci-lint を実行する。
  if command -v docker >/dev/null 2>&1; then
    docker run --rm \
      -v "${ROOT_DIR}/${dir}:/work" \
      -w /work \
      golangci/golangci-lint:v1.64.8 \
      golangci-lint run --config "$(basename "${config}")" ./...
    return 0
  fi

  echo "[arch-check][go] golangci-lint を実行できません"
  return 1
}

# TypeScript 側の構造検査を実行する。
# 優先順位:
# 1) プロジェクトローカルの eslint
# 2) /tmp に導入した eslint 一式
# 3) docker 上の node + eslint
# import/no-restricted-paths の違反があれば非0で失敗させる。
run_ts_check() {
  local dir="$1"
  local config="$2"

  # 対象ディレクトリが無い場合はスキップ扱い。
  if [[ ! -d "${dir}" ]]; then
    echo "[arch-check][ts] skip: directory not found (${dir})"
    return 0
  fi
  # 設定ファイルが無い場合は前提不足として失敗。
  if [[ ! -f "${config}" ]]; then
    echo "[arch-check][ts] config not found: ${config}"
    return 1
  fi

  # 最短経路: frontend/node_modules 配下の eslint を使う。
  if [[ -x "${dir}/node_modules/.bin/eslint" ]]; then
    (cd "${dir}" && ./node_modules/.bin/eslint --config "$(basename "${config}")" --ext .ts,.tsx src)
    return 0
  fi

  # npm が使える場合は、/tmp に eslint と依存プラグインを一時導入して実行する。
  if command -v npm >/dev/null 2>&1; then
    local cache_root="${TMPDIR:-/tmp}/order-inventory-kit-arch-check/npm-deps"
    local bin_path="${cache_root}/node_modules/.bin/eslint"
    # npm --prefix の導入先には package.json が必要なため、最小定義を用意する。
    local cache_package_json="${cache_root}/package.json"
    mkdir -p "${cache_root}"
    if [[ ! -f "${cache_package_json}" ]]; then
      echo '{"name":"arch-check-cache","private":true}' > "${cache_package_json}"
    fi
    if [[ ! -x "${bin_path}" ]]; then
      echo "[arch-check][ts] install eslint deps into cache"
      # TS パースと import ルール実行に必要な最低限を固定バージョンで入れる。
      if ! NPM_CONFIG_CACHE="${cache_root}/cache" \
        npm install --no-save --prefix "${cache_root}" \
        eslint@9.22.0 eslint-plugin-import@2.31.0 @typescript-eslint/parser@8.26.0 typescript@5.7.3; then
        echo "[arch-check][ts] npm cache install failed, fallback to docker if available"
      fi
    fi

    # バイナリが用意できた場合のみ本検査を実行する。
    # 非0は違反としてそのまま返す。
    if [[ -x "${bin_path}" ]]; then
      (cd "${dir}" && \
        NODE_PATH="${cache_root}/node_modules" \
        "${bin_path}" \
          --config "$(basename "${config}")" --ext .ts,.tsx src)
      return $?
    fi
  fi

  # 最終フォールバック: docker 上で一時的に依存を入れて eslint 実行。
  # /src は読み取り専用でマウントし、作業はコンテナ内 /work で完結させる。
  if command -v docker >/dev/null 2>&1; then
    echo "[arch-check][ts] fallback to docker"
    if docker run --rm \
      -v "${ROOT_DIR}/${dir}:/src:ro" \
      node:22-alpine \
      sh -lc "mkdir -p /work && cp -R /src/. /work && cd /work && npm install --no-save eslint@9.22.0 eslint-plugin-import@2.31.0 @typescript-eslint/parser@8.26.0 typescript@5.7.3 && npx eslint --config $(basename "${config}") --ext .ts,.tsx src"; then
      return 0
    fi
  fi

  echo "[arch-check][ts] eslint を実行できません"
  return 1
}

# Go 検査フラグが有効なときだけ実行する。
if [[ "${RUN_GO}" -eq 1 ]]; then
  echo "[arch-check][go] start"
  run_go_check "${GO_DIR}" "${GO_CONFIG}"
  echo "[arch-check][go] ok"
fi

# TS 検査フラグが有効なときだけ実行する。
if [[ "${RUN_TS}" -eq 1 ]]; then
  echo "[arch-check][ts] start"
  run_ts_check "${TS_DIR}" "${TS_CONFIG}"
  echo "[arch-check][ts] ok"
fi

echo "[arch-check] all checks passed"
