#!/usr/bin/env bash
# 目的:
# 構造検査スクリプトが「正常ケースは通す」「違反ケースは落とす」を満たすことを自己検証する。
# 実施内容:
# fixture を差し替えて check.sh を複数回実行し、各ケースの期待終了コードを判定する。
# 自己テスト用スクリプト。
# tools/arch/check.sh が「正常ケースは通す」「違反ケースは落とす」を満たすか検証する。
set -euo pipefail

# 失敗ケース数を集計し、最後にまとめて終了コードへ反映する。
failures=0

# 1ケース分の実行と期待値判定を行う。
# name: ログファイル名と表示名
# expect: pass/fail の期待値
# go_dir/ts_dir: check.sh に渡す対象ディレクトリ（fixture）
run_case() {
  local name="$1"
  local expect="$2"
  local go_dir="$3"
  local ts_dir="$4"

  # check.sh の終了コードを検証したいので、一時的に -e を外して終了コードを取得する。
  set +e
  # 標準出力/標準エラーをケース別ログへ退避して、失敗時だけ内容を表示する。
  ARCH_GO_DIR="${go_dir}" ARCH_TS_DIR="${ts_dir}" ./tools/arch/check.sh >/tmp/arch-check-${name}.log 2>&1
  local status=$?
  set -e

  # 期待: pass なのに失敗した場合。
  if [[ "${expect}" == "pass" && ${status} -ne 0 ]]; then
    echo "[arch-self-test] ${name}: expected pass, but failed"
    cat "/tmp/arch-check-${name}.log"
    failures=$((failures + 1))
    return
  fi

  # 期待: fail なのに成功した場合。
  if [[ "${expect}" == "fail" && ${status} -eq 0 ]]; then
    echo "[arch-self-test] ${name}: expected fail, but passed"
    cat "/tmp/arch-check-${name}.log"
    failures=$((failures + 1))
    return
  fi

  echo "[arch-self-test] ${name}: ${expect} as expected"
}

# 正常系: Go/TS とも依存方向違反なし。
run_case "ok" "pass" "tools/arch/fixtures/go_ok" "tools/arch/fixtures/ts_ok"
# Go 違反系: domain -> usecase の越境 import を含む fixture。
run_case "go_violation" "fail" "tools/arch/fixtures/go_violation" "tools/arch/fixtures/ts_ok"
# TS 違反系: domain -> features の越境 import を含む fixture。
run_case "ts_violation" "fail" "tools/arch/fixtures/go_ok" "tools/arch/fixtures/ts_violation"

# 1件でも期待値とズレたケースがあれば失敗にする。
if [[ ${failures} -gt 0 ]]; then
  exit 1
fi
