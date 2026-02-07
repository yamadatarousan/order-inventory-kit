#!/usr/bin/env bash
set -euo pipefail

BASE="contracts/fixtures/openapi_base.yaml"
BREAKING="contracts/fixtures/openapi_breaking.yaml"
NONBREAKING="contracts/fixtures/openapi_nonbreaking.yaml"

failures=0

# breaking: 非0終了が期待される
if ./tools/openapi/diff.sh "$BASE" "$BREAKING"; then
  echo "[rail-self-test] expected breaking diff to fail, but it succeeded"
  failures=$((failures + 1))
else
  echo "[rail-self-test] breaking diff failed as expected"
fi

# non-breaking: 0終了が期待される
if ./tools/openapi/diff.sh "$BASE" "$NONBREAKING"; then
  echo "[rail-self-test] non-breaking diff succeeded as expected"
else
  echo "[rail-self-test] expected non-breaking diff to succeed, but it failed"
  failures=$((failures + 1))
fi

if [[ $failures -gt 0 ]]; then
  exit 1
fi
